package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/u-root/u-root/pkg/libinit"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/ulog"
)

const banner = `
███████╗██╗      █████╗ ███████╗████████╗██╗  ██╗
██╔════╝██║     ██╔══██╗██╔════╝╚══██╔══╝╚██╗██╔╝
█████╗  ██║     ███████║███████╗   ██║    ╚███╔╝
██╔══╝  ██║     ██╔══██║╚════██║   ██║    ██╔██╗
███████╗███████╗██║  ██║███████║   ██║   ██╔╝ ██╗
╚══════╝╚══════╝╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝
`

var (
	Version = "(devel)"
	GitHash = "(no hash)"
)

func main() {
	fmt.Printf("\n")
	fmt.Printf(banner)
	fmt.Printf("Welcome to Elastx PBA version %s (git %s)\n\n", Version, GitHash)
	log.SetPrefix("elx-pba: ")

	if _, err := mount.Mount("proc", "/proc", "proc", "", 0); err != nil {
		log.Fatalf("Mount(proc): %v", err)
	}
	if _, err := mount.Mount("sysfs", "/sys", "sysfs", "", 0); err != nil {
		log.Fatalf("Mount(sysfs): %v", err)
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
			Shell()
		}
	}()

	for {
		fmt.Printf("Hello, I am elx-pba\n")
		time.Sleep(1 * time.Second)
		break
	}
}

func intrHandler(cmd *exec.Cmd, exited chan bool) {
	for {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		select {
		case _ = <-c:
			cmd.Process.Signal(os.Interrupt)
		case _ = <-exited:
			return
		}
	}
}

func Shell() {
	environ := append(os.Environ(), "USER=root")
	environ = append(environ, "HOME=/root")
	environ = append(environ, "TZ=UTC")

	cmd := exec.Command("/bbin/elvish")
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
	exited := make(chan bool)
	// Forward intr to the shell
	go intrHandler(cmd, exited)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to execute: %v", err)
	}
	exited <- true
}
