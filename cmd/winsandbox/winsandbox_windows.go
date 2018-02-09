// +build windows

package winsandbox

import (
	"errors"

	"github.com/itchio/butler/mansion"
)

var checkArgs = struct{}{}

func Register(ctx *mansion.Context) {
	parentCmd := ctx.App.Command("winsandbox", "Use or manage the itch.io sandbox for Windows")

	{
		cmd := parentCmd.Command("check", "Install prerequisites from an install plan").Hidden()
		ctx.Register(cmd, doCheck)
	}
}

func doCheck(ctx *mansion.Context) {
	ctx.Must(Check(ctx))
}

func Check(ctx *mansion.Context) error {
	return errors.New("stub!")
}
