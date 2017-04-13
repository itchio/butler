package main

import (
	"os"
	"time"

	"path/filepath"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
)

func indexZip(file string) {
	must(doIndexZip(file))
}

func doIndexZip(file string) error {
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

	comm.Opf("Creating index for %s", file)

	ext := filepath.Ext(file)
	withoutExt := file[:len(file)-len(ext)]
	targetFile := withoutExt + ".pzi"

	w, err := os.Create(targetFile)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer w.Close()

	comm.Opf("Writing index to %s", targetFile)

	startTime := time.Now()

	comm.StartProgress()

	err = ic.Index(r, stats.Size(), w)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.EndProgress()

	duration := time.Since(startTime)
	bytesPerSec := float64(ic.TotalCompressedSize) / duration.Seconds()

	targetSize, err := w.Seek(0, os.SEEK_CUR)
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
