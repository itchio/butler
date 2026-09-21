//go:build !linux && !darwin && !windows

package runlock

import "errors"

func processIdentity(pid int) (string, error) {
	return "", errors.New("cannot inspect processes on this platform")
}
