package main

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	tcg "github.com/open-source-firmware/go-tcg-storage/pkg/core"
	"github.com/open-source-firmware/go-tcg-storage/pkg/drive"
	"github.com/open-source-firmware/go-tcg-storage/pkg/locking"
	"github.com/u-root/u-root/pkg/libinit"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/ulog"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

var (
	Version = "(devel)"
	GitHash = "(no hash)"
)

func main() {
	fmt.Printf("\n")
	l, _ := base64.StdEncoding.DecodeString(logo)
	fmt.Println(string(l))
	fmt.Printf("Welcome to Elastx PBA interactive!\nSource: %s\nGit Info: %s\n\n", Version, GitHash)
	log.SetPrefix("elx-pba: ")

	if _, err := mount.Mount("proc", "/proc", "proc", "", 0); err != nil {
		log.Fatalf("Mount(proc): %v", err)
	}
	if _, err := mount.Mount("sysfs", "/sys", "sysfs", "", 0); err != nil {
		log.Fatalf("Mount(sysfs): %v", err)
	}
	if _, err := mount.Mount("efivarfs", "/sys/firmware/efi/efivars", "efivarfs", "", 0); err != nil {
		log.Fatalf("Mount(efivars): %v", err)
	}

	log.Printf("Starting system...")

	if err := ulog.KernelLog.SetConsoleLogLevel(ulog.KLogNotice); err != nil {
		log.Printf("Could not set log level KLogNotice: %v", err)
	}

	libinit.SetEnv()
	libinit.CreateRootfs()
	libinit.NetInit()

	defer func() {
		log.Printf("Starting emergency shell...")
		for {
			Execute("/bbin/elvish")
		}
	}()

	sysblk, err := ioutil.ReadDir("/sys/class/block/")
	if err != nil {
		log.Printf("Failed to enumerate block devices: %v", err)
		return
	}

	startEmergencyShell := true
	password := ""
	for _, fi := range sysblk {
		devname := fi.Name()
		if _, err := os.Stat(filepath.Join("sys/class/block", devname, "device")); os.IsNotExist(err) {
			continue
		}
		devpath := filepath.Join("/dev", devname)
		if _, err := os.Stat(devpath); os.IsNotExist(err) {
			majmin, err := ioutil.ReadFile(filepath.Join("/sys/class/block", devname, "dev"))
			if err != nil {
				log.Printf("Failed to read major:minor for %s: %v", devname, err)
				continue
			}
			parts := strings.Split(strings.TrimSpace(string(majmin)), ":")
			major, _ := strconv.ParseInt(parts[0], 10, 8)
			minor, _ := strconv.ParseInt(parts[1], 10, 8)
			if err := unix.Mknod(filepath.Join("/dev", devname), unix.S_IFBLK|0600, int(major<<16|minor)); err != nil {
				log.Printf("Mknod(%s) failed: %v", devname, err)
				continue
			}
		}

		d, err := drive.Open(devpath)
		if err != nil {
			log.Printf("drive.Open(%s): %v", devpath, err)
			continue
		}
		defer d.Close()
		identity, err := d.Identify()
		if err != nil {
			log.Printf("drive.Identify(%s): %v", devpath, err)
		}
		dsn, err := d.SerialNumber()
		if err != nil {
			log.Printf("drive.SerialNumber(%s): %v", devpath, err)
		}
		d0, err := tcg.Discovery0(d)
		if err != nil {
			if err != tcg.ErrNotSupported {
				log.Printf("tcg.Discovery0(%s): %v", devpath, err)
			}
			continue
		}
		if d0.Locking != nil && d0.Locking.Locked {
			log.Printf("Drive %s is locked", identity)
			if d0.Locking.MBREnabled && !d0.Locking.MBRDone {
				log.Printf("Drive %s has active shadow MBR", identity)
			}
			unlocked := false
			for !unlocked {
				// reuse-existing password for multiple drives
				if password == "" {
					password = getDrivePassword()
					if password == "" {
						// skip on empty password
						break
					}
				}
				if err := unlock(d, password, dsn); err != nil {
					log.Printf("Failed to unlock %s: %v", identity, err)
					// clear password to be queried again
					password = ""
				} else {
					unlocked = true
				}
			}
			if unlocked {
				log.Printf("Drive %s has been unlocked", devpath)
				startEmergencyShell = false
			}
		} else {
			log.Printf("Considered drive %s, but drive is not locked", identity)
		}
	}

	if startEmergencyShell {
		log.Printf("No drives changed state to unlocked, starting shell for troubleshooting")
		return
	}

	fmt.Println()
	if waitForEnter("Starting OS in 3 seconds, press Enter to start shell instead: ", 3) {
		return
	}

	// reboot for now as 'boot' would mount filesystems and therefore mess up hibernation :-(
	// note that ext3 or ext4 will replay its journal even when mounted read-only if the filesystem is dirty
	Execute("/bbin/shutdown", "reboot")
}

func getDrivePassword() string {
	// avoid kernel log messages messing up prompt
	if err := ulog.KernelLog.SetConsoleLogLevel(ulog.KLogWarning); err != nil {
		log.Printf("Could not set log level KLogWarning: %v", err)
	}

	fmt.Println()
	fmt.Printf("Enter OPAL drive password (empty to skip): ")
	bytePassword, err := term.ReadPassword(0)
	fmt.Println()
	if err != nil {
		log.Printf("terminal.ReadPassword(0): %v", err)
		return ""
	}
	return string(bytePassword)
}

func unlock(d tcg.DriveIntf, pass string, driveserial []byte) error {
	// Same format as used by sedutil for compatibility
	salt := fmt.Sprintf("%-20s", string(driveserial))
	pin := pbkdf2.Key([]byte(pass), []byte(salt[:20]), 75000, 32, sha1.New)

	cs, lmeta, err := locking.Initialize(d)
	if err != nil {
		return fmt.Errorf("locking.Initialize: %v", err)
	}
	defer cs.Close()
	l, err := locking.NewSession(cs, lmeta, locking.DefaultAuthority(pin))
	if err != nil {
		return fmt.Errorf("locking.NewSession: %v", err)
	}
	defer l.Close()

	for i, r := range l.Ranges {
		if err := r.UnlockRead(); err != nil {
			log.Printf("Read unlock range %d failed: %v", i, err)
		}
		if err := r.UnlockWrite(); err != nil {
			log.Printf("Write unlock range %d failed: %v", i, err)
		}
	}

	if l.MBREnabled && !l.MBRDone {
		if err := l.SetMBRDone(true); err != nil {
			return fmt.Errorf("SetMBRDone: %v", err)
		}
	}
	return nil
}

func waitForEnter(prompt string, seconds int) bool {

	f, err := os.OpenFile("/dev/console", os.O_RDWR, 0)
	if err != nil {
		log.Printf("ERROR: Open /dev/console failed: %v", err)
		return false
	}
	defer f.Close()

	oldState, err := term.MakeRaw(int(f.Fd()))
	if err != nil {
		log.Printf("ERROR: MakeRaw failed for Fd %d: %v", f.Fd(), err)
		return false
	}
	defer term.Restore(int(f.Fd()), oldState)

	if err = syscall.SetNonblock(int(f.Fd()), true); err != nil {
		log.Printf("ERROR: SetNonblock failed for Fd %d: %v", f.Fd(), err)
		return false
	}

	newTerm := term.NewTerminal(f, prompt)
	for i := 0; i < seconds*2; i++ {
		if i > 0 {
			fmt.Print(".")
		}
		if err = f.SetDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			log.Printf("ERROR: SetDeadline failed for Fd %d: %v", f.Fd(), err)
			return false
		}
		_, err = newTerm.ReadLine()
		if err == nil {
			return true
		}
	}

	// nobody pressed enter (need \r to reset start of line)
	fmt.Println("\r")
	return false
}

func Execute(name string, args ...string) {
	environ := append(os.Environ(), "USER=root")
	environ = append(environ, "HOME=/root")
	environ = append(environ, "TZ=UTC")

	cmd := exec.Command(name, args...)
	cmd.Dir = "/"
	cmd.Env = environ
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setctty = true
	cmd.SysProcAttr.Setsid = true
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to execute: %v", err)
	}
}
