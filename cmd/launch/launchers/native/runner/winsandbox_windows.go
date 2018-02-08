// +build windows

package runner

import "errors"

func newWinSandboxRunner(params *RunnerParams) (Runner, error) {
	return nil, errors.New("stub")
}
