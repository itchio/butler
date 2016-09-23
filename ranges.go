package main

import (
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/pools/blockpool"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/pwr/genie"
)

func ranges(manifest string, patch string, newManifest string) {
	must(doRanges(manifest, patch, newManifest))
}

func doRanges(manifest string, patch string, newManifest string) error {
	patchReader, err := os.Open(patch)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	bigBlockSize := blockpool.BigBlockSize

	g := &genie.Genie{
		BlockSize: bigBlockSize,
	}
	err = g.ParseHeader(patchReader)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	targetContainer := g.TargetContainer
	sourceContainer := g.SourceContainer

	requiredOldBlocks := make(blockpool.BlockFilter)
	requiredOldBlocksList := []blockpool.BlockLocation{}

	freshNewBlocks := make(blockpool.BlockFilter)
	comps := make(chan *genie.Composition)

	newBlockHashes := blockpool.NewBlockHashMap()

	// read the old build's manifest
	manifestReader, err := os.Open(manifest)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	defer manifestReader.Close()

	manContainer, blockHashes, err := blockpool.ReadManifest(manifestReader)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	blockAddresses, err := blockHashes.ToAddressMap(manContainer, pwr.HashAlgorithm_SHAKE128_32)

	go func() {
		for comp := range comps {
			reuse := false

			if len(comp.Origins) == 1 {
				switch origin := comp.Origins[0].(type) {
				case *genie.BlockOrigin:
					if origin.Offset%bigBlockSize == 0 {
						newLoc := blockpool.BlockLocation{FileIndex: comp.FileIndex, BlockIndex: comp.BlockIndex}
						blockIndex := origin.Offset / bigBlockSize
						oldLoc := blockpool.BlockLocation{FileIndex: origin.FileIndex, BlockIndex: blockIndex}
						newBlockHashes.Set(newLoc, blockHashes.Get(oldLoc))
						reuse = true
					}
				}
			}

			if reuse {
				continue
			}

			freshNewBlocks.Set(blockpool.BlockLocation{FileIndex: comp.FileIndex, BlockIndex: comp.BlockIndex})
			for _, anyOrigin := range comp.Origins {
				switch origin := anyOrigin.(type) {
				case *genie.BlockOrigin:
					blockStart := origin.Offset / bigBlockSize
					blockEnd := (origin.Offset + origin.Size + bigBlockSize - 1) / bigBlockSize
					for j := blockStart; j < blockEnd; j++ {
						loc := blockpool.BlockLocation{FileIndex: origin.FileIndex, BlockIndex: j}
						if !requiredOldBlocks.Has(loc) {
							requiredOldBlocksList = append(requiredOldBlocksList, blockpool.BlockLocation{FileIndex: origin.FileIndex, BlockIndex: j})
						}
						requiredOldBlocks.Set(loc)
					}
				}
			}
		}
	}()

	err = g.ParseContents(comps)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.Statf("Old req'd blocks: %s", requiredOldBlocks.Stats(targetContainer))
	comm.Statf("Fresh new blocks: %s", freshNewBlocks.Stats(sourceContainer))

	blockAddresses, err = blockAddresses.TranslateFileIndices(manContainer, targetContainer)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	var source blockpool.Source

	source = &blockpool.DiskSource{
		BasePath:       "./blocks",
		BlockAddresses: blockAddresses,

		Container: targetContainer,
	}

	if *rangesArgs.inlatency > 0 {
		source = &blockpool.DelayedSource{
			Latency: time.Duration(*rangesArgs.inlatency) * time.Millisecond,
			Source:  source,
		}
	}

	if *rangesArgs.infilter {
		source = &blockpool.FilteringSource{
			Filter: requiredOldBlocks,
			Source: source,
		}
	}

	targetPool := &blockpool.BlockPool{
		Container: targetContainer,
		Upstream:  source,

		Consumer: comm.NewStateConsumer(),
	}

	actx := &pwr.ApplyContext{
		Consumer:   comm.NewStateConsumer(),
		TargetPool: targetPool,
	}

	var fanOutSink *blockpool.FanOutSink

	if *rangesArgs.writeToDisk {
		actx.OutputPath = "./out"
	} else {
		var subSink blockpool.Sink

		subSink = &blockpool.DiskSink{
			BasePath:    "./outblocks",
			Container:   sourceContainer,
			BlockHashes: newBlockHashes,
		}

		if *rangesArgs.outlatency > 0 {
			subSink = &blockpool.DelayedSink{
				Latency: time.Duration(*rangesArgs.outlatency) * time.Millisecond,
				Sink:    subSink,
			}
		}

		if *rangesArgs.outfilter {
			subSink = &blockpool.FilteringSink{
				Filter: freshNewBlocks,
				Sink:   subSink,
			}
		}

		errs := make(chan error)
		go func() {
			for sErr := range errs {
				comm.Dief("Fan out sink error: %s", sErr.Error())
			}
		}()

		fanOutSink, err = blockpool.NewFanOutSink(subSink, *rangesArgs.fanout)
		if err != nil {
			return errors.Wrap(err, 1)
		}
		fanOutSink.Start()

		actx.OutputPool = &blockpool.BlockPool{
			Container:  sourceContainer,
			Downstream: fanOutSink,
			Consumer:   comm.NewStateConsumer(),
		}
	}

	_, err = patchReader.Seek(0, os.SEEK_SET)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	startTime := time.Now()

	comm.StartProgress()
	err = actx.ApplyPatch(patchReader)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	comm.EndProgress()

	if fanOutSink != nil {
		err = fanOutSink.Close()
		if err != nil {
			return errors.Wrap(err, 1)
		}
	}

	totalTime := time.Since(startTime)
	comm.Statf("Processed in %s (%s/s)", totalTime, humanize.IBytes(uint64(float64(targetContainer.Size)/totalTime.Seconds())))

	comm.Opf("Writing manifest...")

	manifestWriter, err := os.Create(newManifest)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	compression := butlerCompressionSettings()
	err = blockpool.WriteManifest(manifestWriter, &compression, sourceContainer, newBlockHashes)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	return nil
}
