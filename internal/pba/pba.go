package pba

import (
	"crypto/sha1"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	tcg "github.com/open-source-firmware/go-tcg-storage/pkg/core"
	"github.com/open-source-firmware/go-tcg-storage/pkg/locking"
	"golang.org/x/crypto/pbkdf2"
)

// Unlock derives a key using the sedutil-compatible PBKDF2 format and unlocks
// all TCG locking ranges on the drive, clearing the shadow MBR if active.
func Unlock(d tcg.DriveIntf, pass string, driveserial []byte) error {
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
