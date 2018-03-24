// +build !linux

package runner

import (
	"runtime"

	"github.com/go-errors/errors"
)

func newFirejailRunner(params *RunnerParams) (Runner, error) {
	return nil, errors.Errorf("firejail runner is not implemented on %s", runtime.GOOS)
}
