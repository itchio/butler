package runlock

import (
	"strconv"

	"golang.org/x/sys/windows"
)

// processIdentity names the process with the given PID by its creation
// time, which a reused PID does not share.
func processIdentity(pid int) (string, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)

	var creation, exit, kernel, user windows.Filetime
	err = windows.GetProcessTimes(handle, &creation, &exit, &kernel, &user)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(creation.Nanoseconds(), 10), nil
}
