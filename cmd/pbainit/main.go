package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tcg "github.com/open-source-firmware/go-tcg-storage/pkg/core"
	"github.com/open-source-firmware/go-tcg-storage/pkg/drive"
	"github.com/u-root/u-root/pkg/libinit"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/mount/block"
	"github.com/u-root/u-root/pkg/ulog"
	"golang.org/x/sys/unix"

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
	fmt.Printf("Welcome to Elastx PBA version %s (git %s)\n\n", Version, GitHash)
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
		log.Printf("Could not set log level: %v", err)
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

	dmi, err := readDMI()
	if err != nil {
		log.Printf("Failed to read SMBIOS/DMI data: %v", err)
		return
	}

	log.Printf("System UUID:            %s", dmi.SystemUUID)
	log.Printf("System serial:          %s", dmi.SystemSerialNumber)
	log.Printf("Baseboard manufacturer: %s", dmi.BaseboardManufacturer)
	log.Printf("Baseboard product:      %s", dmi.BaseboardProduct)
	log.Printf("Baseboard serial:       %s", dmi.BaseboardSerialNumber)
	log.Printf("Chassis serial:         %s", dmi.ChassisSerialNumber)

	sysblk, err := os.ReadDir("/sys/class/block/")
	if err != nil {
		log.Printf("Failed to enumerate block devices: %v", err)
		return
	}

	unlocked := false
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
		if d0.Locking != nil {
			if d0.Locking.Locked {
				log.Printf("Drive %s is locked", identity)
			}
			if d0.Locking.MBREnabled && !d0.Locking.MBRDone {
				log.Printf("Drive %s has active shadow MBR", identity)
			}
			pass := fmt.Sprintf("%s", dmi.SystemUUID)
			if err := pba.Unlock(d, pass, dsn); err != nil {
				log.Printf("Failed to unlock %s: %v", identity, err)
				d.Close()
				continue
			}
			bd, err := block.Device(devpath)
			if err != nil {
				log.Printf("block.Device(%s): %v", devpath, err)
				d.Close()
				continue
			}
			if err := bd.ReadPartitionTable(); err != nil {
				log.Printf("block.ReadPartitionTable(%s): %v", devpath, err)
				d.Close()
				continue
			}
			log.Printf("Drive %s has been unlocked", devpath)
			unlocked = true
		} else {
			log.Printf("Considered drive %s, but drive is not locked", identity)
		}
		d.Close()
	}

	if !unlocked {
		log.Printf("No drives changed state to unlocked, starting shell for troubleshooting")
		return
	}

	reader := bufio.NewReader(os.Stdin)
	abort := make(chan bool)
	var pciBlock bool
	go func() {
		fmt.Println("")
		log.Printf("Starting 'boot' in 5 seconds, press Enter to start shell instead")
		select {
		case <-abort:
			return
		case <-time.After(5 * time.Second):
		}
		if dmi.BaseboardManufacturer == "Supermicro" {
			if strings.HasPrefix(dmi.BaseboardProduct, "X10SDV-7TP4F") || strings.HasPrefix(dmi.BaseboardProduct, "X11SDV-8C-TP8F") || strings.HasPrefix(dmi.BaseboardProduct, "X12DPD-A6M25") {
				pciBlock = true
			}
		}
		if pciBlock {
			// object storage nodes with OpenCAS get corrupted filesystems if LSI cards are mounted during boot (0064, 00c9, 00c4 are LSI, per lspci -nn)
			log.Printf("Work-around: Swift node, do not mount data disks when searching for kernel!")
			pba.Execute("/bbin/boot", "-block=0x1000:0x0064,0x1000:0x00c9,0x1000:0x00c4")
		} else {
			pba.Execute("/bbin/boot")
		}
	}()

	reader.ReadString('\n')
	abort <- true
}
