// +build !windows

package runner

import (
	"os/exec"

	"github.com/pkg/errors"
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
	consumer := params.Consumer

	cmd := exec.Command(params.FullTargetPath, params.Args...)
	cmd.Dir = params.Dir
	cmd.Env = params.Env
	cmd.Stdout = params.Stdout
	cmd.Stderr = params.Stderr

	pg, err := NewProcessGroup(consumer, cmd, params.Ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	err = cmd.Start()
	if err != nil {
		return errors.WithStack(err)
	}

	err = pg.AfterStart()
	if err != nil {
		return errors.WithStack(err)
	}

	err = pg.Wait()
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
