// +build windows

package runner

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/runner/execas"
	"github.com/itchio/butler/runner/syscallex"
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

	cmd := execas.Command(params.FullTargetPath, params.Args...)
	cmd.Dir = params.Dir
	cmd.Env = params.Env
	cmd.Stdout = params.Stdout
	cmd.Stderr = params.Stderr

	cmd.SysProcAttr = &syscallex.SysProcAttr{
		CreationFlags: syscallex.CREATE_SUSPENDED,
	}

	pg, err := NewProcessGroup(consumer, cmd, params.Ctx)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = pg.AfterStart()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = pg.Wait()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
