// +build !windows

package runner

import (
	"fmt"
	"runtime"
)

func newFujiRunner(params RunnerParams) (Runner, error) {
	return nil, fmt.Errorf("fuji runner: not supported on %s", runtime.GOOS)
}
