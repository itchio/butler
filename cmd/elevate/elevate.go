package elevate

import (
	"io"
	"os"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/mansion"
)

var args = struct {
	command *[]string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("elevate", "Runs a command as administrator").Hidden()
	args.command = cmd.Arg("command", "A command to run, with arguments").Strings()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(*args.command))
}

type ElevateParams struct {
	Command []string
	Stdout  io.Writer
	Stderr  io.Writer
}

func Do(command []string) error {
	ret, err := Elevate(&ElevateParams{
		Command: command,
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	os.Exit(ret)
	return nil // you silly goose of a compiler...
}
