package cave

import (
	"fmt"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/butler"
)

func Register(ctx *butler.Context) {
	cmd := ctx.App.Command("cave", "Handle a cave (game install) for the itch app").Hidden()
	ctx.Register(cmd, do)
}

func do(ctx *butler.Context) {
	ctx.Must(Do(ctx))
}

var tr *JSONTransport

func Do(ctx *butler.Context) error {
	tr = NewJSONTransport()
	tr.Start()

	var command CaveCommand
	err := readMessage("cave-command", &command)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	switch command.Operation {
	case CaveCommandOperationInstall:
		return doCaveInstall(ctx, command.InstallParams)
	default:
		return fmt.Errorf("Unknown cave command operation '%s'", command.Operation)
	}
}
