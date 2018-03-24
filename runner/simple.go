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

	ctx := params.Ctx
	cmd := exec.Command(params.FullTargetPath, params.Args...)
	cmd.Dir = params.Dir
	cmd.Env = params.Env
	cmd.Stdout = params.Stdout
	cmd.Stderr = params.Stderr

	err := SetupProcessGroup(consumer, cmd)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = WaitProcessGroup(consumer, cmd, ctx)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
