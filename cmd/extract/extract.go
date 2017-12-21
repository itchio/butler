package extract

import (
	"time"

	"github.com/itchio/savior"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
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
	cmd := ctx.App.Command("extract", "Extract any archive file supported by butler or 7-zip").Hidden()
	args.file = cmd.Arg("file", "Path of the archive to extract").Required().String()
	args.dir = cmd.Flag("dir", "An optional directory to which to extract files (defaults to CWD)").Default(".").Short('d').String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, &ExtractParams{
		File: *args.file,
		Dir:  *args.dir,

		Consumer: comm.NewStateConsumer(),
	}))
}

type ExtractParams struct {
	File string
	Dir  string

	Consumer *state.Consumer
}

func Do(ctx *mansion.Context, params *ExtractParams) error {
	if params.File == "" {
		return errors.New("extract: File must be specified")
	}
	if params.Dir == "" {
		return errors.New("extract: Dir must be specified")
	}

	consumer := params.Consumer

	file, err := eos.Open(params.File)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Opf("Extracting %s to %s", stats.Name(), params.Dir)

	archiveInfo, err := archive.Probe(&archive.TryOpenParams{
		File:     file,
		Consumer: consumer,
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}
	consumer.Opf("Using %s", archiveInfo.Features)

	ex, err := archiveInfo.GetExtractor(file, consumer)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	ex.SetConsumer(consumer)

	startTime := time.Now()

	sink := &savior.FolderSink{
		Directory: params.Dir,
	}

	comm.StartProgress()
	res, err := ex.Resume(nil, sink)
	comm.EndProgress()

	if err != nil {
		return errors.Wrap(err, 0)
	}

	duration := time.Since(startTime)
	bytesPerSec := float64(res.Size()) / duration.Seconds()
	consumer.Statf("Overall extraction speed: %s/s", humanize.IBytes(uint64(bytesPerSec)))

	return nil
}
