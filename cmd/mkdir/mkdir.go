package mkdir

import (
	"os"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/archiver"
)

var args = struct {
	path *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("mkdir", "Create an empty directory and all required parent directories (mkdir -p)").Hidden()
	args.path = cmd.Arg("path", "Directory to create").Required().String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, *args.path))
}

func Do(ctx *mansion.Context, dir string) error {
	comm.Debugf("mkdir -p %s", dir)

	err := os.MkdirAll(dir, archiver.DirMode)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
