package singlediff

import (
	"os"
	"path/filepath"
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"

	"github.com/itchio/wharf/bsdiff"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/wire"

	"github.com/itchio/headway/united"

	"github.com/pkg/errors"

	"github.com/itchio/lake/tlc"
)

var args = struct {
	old         string
	new         string
	output      string
	partitions  int
	concurrency int
	measureMem  bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("singlediff", "(Advanced) generate a wharf patch with bsdiff between two files")
	cmd.Arg("old", "Old file").Required().StringVar(&args.old)
	cmd.Arg("new", "New file").Required().StringVar(&args.new)
	cmd.Flag("output", "Patch file to write").Short('o').Required().StringVar(&args.output)
	cmd.Flag("partitions", "Number of partitions to use").Default("1").IntVar(&args.partitions)
	cmd.Flag("concurrency", "Suffix sort concurrency").Default("1").IntVar(&args.concurrency)
	cmd.Flag("measuremem", "Measure memory usage").BoolVar(&args.measureMem)
	ctx.Register(cmd, func(ctx *mansion.Context) {
		ctx.Must(do(ctx))
	})
}

func do(ctx *mansion.Context) error {
	consumer := comm.NewStateConsumer()

	if filepath.IsAbs(args.old) {
		return errors.Errorf("%s: singlediff only works with relative paths", args.old)
	}
	if filepath.IsAbs(args.new) {
		return errors.Errorf("%s: singlediff only works with relative paths", args.new)
	}

	oldfile, err := os.Open(args.old)
	if err != nil {
		return err
	}

	oldstats, err := oldfile.Stat()
	if err != nil {
		return err
	}

	newfile, err := os.Open(args.new)
	if err != nil {
		return err
	}

	newstats, err := newfile.Stat()
	if err != nil {
		return err
	}

	targetContainer := &tlc.Container{}
	targetContainer.Size = oldstats.Size()
	targetContainer.Files = []*tlc.File{
		&tlc.File{
			Mode:   0o644,
			Offset: 0,
			Size:   oldstats.Size(),
			Path:   args.old,
		},
	}

	sourceContainer := &tlc.Container{}
	sourceContainer.Size = newstats.Size()
	sourceContainer.Files = []*tlc.File{
		&tlc.File{
			Mode:   0o644,
			Offset: 0,
			Size:   newstats.Size(),
			Path:   args.new,
		},
	}

	consumer.Infof("Before: %s (%s)", united.FormatBytes(targetContainer.Size), targetContainer.Stats())
	consumer.Infof(" After: %s (%s)", united.FormatBytes(sourceContainer.Size), sourceContainer.Stats())

	writer, err := os.Create(args.output)
	if err != nil {
		return err
	}

	rawPatchWire := wire.NewWriteContext(writer)
	err = rawPatchWire.WriteMagic(pwr.PatchMagic)
	if err != nil {
		return nil
	}

	ctxCompression := ctx.CompressionSettings()
	compression := &ctxCompression
	consumer.Infof("Using compression settings: %s", compression)

	header := &pwr.PatchHeader{
		Compression: compression,
	}

	err = rawPatchWire.WriteMessage(header)
	if err != nil {
		return nil
	}

	patchWire, err := pwr.CompressWire(rawPatchWire, compression)
	if err != nil {
		return nil
	}

	err = patchWire.WriteMessage(targetContainer)
	if err != nil {
		return err
	}

	err = patchWire.WriteMessage(sourceContainer)
	if err != nil {
		return err
	}

	syncHeader := &pwr.SyncHeader{
		FileIndex: 0,
		Type:      pwr.SyncHeader_BSDIFF,
	}
	err = patchWire.WriteMessage(syncHeader)
	if err != nil {
		return err
	}

	bsdiffHeader := &pwr.BsdiffHeader{
		TargetIndex: 0,
	}
	err = patchWire.WriteMessage(bsdiffHeader)
	if err != nil {
		return err
	}

	consumer.Opf("Suffix sort concurrency: %d, partitions: %d", args.concurrency, args.partitions)
	bdc := &bsdiff.DiffContext{
		SuffixSortConcurrency: args.concurrency,
		Partitions:            args.partitions,
		MeasureMem:            args.measureMem,

		Stats: &bsdiff.DiffStats{},
	}

	startTime := time.Now()

	comm.StartProgress()
	err = bdc.Do(oldfile, newfile, patchWire.WriteMessage, consumer)
	comm.EndProgress()
	if err != nil {
		return err
	}

	err = patchWire.WriteMessage(&pwr.SyncOp{
		Type: pwr.SyncOp_HEY_YOU_DID_IT,
	})
	if err != nil {
		return err
	}

	err = patchWire.Close()
	if err != nil {
		return err
	}

	outStats, err := os.Stat(args.output)
	if err != nil {
		return err
	}

	duration := time.Since(startTime)
	perSec := united.FormatBPS(outStats.Size(), duration)

	consumer.Statf("Wrote %s patch to %s @ %s (%s total)",
		united.FormatBytes(outStats.Size()),
		args.output,
		perSec,
		duration,
	)

	consumer.Statf("Spent %s sorting", bdc.Stats.TimeSpentSorting)
	consumer.Statf("Spent %s scanning", bdc.Stats.TimeSpentScanning)
	return nil
}
