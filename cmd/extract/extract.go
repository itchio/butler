package extract

import (
	"time"

	"github.com/itchio/boar"
	"github.com/itchio/boar/szextractor"

	"github.com/itchio/savior"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"

	"github.com/itchio/headway/state"
	"github.com/itchio/headway/united"

	"github.com/pkg/errors"
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

	fetch7zLibsCmd := ctx.App.Command("fetch-7z-libs", "Fetch 7-zip dependencies").Hidden()
	ctx.Register(fetch7zLibsCmd, doFetch7zLibs)
}

func doFetch7zLibs(ctx *mansion.Context) {
	ctx.Must(szextractor.EnsureDeps(comm.NewStateConsumer()))
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

	file, err := eos.Open(params.File, option.WithConsumer(consumer))
	if err != nil {
		return errors.Wrap(err, "opening archive file")
	}
	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		return errors.Wrap(err, "stat'ing archive file")
	}

	consumer.Opf("Extracting (%s) to (%s)", stats.Name(), params.Dir)

	archiveInfo, err := boar.Probe(boar.ProbeParams{
		File:     file,
		Consumer: consumer,
	})
	if err != nil {
		return errors.Wrap(err, "probing archive")
	}

	var extractSize int64

	startTime := time.Now()

	if archiveInfo.Strategy == boar.StrategyDmg {
		return errors.New("Extracting DMGs is deprecated, sorry!")
	} else {
		consumer.Opf("Using %s", archiveInfo.Features)
		ex, err := archiveInfo.GetExtractor(file, consumer)
		if err != nil {
			return errors.Wrap(err, "getting extractor for archive")
		}

		if szex, ok := ex.(szextractor.SzExtractor); ok {
			consumer.Opf("Archive format: (%s)", szex.GetFormat())
		}

		progressStarted := false
		// create a copy of consumer that starts progress on the first
		// Progress() call
		delayedConsumer := *consumer
		delayedConsumer.OnProgress = func(progress float64) {
			if !progressStarted {
				comm.StartProgress()
				progressStarted = true
			}
			consumer.Progress(progress)
		}

		ex.SetConsumer(&delayedConsumer)

		sink := &savior.FolderSink{
			Directory: params.Dir,
		}

		res, err := ex.Resume(nil, sink)
		comm.EndProgress()

		if err != nil {
			return errors.Wrap(err, "extracting archive")
		}
		extractSize = res.Size()

		consumer.Statf("Extracted %s", res.Stats())
	}

	duration := time.Since(startTime)
	consumer.Statf("Overall extraction speed: %s", united.FormatBPS(extractSize, duration))

	return nil
}
