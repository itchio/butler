package main

import (
	"archive/zip"
	"io/ioutil"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/pools/zippool"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/sync"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

func diff(target string, source string, patch string, compression pwr.CompressionSettings) {
	must(doDiff(target, source, patch, compression))
}

func doDiff(target string, source string, patch string, compression pwr.CompressionSettings) error {
	startTime := time.Now()

	// var targetSignature []sync.BlockHash
	// var targetContainer *tlc.Container

	targetSignature := &pwr.SignatureInfo{}

	if target == "/dev/null" {
		targetSignature.Container = &tlc.Container{}
	} else {
		targetInfo, err := os.Lstat(target)
		if err != nil {
			return errors.Wrap(err, 1)
		}

		if targetInfo.IsDir() {
			comm.Opf("Hashing %s", target)
			targetSignature.Container, err = tlc.Walk(target, filterPaths)
			if err != nil {
				return errors.Wrap(err, 1)
			}

			comm.StartProgress()
			targetPool := fspool.New(targetSignature.Container, target)
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
		} else {
			signatureReader, err := os.Open(target)
			if err != nil {
				return errors.Wrap(err, 1)
			}

			targetSignature, err = pwr.ReadSignature(signatureReader)
			if err != nil {
				if err != wire.ErrFormat {
					return errors.Wrap(err, 1)
				}

				_, err = signatureReader.Seek(0, os.SEEK_SET)
				if err != nil {
					return errors.Wrap(err, 1)
				}

				var stats os.FileInfo
				stats, err = os.Lstat(target)
				if err != nil {
					return errors.Wrap(err, 1)
				}

				var zr *zip.Reader
				zr, err = zip.NewReader(signatureReader, stats.Size())
				if err != nil {
					return errors.Wrap(err, 1)
				}

				targetSignature.Container, err = tlc.WalkZip(zr, filterPaths)
				if err != nil {
					return errors.Wrap(err, 1)
				}
				comm.Opf("Walking archive (%s)", targetSignature.Container.Stats())

				comm.StartProgress()
				targetPool := zippool.New(targetSignature.Container, zr)
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
			} else {
				comm.Opf("Read signature from %s", target)
			}

			err = signatureReader.Close()
			if err != nil {
				return errors.Wrap(err, 1)
			}
		}

	}

	startTime = time.Now()

	var sourceContainer *tlc.Container
	var sourcePool sync.Pool
	if source == "/dev/null" {
		sourceContainer = &tlc.Container{}
		sourcePool = fspool.New(sourceContainer, source)
	} else {
		var err error
		stats, err := os.Lstat(source)
		if err != nil {
			return errors.Wrap(err, 1)
		}

		if stats.IsDir() {
			sourceContainer, err = tlc.Walk(source, filterPaths)
			if err != nil {
				return errors.Wrap(err, 1)
			}
			sourcePool = fspool.New(sourceContainer, source)
		} else {
			sourceReader, err := os.Open(source)
			if err != nil {
				return errors.Wrap(err, 1)
			}
			defer sourceReader.Close()

			zr, err := zip.NewReader(sourceReader, stats.Size())
			if err != nil {
				return errors.Wrap(err, 1)
			}
			sourceContainer, err = tlc.WalkZip(zr, filterPaths)
			if err != nil {
				return errors.Wrap(err, 1)
			}

			sourcePool = zippool.New(sourceContainer, zr)
		}
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
