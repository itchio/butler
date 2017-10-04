package elevate

import "github.com/itchio/butler/butler"

var args = struct {
	command *[]string
}{}

func Register(ctx *butler.Context) {
	cmd := ctx.App.Command("elevate", "Runs a command as administrator").Hidden()
	args.command = cmd.Arg("command", "A command to run, with arguments").Strings()
	ctx.Register(cmd, do)
}

func do(ctx *butler.Context) {
	ctx.Must(Do(*args.command))
}
