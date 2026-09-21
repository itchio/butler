package runlock

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// processIdentity names the process with the given PID in a way that a
// reused PID does not share: the boot it started in and its start time
// in clock ticks since that boot.
func processIdentity(pid int) (string, error) {
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return "", err
	}
	// The command name in parentheses can hold spaces; fields follow it.
	rest := string(stat)
	if i := strings.LastIndexByte(rest, ')'); i >= 0 {
		rest = rest[i+1:]
	}
	fields := strings.Fields(rest)
	// Field 22 of the file is the start time; 19 after the trailing paren.
	if len(fields) < 20 {
		return "", fmt.Errorf("unexpected /proc/%d/stat format", pid)
	}
	start := fields[19]
	bootID, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bootID)) + ":" + start, nil
}
