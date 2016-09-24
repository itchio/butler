package main

import (
	"os"
	"runtime"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/pools"
	"github.com/itchio/wharf/pools/blockpool"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
)

func split(target string, manifest string) {
	must(doSplit(target, manifest))
}

func doSplit(target string, manifest string) error {
	container, err := tlc.WalkDirOrArchive(target, filterPaths)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.Statf("Splitting %s in %s", humanize.IBytes(uint64(container.Size)), container.Stats())

	inPool, err := pools.New(container, target)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	blockDir := "blocks"

	var sink blockpool.Sink

	blockHashes := blockpool.NewBlockHashMap()
	sink = &blockpool.DiskSink{
		BasePath: blockDir,

		Container:   container,
		BlockHashes: blockHashes,
	}

	sink = &blockpool.CompressingSink{
		Sink: sink,
	}

	fanOutSink, err := blockpool.NewFanOutSink(sink, runtime.NumCPU()*2+1)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	fanOutSink.Start()

	sink = fanOutSink

	outPool := &blockpool.BlockPool{
		Container:  container,
		Downstream: sink,
	}

	startTime := time.Now()

	comm.StartProgress()

	err = pwr.CopyContainer(container, outPool, inPool, comm.NewStateConsumer())
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = fanOutSink.Close()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.EndProgress()

	duration := time.Since(startTime)
	perSec := humanize.IBytes(uint64(float64(container.Size) / duration.Seconds()))

	comm.Statf("Split %s in %s (%s/s)", humanize.IBytes(uint64(container.Size)), duration, perSec)

	// write manifest
	manifestWriter, err := os.Create(manifest)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	defer manifestWriter.Close()

	compression := butlerCompressionSettings()
	err = blockpool.WriteManifest(manifestWriter, &compression, container, blockHashes)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	return nil
}
