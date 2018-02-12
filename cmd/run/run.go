package run

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/mansion"
)

var args = struct {
	dir     *string
	command *[]string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("run", "Runs a command").Hidden()
	args.dir = cmd.Flag("dir", "The working directory for the command").Hidden().String()
	args.command = cmd.Arg("command", "A command to run, with arguments").Strings()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do())
}

func Do() error {
	command := *args.command
	dir := *args.dir

	cmd := exec.Command(command[0], command[1:]...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		return errors.Wrap(err, 0)
	}

	return nil
}
