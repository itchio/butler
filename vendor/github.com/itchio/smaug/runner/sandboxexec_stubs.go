//+build !darwin

package runner

import (
	"fmt"
	"runtime"
)

func newSandboxExecRunner(params RunnerParams) (Runner, error) {
	return nil, fmt.Errorf("sandbox-exec runner: not supported on %s", runtime.GOOS)
}
