package indexzip

import (
	"io"
	"os"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
)

var args = struct {
	file   *string
	output *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("index-zip", "Generate an index for a .zip file").Hidden()
	args.file = cmd.Arg("file", "Path of the .zip archive to index").Required().String()
	args.output = cmd.Flag("output", "Path to write the .pzi file to").Short('o').Default("index.pzi").String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, *args.file, *args.output))
}

func Do(ctx *mansion.Context, file string, output string) error {
	ic := &pwr.ZipIndexerContext{
		Consumer: comm.NewStateConsumer(),
	}

	r, err := eos.Open(file)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer r.Close()

	stats, err := r.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Opf("Creating index for %s", eos.Redact(file))

	w, err := os.Create(output)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer w.Close()

	comm.Opf("Writing index to %s", output)

	startTime := time.Now()

	comm.StartProgress()

	err = ic.Index(r, stats.Size(), w)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.EndProgress()

	duration := time.Since(startTime)
	bytesPerSec := float64(ic.TotalCompressedSize) / duration.Seconds()

	targetSize, err := w.Seek(0, io.SeekStart)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Statf("Indexed %s (%s uncompressed) @ %s/s",
		humanize.IBytes(uint64(ic.TotalCompressedSize)),
		humanize.IBytes(uint64(ic.TotalUncompressedSize)),
		humanize.IBytes(uint64(bytesPerSec)))
	comm.Statf("Index size: %s (%.2f%% of zip)",
		humanize.IBytes(uint64(targetSize)),
		float64(targetSize)/float64(stats.Size())*100.0)
	comm.Statf("%d segments total, largest segment is %s",
		ic.TotalSegments,
		humanize.IBytes(uint64(ic.LargestSegmentSize)))

	return nil
}
