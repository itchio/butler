package unsz

import (
	"time"

	"github.com/itchio/butler/archive"
	"github.com/itchio/butler/archive/backends/szah"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

var args = struct {
	file *string
	dir  *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("unsz", "Extract any archive file supported by 7-zip").Hidden()
	args.file = cmd.Arg("file", "Path of the archive to extract").Required().String()
	args.dir = cmd.Flag("dir", "An optional directory to which to extract files (defaults to CWD)").Default(".").Short('d').String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, &UnszParams{
		File: *args.file,
		Dir:  *args.dir,

		Consumer: comm.NewStateConsumer(),
		OnUncompressedSizeKnown: func(uncompressedSize int64) {
			comm.StartProgressWithTotalBytes(uncompressedSize)
		},
		OnFinished: func() {
			comm.EndProgress()
		},
	}))
}

type OnFinishedFunc func()

type UnszParams struct {
	File string
	Dir  string

	Consumer *state.Consumer

	OnUncompressedSizeKnown archive.UncompressedSizeKnownFunc
	OnFinished              OnFinishedFunc
}

func Do(ctx *mansion.Context, params *UnszParams) error {
	if params.File == "" {
		return errors.New("unsz: File must be specified")
	}
	if params.Dir == "" {
		return errors.New("unsz: Dir must be specified")
	}

	consumer := params.Consumer

	h := &szah.Handler{}

	file, err := eos.Open(params.File)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Infof("Extracting %s to %s", stats.Name(), params.Dir)

	var uncompressedSize int64

	startTime := time.Now()

	_, err = h.Extract(&archive.ExtractParams{
		OutputPath: params.Dir,
		File:       file,
		Consumer:   params.Consumer,
		OnUncompressedSizeKnown: func(size int64) {
			uncompressedSize = size
			if params.OnUncompressedSizeKnown != nil {
				params.OnUncompressedSizeKnown(size)
			}
		},
	})
	if params.OnFinished != nil {
		params.OnFinished()
	}

	if err != nil {
		return errors.Wrap(err, 0)
	}

	duration := time.Since(startTime)
	bytesPerSec := float64(uncompressedSize) / duration.Seconds()

	consumer.Statf("Extracted %s at %s/s", humanize.IBytes(uint64(uncompressedSize)), humanize.IBytes(uint64(bytesPerSec)))

	return nil
}
