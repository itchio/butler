//+build !darwin

package runner

import (
	"fmt"
	"runtime"
)

func newAppRunner(params *RunnerParams) (Runner, error) {
	return nil, fmt.Errorf("app runner: not supported on %s", runtime.GOOS)
}
