package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pools"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
	"github.com/itchio/wharf/wsync"
)

func diff(target string, source string, patch string, compression pwr.CompressionSettings) {
	must(doDiff(target, source, patch, compression))
}

func doDiff(target string, source string, patch string, compression pwr.CompressionSettings) error {
	var err error

	startTime := time.Now()

	targetSignature := &pwr.SignatureInfo{}

	targetSignature.Container, err = tlc.WalkDirOrArchive(target, filterPaths)
	if err != nil {
		// Signature file perhaps?
		var signatureReader io.ReadCloser

		signatureReader, err = os.Open(target)
		if err != nil {
			if errors.Is(err, wire.ErrFormat) {
				return fmt.Errorf("unrecognized target %s (not a container, not a signature file)", target)
			}
			return errors.Wrap(err, 1)
		}

		targetSignature, err = pwr.ReadSignature(signatureReader)
		comm.Opf("Read signature from %s", target)

		err = signatureReader.Close()
		if err != nil {
			return errors.Wrap(err, 1)
		}
	} else {
		// Container (dir, archive, etc.)
		comm.Opf("Hashing %s", target)

		comm.StartProgress()
		var targetPool wsync.Pool
		targetPool, err = pools.New(targetSignature.Container, target)
		if err != nil {
			return errors.Wrap(err, 1)
		}

		targetSignature.Hashes, err = pwr.ComputeSignature(targetSignature.Container, targetPool, comm.NewStateConsumer())
		comm.EndProgress()
		if err != nil {
			return errors.Wrap(err, 1)
		}

		{
			prettySize := humanize.IBytes(uint64(targetSignature.Container.Size))
			perSecond := humanize.IBytes(uint64(float64(targetSignature.Container.Size) / time.Since(startTime).Seconds()))
			comm.Statf("%s (%s) @ %s/s\n", prettySize, targetSignature.Container.Stats(), perSecond)
		}
	}

	startTime = time.Now()

	var sourceContainer *tlc.Container
	sourceContainer, err = tlc.WalkDirOrArchive(source, filterPaths)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	var sourcePool wsync.Pool
	sourcePool, err = pools.New(sourceContainer, source)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	patchWriter, err := os.Create(patch)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	defer patchWriter.Close()

	signaturePath := patch + ".sig"
	signatureWriter, err := os.Create(signaturePath)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	defer signatureWriter.Close()

	patchCounter := counter.NewWriter(patchWriter)
	signatureCounter := counter.NewWriter(signatureWriter)

	dctx := &pwr.DiffContext{
		SourceContainer: sourceContainer,
		Pool:            sourcePool,

		TargetContainer: targetSignature.Container,
		TargetSignature: targetSignature.Hashes,

		Consumer:    comm.NewStateConsumer(),
		Compression: &compression,
	}

	comm.Opf("Diffing %s", source)
	comm.StartProgress()
	err = dctx.WritePatch(patchCounter, signatureCounter)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	comm.EndProgress()

	totalDuration := time.Since(startTime)
	{
		prettySize := humanize.IBytes(uint64(sourceContainer.Size))
		perSecond := humanize.IBytes(uint64(float64(sourceContainer.Size) / totalDuration.Seconds()))
		comm.Statf("%s (%s) @ %s/s\n", prettySize, sourceContainer.Stats(), perSecond)
	}

	if *diffArgs.verify {
		tmpDir, err := ioutil.TempDir("", "pwr")
		if err != nil {
			return errors.Wrap(err, 1)
		}
		defer os.RemoveAll(tmpDir)

		apply(patch, target, tmpDir, false, signaturePath)
	}

	{
		prettyPatchSize := humanize.IBytes(uint64(patchCounter.Count()))
		percReused := 100.0 * float64(dctx.ReusedBytes) / float64(dctx.FreshBytes+dctx.ReusedBytes)
		relToNew := 100.0 * float64(patchCounter.Count()) / float64(sourceContainer.Size)
		prettyFreshSize := humanize.IBytes(uint64(dctx.FreshBytes))

		comm.Statf("Re-used %.2f%% of old, added %s fresh data", percReused, prettyFreshSize)
		comm.Statf("%s patch (%.2f%% of the full size) in %s", prettyPatchSize, relToNew, totalDuration)
	}

	return nil
}
