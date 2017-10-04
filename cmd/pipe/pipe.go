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

var args = struct {
	command *[]string
	stdin   *string
	stdout  *string
	stderr  *string
}{}

func Register(ctx *butler.Context) {
	cmd := ctx.App.Command("pipe", "Runs a command, redirecting stdin/stdout/stderr to named pipes").Hidden()
	args.command = cmd.Arg("command", "A command to run, with arguments").Strings()
	args.stdin = cmd.Flag("stdin", "A named pipe to read stdin from").String()
	args.stdout = cmd.Flag("stdout", "A named pipe to write stdout to").String()
	args.stderr = cmd.Flag("stderr", "A named pipe to write stderr to").String()
	ctx.Register(cmd, do)
}

func do(ctx *butler.Context) {
	ctx.Must(Do(ctx, *args.command, *args.stdin, *args.stdout, *args.stderr))
}

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
