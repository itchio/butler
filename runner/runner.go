package runner

import (
	"context"
	"fmt"
	"io"
	"runtime"

	"github.com/itchio/wharf/state"
	"github.com/sourcegraph/jsonrpc2"
)

type RunnerParams struct {
	Consumer *state.Consumer
	Conn     *jsonrpc2.Conn
	Ctx      context.Context

	Sandbox bool

	InstallFolder  string
	FullTargetPath string

	Name   string
	Dir    string
	Args   []string
	Env    []string
	Stdout io.Writer
	Stderr io.Writer
}

type Runner interface {
	Prepare() error
	Run() error
}

func GetRunner(params *RunnerParams) (Runner, error) {
	switch runtime.GOOS {
	case "windows":
		if params.Sandbox {
			return newWinSandboxRunner(params)
		}
		return newSimpleRunner(params)
	case "linux":
		if params.Sandbox {
			return newFirejailRunner(params)
		}
		return newSimpleRunner(params)
	case "osx":
		if params.Sandbox {
			return newSandboxExecRunner(params)
		}
		return newAppRunner(params)
	}

	return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
}
