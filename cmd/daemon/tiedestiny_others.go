//go:build !windows
// +build !windows

package daemon

import (
	"log"
	"os"
	"syscall"
	"time"
)

func tieDestiny(destinyPid int64) {
	step := func() {
		proc, err := os.FindProcess(int(destinyPid))
		if err != nil {
			log.Printf("While looking for destiny PID %d: %+v", destinyPid, err)
			os.Exit(0)
		}

		if proc == nil {
			log.Printf("Destiny PID %d exited, exiting too", destinyPid)
			os.Exit(0)
		}
		defer proc.Release()

		err = proc.Signal(syscall.Signal(0))
		if err != nil {
			log.Printf("While signalling destiny PID %d: %+v, exiting", destinyPid, err)
			os.Exit(0)
		}
	}

	for {
		step()
		time.Sleep(1 * time.Second)
	}
}
