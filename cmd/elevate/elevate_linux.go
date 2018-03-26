// +build linux

package elevate

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
)

func Elevate(params *ElevateParams) (int, error) {
	butlerExe, err := os.Executable()
	if err != nil {
		return -1, errors.WithStack(err)
	}

	dir, err := os.Getwd()
	if err != nil {
		return 1, errors.WithStack(err)
	}

	// we use 'butler run' because pkexec loses the CWD,
	// which is impractical. it also sanitizes the environment,
	// which is a good thing!
	var args []string
	args = append(args, butlerExe)
	args = append(args, "run")
	args = append(args, "--dir")
	args = append(args, dir)
	args = append(args, "--")
	args = append(args, params.Command...)

	cmd := exec.Command("pkexec", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = dir

	err = cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				// pkexec returns 126 if the user declines, we convert it
				// to our standard exit code
				if status.ExitStatus() == 126 {
					return ExitCodeAccessDenied, nil
				}
				return status.ExitStatus(), nil
			}
		}

		return 1, err
	}

	return 0, nil
}
