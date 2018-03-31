package runner

import (
	"context"
	"fmt"
	"io"
	"runtime"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/manager"
)

type RunnerParams struct {
	RequestContext *butlerd.RequestContext
	Ctx            context.Context

	Sandbox bool

	FullTargetPath string

	Name   string
	Dir    string
	Args   []string
	Env    []string
	Stdout io.Writer
	Stderr io.Writer

	PrereqsDir    string
	Credentials   *butlerd.GameCredentials
	InstallFolder string
	Runtime       *manager.Runtime
}

type Runner interface {
	Prepare() error
	Run() error
}

func GetRunner(params *RunnerParams) (Runner, error) {
	consumer := params.RequestContext.Consumer

	attachRunner, err := getAttachRunner(params)
	if attachRunner != nil {
		return attachRunner, nil
	}
	if err != nil {
		consumer.Warnf("Could not determine if app is aslready running: %s", err.Error())
	}

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
	case "darwin":
		if params.Sandbox {
			return newSandboxExecRunner(params)
		}
		return newAppRunner(params)
	}

	return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
}
