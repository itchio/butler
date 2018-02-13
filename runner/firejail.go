package runner

import (
	"fmt"
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
	// nothing to prepare
	return nil
}

func (fr *firejailRunner) Run() error {
	params := fr.params
	consumer := params.Consumer

	firejailName := fmt.Sprintf("firejail-%s", params.Runtime.Arch())
	firejailPath := filepath.Join(params.PrereqsDir, firejailName, "firejail")

	consumer.Infof("Running (%s) through firejail", params.FullTargetPath)

	var args []string
	args = append(args, "--noprofile")
	args = append(args, "--")
	args = append(args, params.FullTargetPath)
	args = append(args, params.Args...)

	cmd := exec.CommandContext(params.Ctx, firejailPath, args...)
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
