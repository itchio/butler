package unzip

import (
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/archiver"
	"github.com/itchio/wharf/eos"
	"github.com/pkg/errors"
)

var args = struct {
	file        *string
	dir         *string
	resumeFile  *string
	dryRun      *bool
	concurrency *int
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("unzip", "Extract a .zip file").Hidden()
	args.file = cmd.Arg("file", "Path of the .zip archive to extract").Required().String()
	args.dir = cmd.Flag("dir", "An optional directory to which to extract files (defaults to CWD)").Default(".").Short('d').String()
	args.resumeFile = cmd.Flag("resume-file", "When given, write current progress to this file, resume from last location if it exists.").Short('f').String()
	args.dryRun = cmd.Flag("dry-run", "Do not write anything to disk").Short('n').Bool()
	args.concurrency = cmd.Flag("concurrency", "Number of workers to use (negative for numbers of CPUs - j)").Default("-1").Int()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, &UnzipParams{
		File: *args.file,
		Dir:  *args.dir,

		ResumeFile:  *args.resumeFile,
		DryRun:      *args.dryRun,
		Concurrency: *args.concurrency,
	}))
}

type UnzipParams struct {
	File string
	Dir  string

	ResumeFile  string
	DryRun      bool
	Concurrency int
}

func Do(ctx *mansion.Context, params *UnzipParams) error {
	if params.File == "" {
		return errors.New("unzip: File must be specified")
	}
	if params.Dir == "" {
		return errors.New("unzip: Dir must be specified")
	}

	comm.Opf("Extracting zip %s to %s", eos.Redact(params.File), params.Dir)

	var zipUncompressedSize int64

	onEntryDone := func(path string) {
		comm.Result(&mansion.FileExtractedResult{
			Type: "entry",
			Path: path,
		})
	}

	settings := archiver.ExtractSettings{
		Consumer:   comm.NewStateConsumer(),
		ResumeFrom: params.ResumeFile,
		OnUncompressedSizeKnown: func(uncompressedSize int64) {
			zipUncompressedSize = uncompressedSize
			comm.StartProgressWithTotalBytes(uncompressedSize)
		},
		DryRun:      params.DryRun,
		OnEntryDone: onEntryDone,
		Concurrency: params.Concurrency,
	}

	startTime := time.Now()

	res, err := archiver.ExtractPath(params.File, params.Dir, settings)
	comm.EndProgress()

	duration := time.Since(startTime)
	bytesPerSec := float64(zipUncompressedSize) / duration.Seconds()

	if err != nil {
		return errors.Wrap(err, "unzipping")
	}
	comm.Logf("Extracted %d dirs, %d files, %d symlinks, %s at %s/s", res.Dirs, res.Files, res.Symlinks,
		humanize.IBytes(uint64(zipUncompressedSize)), humanize.IBytes(uint64(bytesPerSec)))

	return nil
}
