//+build darwin

package runner

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/itchio/butler/runner/policies"
	"github.com/itchio/ox/macox"
	"github.com/pkg/errors"
)

var investigateSandbox = os.Getenv("INVESTIGATE_SANDBOX") == "1"

type sandboxExecRunner struct {
	params *RunnerParams
}

var _ Runner = (*sandboxExecRunner)(nil)

func newSandboxExecRunner(params *RunnerParams) (Runner, error) {
	ser := &sandboxExecRunner{
		params: params,
	}
	return ser, nil
}

func (ser *sandboxExecRunner) Prepare() error {
	consumer := ser.params.RequestContext.Consumer

	// make sure we have sandbox-exec
	{
		cmd := exec.Command("sandbox-exec", "-n", "no-network", "true")
		err := cmd.Run()
		if err != nil {
			consumer.Warnf("While verifying sandbox-exec: %s", err.Error())
			return errors.New("Cannot set up itch.io sandbox, see logs for details")
		}
	}

	return nil
}

func (ser *sandboxExecRunner) Run() error {
	params := ser.params
	consumer := params.RequestContext.Consumer

	consumer.Infof("Creating shim app bundle to enable sandboxing")
	realBundlePath := params.FullTargetPath

	binaryPath, err := macox.GetExecutablePath(realBundlePath)
	if err != nil {
		return errors.WithStack(err)
	}
	binaryName := filepath.Base(binaryPath)

	sandboxProfilePath := filepath.Join(params.InstallFolder, ".itch", "isolate-app.sb")
	consumer.Opf("Writing sandbox profile to (%s)", sandboxProfilePath)
	err = os.MkdirAll(filepath.Dir(sandboxProfilePath), 0755)
	if err != nil {
		return errors.WithStack(err)
	}

	userLibrary, err := macox.GetLibraryPath()
	if err != nil {
		return errors.WithStack(err)
	}

	sandboxSource := policies.SandboxExecTemplate
	sandboxSource = strings.Replace(
		sandboxSource,
		"{{USER_LIBRARY}}",
		userLibrary,
		-1, /* replace all instances */
	)
	sandboxSource = strings.Replace(
		sandboxSource,
		"{{INSTALL_LOCATION}}",
		params.InstallFolder,
		-1, /* replace all instances */
	)

	err = ioutil.WriteFile(sandboxProfilePath, []byte(sandboxSource), 0644)
	if err != nil {
		return errors.WithStack(err)
	}

	workDir, err := ioutil.TempDir("", "butler-shim-bundle")
	if err != nil {
		return errors.WithStack(err)
	}
	defer os.RemoveAll(workDir)

	shimBundlePath := filepath.Join(
		workDir,
		filepath.Base(realBundlePath),
	)
	consumer.Opf("Generating shim bundle as (%s)", shimBundlePath)

	shimBinaryPath := filepath.Join(
		shimBundlePath,
		"Contents",
		"MacOS",
		binaryName,
	)
	err = os.MkdirAll(filepath.Dir(shimBinaryPath), 0755)
	if err != nil {
		return errors.WithStack(err)
	}

	shimBinaryContents := fmt.Sprintf(`#!/bin/bash
		cd "%s"
		sandbox-exec -f "%s" "%s" "$@"
		`,
		params.Dir,
		sandboxProfilePath,
		binaryPath,
	)

	err = ioutil.WriteFile(shimBinaryPath, []byte(shimBinaryContents), 0744)
	if err != nil {
		return errors.WithStack(err)
	}

	err = os.Symlink(
		filepath.Join(realBundlePath, "Contents", "Resources"),
		filepath.Join(shimBundlePath, "Contents", "Resources"),
	)
	if err != nil {
		return errors.WithStack(err)
	}

	err = os.Symlink(
		filepath.Join(realBundlePath, "Contents", "Info.plist"),
		filepath.Join(shimBundlePath, "Contents", "Info.plist"),
	)
	if err != nil {
		return errors.WithStack(err)
	}

	if investigateSandbox {
		consumer.Warnf("Wrote shim app to (%s), waiting forever because INVESTIGATE_SANDBOX is set to 1")
		for {
			time.Sleep(1 * time.Second)
		}
	}

	consumer.Statf("All set, hope for the best")

	return RunAppBundle(
		params,
		shimBundlePath,
	)
}
