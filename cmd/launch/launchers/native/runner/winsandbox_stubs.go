// +build !windows

package runner

import (
	"fmt"
	"runtime"
)

func newWinSandboxRunner(params *RunnerParams) (Runner, error) {
	return nil, fmt.Errorf("winsandbox runner: not supported on %s", runtime.GOOS)
}
