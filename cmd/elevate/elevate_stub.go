//go:build !windows && !linux
// +build !windows,!linux

package elevate

import (
	"fmt"
	"runtime"
)

func Elevate(params *ElevateParams) (int, error) {
	return 0, fmt.Errorf("elevate is a not supported on %s", runtime.GOOS)
}
