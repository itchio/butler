package runlock

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// processIdentity names the process with the given PID by its start
// time, which a reused PID does not share.
func processIdentity(pid int) (string, error) {
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return "", err
	}
	start := kp.Proc.P_starttime
	return fmt.Sprintf("%d.%06d", start.Sec, start.Usec), nil
}
