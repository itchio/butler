//+build windows

package daemon

import (
	"log"
	"os"
)

func tieDestiny(destinyPid int64) {
	proc, err := os.FindProcess(int(destinyPid))
	if err != nil {
		log.Printf("While looking for destiny PID %d: %+v", destinyPid, err)
		os.Exit(1)
	}

	if proc == nil {
		log.Printf("Destiny PID %d exited, exiting too", destinyPid)
		os.Exit(1)
	}

	_, err = proc.Wait()
	if err != nil {
		log.Printf("While waiting on destiny PID %d: %+v, exiting", destinyPid, err)
		os.Exit(1)
	}

	log.Printf("Destiny PID %d exited, exiting too", destinyPid)
	os.Exit(0)
}
