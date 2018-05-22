package rediff

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/httpkit/progress"
	"github.com/itchio/savior/filesource"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/pwr"
)

var args = struct {
	patch       string
	old         string
	new         string
	output      string
	partitions  int
	concurrency int
	quality     int32
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("rediff", "(Advanced) optimize a default wharf patch")
	cmd.Flag("patch", "Patch file").Required().StringVar(&args.patch)
	cmd.Flag("old", "Old file").Required().StringVar(&args.old)
	cmd.Flag("new", "New file").Required().StringVar(&args.new)
	cmd.Flag("output", "Optimized patch file to write").Short('o').Required().StringVar(&args.output)
	cmd.Flag("partitions", "Number of partitions to use").Default(fmt.Sprintf("%d", runtime.NumCPU()/2)).IntVar(&args.partitions)
	cmd.Flag("concurrency", "Suffix sort concurrency").Default("-1").IntVar(&args.concurrency)
	cmd.Flag("rediff-quality", "Quality of compression to use").Default("1").Int32Var(&args.quality)
	ctx.Register(cmd, func(ctx *mansion.Context) {
		ctx.Must(do(ctx))
	})
}

func do(ctx *mansion.Context) error {
	consumer := comm.NewStateConsumer()

	compression := &pwr.CompressionSettings{
		Algorithm: pwr.CompressionAlgorithm_BROTLI,
		Quality:   args.quality,
	}
	consumer.Opf("Writing with compression %s", compression)

	rc := pwr.RediffContext{
		Consumer: consumer,

		SuffixSortConcurrency: args.concurrency,
		Partitions:            args.partitions,
		Compression:           compression,
	}

	patchSource, err := filesource.Open(args.patch)

	err = rc.AnalyzePatch(patchSource)
	if err != nil {
		return err
	}

	consumer.Statf("Analyzed.")
	consumer.Infof("Before: %s (%s)", progress.FormatBytes(rc.TargetContainer.Size), rc.TargetContainer.Stats())
	consumer.Infof(" After: %s (%s)", progress.FormatBytes(rc.SourceContainer.Size), rc.SourceContainer.Stats())

	rc.TargetPool = fspool.New(rc.TargetContainer, args.old)
	rc.SourcePool = fspool.New(rc.SourceContainer, args.new)

	_, err = patchSource.Resume(nil)
	if err != nil {
		return err
	}

	patchWriter, err := os.Create(args.output)
	if err != nil {
		return err
	}

	startTime := time.Now()

	consumer.Opf("Optimizing...")

	comm.StartProgress()
	err = rc.OptimizePatch(patchSource, patchWriter)
	comm.EndProgress()
	if err != nil {
		return err
	}

	outputStats, err := os.Stat(args.output)
	if err != nil {
		return err
	}

	duration := time.Since(startTime)
	perSec := progress.FormatBPS(rc.SourceContainer.Size, duration)
	consumer.Statf("Wrote %s patch, processed at %s / s (%s total)",
		progress.FormatBytes(outputStats.Size()),
		perSec,
		progress.FormatDuration(duration),
	)

	return nil
}
