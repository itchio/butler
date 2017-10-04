// +build windows

package pipe

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/butler"
	"github.com/natefinch/npipe"
)

func Do(ctx *butler.Context, command []string, stdin string, stdout string, stderr string) error {
	cmd := exec.Command(command[0], command[1:]...)

	hook := func(namedPath string, fallback *os.File) io.Writer {
		pipe, err := npipe.DialTimeout(namedPath, 1*time.Second)
		if err != nil {
			return fallback
		}
		return pipe
	}

	cmd.Stdout = hook(stdout, os.Stdout)
	cmd.Stderr = hook(stderr, os.Stderr)

	exitCode := 0

	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if stat, ok := ee.ProcessState.Sys().(syscall.WaitStatus); ok {
				exitCode = int(stat.ExitCode)
			}
		} else {
			fmt.Fprintf(cmd.Stderr, "While running %s: %s", command[0], err.Error())
			exitCode = 1
			return errors.Wrap(err, 0)
		}
	}

	os.Exit(exitCode)
	return nil // you're a silly compiler, you know that?
}
