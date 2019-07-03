package mkzip

import (
	"io"
	"os"
	"time"

	"github.com/itchio/headway/united"
	"github.com/itchio/headway/counter"

	"github.com/itchio/arkive/zip"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/butler/mansion"

	"github.com/itchio/lake/tlc"
	"github.com/itchio/lake/pools/fspool"
	"github.com/itchio/lake/pools/zipwriterpool"
)

var args = struct {
	out string
	dir string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("mkzip", "(Advanced) Create a .zip file").Hidden()
	cmd.Arg("out", "Output file").Required().StringVar(&args.out)
	cmd.Arg("dir", "Directory to compress").Required().ExistingDirVar(&args.dir)
	ctx.Register(cmd, func(ctx *mansion.Context) {
		ctx.Must(do(ctx))
	})
}

func do(ctx *mansion.Context) error {
	consumer := comm.NewStateConsumer()

	consumer.Opf("Walking %s...", args.dir)
	walkOpts := &tlc.WalkOpts{
		Filter: filtering.FilterPaths,
	}
	walkOpts.Wrap(&args.dir)
	container, err := tlc.WalkDir(args.dir, walkOpts)
	if err != nil {
		return err
	}

	consumer.Statf("Found %s", container)

	src := fspool.New(container, args.dir)

	w, err := os.Create(args.out)
	if err != nil {
		return err
	}

	zw := zip.NewWriter(w)

	dst := zipwriterpool.New(container, zw)

	var totalBytes int64

	doFile := func(fileIndex int64) error {
		file := container.Files[fileIndex]
		consumer.ProgressLabel(file.Path)

		fsrc, err := src.GetReader(fileIndex)
		if err != nil {
			return err
		}

		fdst, err := dst.GetWriter(fileIndex)
		if err != nil {
			return err
		}
		defer fdst.Close()

		cw := counter.NewWriterCallback(func(done int64) {
			p := float64(totalBytes+done) / float64(container.Size)
			consumer.Progress(p)
		}, fdst)

		_, err = io.Copy(cw, fsrc)
		if err != nil {
			return err
		}

		totalBytes += file.Size

		return nil
	}

	consumer.Opf("Compressing...")
	comm.StartProgressWithTotalBytes(container.Size)
	startTime := time.Now()

	numFiles := len(container.Files)
	for i := 0; i < numFiles; i++ {
		err = doFile(int64(i))
		if err != nil {
			return err
		}
	}
	comm.EndProgress()

	err = dst.Close()
	if err != nil {
		return err
	}

	duration := time.Since(startTime)
	consumer.Statf("Compressed @ %s (%s total)",
		united.FormatBPS(container.Size, duration),
		united.FormatDuration(duration),
	)
	return nil
}
