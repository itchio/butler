package mkzip

import (
	"io"
	"os"
	"time"

	"github.com/itchio/headway/counter"
	"github.com/itchio/headway/united"

	"github.com/itchio/arkive/zip"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/butler/mansion"

	"github.com/itchio/lake/pools/fspool"
	"github.com/itchio/lake/pools/zipwriterpool"
	"github.com/itchio/lake/tlc"
)

var args = struct {
	out    string
	dir    string
	preset string

	blockSize int
	blocks    int
	level     int
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("mkzip", "(Advanced) Create a .zip file").Hidden()
	cmd.Arg("out", "Output file").Required().StringVar(&args.out)
	cmd.Arg("dir", "Directory to compress").Required().ExistingDirVar(&args.dir)
	cmd.Flag("preset", "Compression preset").Default("default").EnumVar(&args.preset, "default", "best")
	cmd.Flag("level", "Compression level").Default("-3").IntVar(&args.level)
	cmd.Flag("block-size", "Compression block size (for pflate)").Default("-1").IntVar(&args.blockSize)
	cmd.Flag("blocks", "Number of parallel blocks (for pflate)").Default("-1").IntVar(&args.blocks)
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

	{
		if args.preset != "default" {
			consumer.Opf("Using compression preset %q", args.preset)
		}

		var err error
		switch args.preset {
		case "default":
			err = zw.SetCompressionSettings(zip.DefaultCompressionSettings())
		case "best":
			err = zw.SetCompressionSettings(zip.BestCompressionSettings())
		}
		if err != nil {
			return err
		}
	}

	if args.level >= 0 {
		settings := zw.GetCompressionSettings()
		consumer.Opf("Forcing flate level to %d", args.level)
		settings.Flate.Level = args.level
		err := zw.SetCompressionSettings(settings)
		if err != nil {
			return err
		}
	}

	if args.blockSize >= 0 {
		settings := zw.GetCompressionSettings()
		consumer.Opf("Forcing block size to %d", args.blockSize)
		settings.Flate.BlockSize = args.blockSize
		err := zw.SetCompressionSettings(settings)
		if err != nil {
			return err
		}
	}

	if args.blocks >= 0 {
		settings := zw.GetCompressionSettings()
		consumer.Opf("Forcing blocks to %d", args.blocks)
		settings.Flate.Blocks = args.blocks
		err := zw.SetCompressionSettings(settings)
		if err != nil {
			return err
		}
	}

	{
		cs := zw.GetCompressionSettings()
		consumer.Opf("Compressing at Q%d, with %d blocks of %s",
			cs.Flate.Level,
			cs.Flate.Blocks,
			united.FormatBytes(int64(cs.Flate.BlockSize)),
		)
	}

	dst, err := zipwriterpool.New(container, zw)
	if err != nil {
		return err
	}

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
