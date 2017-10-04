package pipe

import (
	"github.com/itchio/butler/mansion"
)

var args = struct {
	command *[]string
	stdin   *string
	stdout  *string
	stderr  *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("pipe", "Runs a command, redirecting stdin/stdout/stderr to named pipes").Hidden()
	args.command = cmd.Arg("command", "A command to run, with arguments").Strings()
	args.stdin = cmd.Flag("stdin", "A named pipe to read stdin from").String()
	args.stdout = cmd.Flag("stdout", "A named pipe to write stdout to").String()
	args.stderr = cmd.Flag("stderr", "A named pipe to write stderr to").String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, *args.command, *args.stdin, *args.stdout, *args.stderr))
}
