package walk

import (
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/headway/united"
	"github.com/itchio/lake/tlc"
	"github.com/pkg/errors"
)

var args = struct {
	dir         *string
	dereference *bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("walk", "Finds all files in a directory").Hidden()
	args.dir = cmd.Arg("dir", "A dir you want to walk").Required().String()
	args.dereference = cmd.Flag("dereference", "Follow symlinks").Default("false").Bool()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(*args.dir, *args.dereference))
}

func Do(dir string, dereference bool) error {
	startTime := time.Now()

	container, err := tlc.WalkDir(dir, tlc.WalkOpts{
		Dereference: dereference,
	})
	if err != nil {
		return errors.Wrap(err, "walking")
	}

	totalEntries := 0
	send := func(path string) {
		totalEntries++
		comm.ResultOrPrint(&mansion.WalkResult{
			Type: "entry",
			Path: path,
		}, func() {
			comm.Logf("- %s", path)
		})
	}

	for _, f := range container.Files {
		send(f.Path)
	}

	for _, s := range container.Symlinks {
		send(s.Path)
	}

	comm.ResultOrPrint(&mansion.WalkResult{
		Type: "totalSize",
		Size: container.Size,
	}, func() {
		comm.Statf("%s walked in %s",
			container,
			united.FormatDuration(time.Since(startTime)),
		)
	})

	return nil
}
