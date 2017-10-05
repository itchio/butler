package sizeof

import (
	"os"
	"path/filepath"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
)

var args = struct {
	path *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("sizeof", "Compute the total size of a directory").Hidden()
	args.path = cmd.Arg("path", "Directory to compute the size of").Required().String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, *args.path))
}

func Do(ctx *mansion.Context, path string) error {
	totalSize := int64(0)

	inc := func(_ string, f os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		totalSize += f.Size()
		return nil
	}

	filepath.Walk(path, inc)
	comm.ResultOrPrint(totalSize, func() {
		comm.Logf("Total size of %s: %s", path, humanize.IBytes(uint64(totalSize)))
	})

	return nil
}
