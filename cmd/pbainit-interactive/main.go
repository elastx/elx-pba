package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	tcg "github.com/open-source-firmware/go-tcg-storage/pkg/core"
	"github.com/open-source-firmware/go-tcg-storage/pkg/drive"
	"github.com/u-root/u-root/pkg/libinit"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/ulog"
	"golang.org/x/sys/unix"
	"golang.org/x/term"

	"github.com/elastx/elx-pba/internal/pba"
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
			pba.Execute("/bbin/elvish")
		}
	}()

	sysblk, err := os.ReadDir("/sys/class/block/")
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
			majmin, err := os.ReadFile(filepath.Join("/sys/class/block", devname, "dev"))
			if err != nil {
				log.Printf("Failed to read major:minor for %s: %v", devname, err)
				continue
			}
			parts := strings.Split(strings.TrimSpace(string(majmin)), ":")
			if len(parts) != 2 {
				log.Printf("Unexpected major:minor format for %s: %q", devname, string(majmin))
				continue
			}
			major, errMaj := strconv.ParseInt(parts[0], 10, 32)
			minor, errMin := strconv.ParseInt(parts[1], 10, 32)
			if errMaj != nil || errMin != nil {
				log.Printf("Failed to parse major:minor for %s: %v %v", devname, errMaj, errMin)
				continue
			}
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
			d.Close()
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
				if password == "" {
					password = getDrivePassword()
					if password == "" {
						break
					}
				}
				if err := pba.Unlock(d, password, dsn); err != nil {
					log.Printf("Failed to unlock %s: %v", identity, err)
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
		d.Close()
	}

	if startEmergencyShell {
		log.Printf("No drives changed state to unlocked, starting shell for troubleshooting")
		return
	}

	fmt.Println()
	if waitForEnter("Starting OS in 3 seconds, press Enter to start shell instead: ", 3) {
		return
	}

	// reboot rather than kexec: 'boot' mounts filesystems which breaks hibernation,
	// and ext3/ext4 replays its journal even on read-only mounts when the fs is dirty
	pba.Execute("/bbin/shutdown", "reboot")
}

func getDrivePassword() string {
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

	fmt.Println("\r")
	return false
}
