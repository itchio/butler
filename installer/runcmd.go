package installer

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"github.com/itchio/butler/cmd/elevate"
	"github.com/itchio/butler/installer/loggerwriter"
	"github.com/itchio/headway/state"
	"github.com/pkg/errors"
)

// RunCommand starts and waits for an *exec.Cmd to finish,
// and goes through a weird type-casting dance to retrieve
// the actual exit code.
func RunCommand(consumer *state.Consumer, cmdTokens []string) (int, error) {
	consumer.Infof("→ Running command:")
	consumer.Infof("  %s", strings.Join(cmdTokens, " ::: "))

	cmd := exec.Command(cmdTokens[0], cmdTokens[1:]...)
	cmd.Stdout = loggerwriter.New(consumer, "out")
	cmd.Stderr = loggerwriter.New(consumer, "err")

	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus(), nil
			}
		}

		return 127, err
	}

	return 0, nil
}

func RunElevatedCommand(consumer *state.Consumer, cmdTokens []string) (int, error) {
	consumer.Infof("→ Running elevated command:")
	consumer.Infof("  %s", strings.Join(cmdTokens, " ::: "))

	elevateParams := &elevate.ElevateParams{
		Command: cmdTokens,
		Stdout:  loggerwriter.New(consumer, "out"),
		Stderr:  loggerwriter.New(consumer, "err"),
	}

	return elevate.Elevate(elevateParams)
}

func CheckExitCode(exitCode int, err error) error {
	if err != nil {
		return errors.WithStack(err)
	}

	if exitCode != 0 {
		return fmt.Errorf("non-zero exit code %d (%x)", exitCode, exitCode)
	}

	return nil
}
