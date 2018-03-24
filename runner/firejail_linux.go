// +build linux

package runner

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/runner/policies"
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
	consumer := params.RequestContext.Consumer

	firejailName := fmt.Sprintf("firejail-%s", params.Runtime.Arch())
	firejailPath := filepath.Join(params.PrereqsDir, firejailName, "firejail")

	sandboxProfilePath := filepath.Join(params.InstallFolder, ".itch", "isolate-app.profile")
	consumer.Opf("Writing sandbox profile to (%s)", sandboxProfilePath)
	err := os.MkdirAll(filepath.Dir(sandboxProfilePath), 0755)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	sandboxSource := policies.FirejailTemplate
	err = ioutil.WriteFile(sandboxProfilePath, []byte(sandboxSource), 0644)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Opf("Running (%s) through firejail", params.FullTargetPath)

	var args []string
	args = append(args, fmt.Sprintf("--profile=%s", sandboxProfilePath))
	args = append(args, "--")
	args = append(args, params.FullTargetPath)
	args = append(args, params.Args...)

	cmd := exec.Command(firejailPath, args...)
	cmd.Dir = params.Dir
	cmd.Env = params.Env
	cmd.Stdout = params.Stdout
	cmd.Stderr = params.Stderr

	pg, err := NewProcessGroup(consumer, cmd, params.Ctx)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = pg.AfterStart()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = pg.Wait()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
