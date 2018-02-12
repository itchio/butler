package runner

import (
	"os/exec"
	"path/filepath"

	"github.com/go-errors/errors"
)

type firejailRunner struct {
	params *RunnerParams
}

var _ Runner = (*firejailRunner)(nil)

func newFirejailRunner(params *RunnerParams) (Runner, error) {
	fr := &firejailRunner{
		params: params,
	}
	return fr, nil
}

func (fr *firejailRunner) Prepare() error {
	params := fr.params
	consumer := params.Consumer

	// nothing to prepare
	prereqsDir := fr.params.LauncherParams.ParentParams.PrereqsDir
	firejailPath := filepath.Join(prereqsDir, "firejail", "firejail")

	args := []string{
		"--noprofile",
		"--",
		"whoami",
	}
	cmd := exec.Command(firejailPath, args...)
	err := cmd.Run()
	if err != nil {
		consumer.Warnf("firejail sanity check failed: %s", err.Error())
		consumer.Infof("Installing firejail...")
	}

	return nil
}

func (fr *firejailRunner) Run() error {
	params := fr.params
	consumer := params.Consumer

	consumer.Infof("Running (%s) through firejail", params.FullTargetPath)

	cmd := exec.CommandContext(params.Ctx, params.FullTargetPath, params.Args...)
	cmd.Dir = params.Dir
	cmd.Env = params.Env
	cmd.Stdout = params.Stdout
	cmd.Stderr = params.Stderr

	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
