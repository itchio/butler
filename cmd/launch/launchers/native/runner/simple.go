package runner

import "os/exec"

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
	cmd := exec.CommandContext(params.Ctx, params.FullTargetPath, params.Args...)
	cmd.Dir = params.Dir
	cmd.Env = params.Env
	cmd.Stdout = params.Stdout
	cmd.Stderr = params.Stderr

	return cmd.Run()
}
