package runner

import (
	"os/exec"

	"github.com/go-errors/errors"
)

type simpleRunner struct {
	params *RunnerParams
}

var _ Runner = (*simpleRunner)(nil)

func newSimpleRunner(params *RunnerParams) (Runner, error) {
	sr := &simpleRunner{
		params: params,
	}
	return sr, nil
}

func (sr *simpleRunner) Prepare() error {
	// nothing to prepare
	return nil
}

func (sr *simpleRunner) Run() error {
	params := sr.params
	consumer := params.RequestContext.Consumer

	err := SetupJobObject(consumer)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	cmd := exec.CommandContext(params.RequestContext.Ctx, params.FullTargetPath, params.Args...)
	cmd.Dir = params.Dir
	cmd.Env = params.Env
	cmd.Stdout = params.Stdout
	cmd.Stderr = params.Stderr

	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = WaitJobObject(consumer)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
