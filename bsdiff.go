package main

import (
	"os"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/bsdiff"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

func cmdBsdiff(target string, source string, patch string) {
	must(doCmdBsdiff(target, source, patch))
}

func doCmdBsdiff(target string, source string, patch string) error {
	targetReader, err := os.Open(target)
	if err != nil {
		return err
	}

	defer targetReader.Close()

	targetStats, err := targetReader.Stat()
	if err != nil {
		return err
	}

	sourceReader, err := os.Open(source)
	if err != nil {
		return err
	}

	defer sourceReader.Close()

	sourceStats, err := sourceReader.Stat()
	if err != nil {
		return err
	}

	comm.Opf("Diffing %s (%s) and %s (%s)...", target, humanize.IBytes(uint64(targetStats.Size())), source,
		humanize.IBytes(uint64(sourceStats.Size())))

	patchWriter, err := os.Create(patch)
	if err != nil {
		return err
	}

	wctx := wire.NewWriteContext(patchWriter)

	err = wctx.WriteMagic(pwr.PatchMagic)
	if err != nil {
		return err
	}

	compression := butlerCompressionSettings()

	err = wctx.WriteMessage(&pwr.PatchHeader{
		Compression: &compression,
	})
	if err != nil {
		return err
	}

	wctx, err = pwr.CompressWire(wctx, &compression)
	if err != nil {
		return err
	}

	targetContainer := &tlc.Container{}
	targetContainer.Files = []*tlc.File{
		&tlc.File{
			Path: target,
			Size: targetStats.Size(),
		},
	}

	err = wctx.WriteMessage(targetContainer)
	if err != nil {
		return err
	}

	sourceContainer := &tlc.Container{}
	sourceContainer.Files = []*tlc.File{
		&tlc.File{
			Path: source,
			Size: sourceStats.Size(),
		},
	}

	err = wctx.WriteMessage(sourceContainer)
	if err != nil {
		return err
	}

	err = wctx.WriteMessage(&pwr.SyncHeader{
		FileIndex: 0,
	})
	if err != nil {
		return err
	}

	err = wctx.WriteMessage(&pwr.SyncOp{
		Type:      pwr.SyncOp_BSDIFF,
		FileIndex: 0,
	})
	if err != nil {
		return err
	}

	startTime := time.Now()

	comm.StartProgress()

	dc := bsdiff.DiffContext{}

	if *appArgs.memstats {
		dc.DebugMem = true
	}

	err = dc.Do(targetReader, sourceReader, wctx.WriteMessage, comm.NewStateConsumer())
	if err != nil {
		return err
	}

	comm.EndProgress()

	err = wctx.WriteMessage(&pwr.SyncOp{
		Type: pwr.SyncOp_HEY_YOU_DID_IT,
	})
	if err != nil {
		return err
	}

	err = wctx.Close()
	if err != nil {
		return err
	}

	patchStats, err := os.Lstat(patch)
	if err != nil {
		return err
	}

	duration := time.Since(startTime)
	perSec := float64(sourceStats.Size()) / duration.Seconds()

	relToNew := 100.0 * float64(patchStats.Size()) / float64(sourceStats.Size())

	comm.Statf("Processed %s @ %s / s, total %s", humanize.IBytes(uint64(sourceStats.Size())), humanize.IBytes(uint64(perSec)), duration)
	comm.Statf("Wrote %s patch (%.2f%% of total size) to %s", humanize.IBytes(uint64(patchStats.Size())), relToNew, patch)

	return nil
}
