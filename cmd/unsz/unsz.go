package unsz

import (
	"time"

	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"

	"github.com/itchio/savior"

	"github.com/itchio/boar/szextractor"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"

	"github.com/itchio/headway/state"
	"github.com/itchio/headway/united"

	"github.com/pkg/errors"
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
	}))
}

type UnszParams struct {
	File string
	Dir  string

	Consumer *state.Consumer
}

func Do(ctx *mansion.Context, params *UnszParams) error {
	if params.File == "" {
		return errors.New("unsz: File must be specified")
	}
	if params.Dir == "" {
		return errors.New("unsz: Dir must be specified")
	}

	consumer := params.Consumer

	file, err := eos.Open(params.File, option.WithConsumer(consumer))
	if err != nil {
		return errors.WithStack(err)
	}
	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		return errors.WithStack(err)
	}

	consumer.Opf("Extracting %s to %s", stats.Name(), params.Dir)

	ex, err := szextractor.New(file, consumer)
	if err != nil {
		return errors.WithStack(err)
	}

	startTime := time.Now()

	sink := &savior.FolderSink{
		Directory: params.Dir,
	}

	comm.StartProgress()
	res, err := ex.Resume(nil, sink)
	comm.EndProgress()

	if err != nil {
		return errors.WithStack(err)
	}

	duration := time.Since(startTime)
	consumer.Statf("Overall extraction speed: %s/s", united.FormatBPS(res.Size(), duration))

	return nil
}
