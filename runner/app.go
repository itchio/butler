package runner

import (
	"os/exec"
	"strings"

	"github.com/go-errors/errors"
)

type appRunner struct {
	params *RunnerParams
}

var _ Runner = (*appRunner)(nil)

func newAppRunner(params *RunnerParams) (Runner, error) {
	ar := &appRunner{
		params: params,
	}
	return ar, nil
}

func (ar *appRunner) Prepare() error {
	// TODO: if sandbox is enabled, check that we have
	// "sandbox-exec" here

	// nothing to prepare
	return nil
}

func (ar *appRunner) Run() error {
	params := ar.params
	consumer := params.Consumer

	fullLowerPath := strings.ToLower(params.FullTargetPath)
	if !strings.HasSuffix(fullLowerPath, ".app") {
		// TODO: relax that a bit at some point
		return errors.New("itch only supports launching .app bundles")
	}

	if params.Sandbox {
		// TODO: support
		consumer.Warnf("Sandbox enabled but not supported yet")
	}

	var args = []string{
		"-W",
		params.FullTargetPath,
		"--args",
	}
	args = append(args, params.Args...)

	cmd := exec.CommandContext(params.Ctx, "open", args...)
	// I doubt this matters
	cmd.Dir = params.Dir
	cmd.Env = params.Env
	// 'open' does not relay stdout or stderr, so we don't
	// even bother setting them

	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
