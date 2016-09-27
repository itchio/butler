package main

import (
	"archive/zip"
	"io/ioutil"
	"os"
	"path"
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

func apply(patch string, target string, output string, inplace bool, sigpath string) {
	must(doApply(patch, target, output, inplace, sigpath))
}

func doApply(patch string, target string, output string, inplace bool, sigpath string) error {
	if output == "" {
		output = target
	}

	target = path.Clean(target)
	output = path.Clean(output)
	if output == target {
		if !inplace {
			comm.Dief("Refusing to destructively patch %s without --inplace", output)
		}
	}

	if sigpath == "" {
		comm.Opf("Patching %s", output)
	} else {
		comm.Opf("Patching %s with validation", output)
	}

	startTime := time.Now()

	patchReader, err := os.Open(patch)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	var signature *pwr.SignatureInfo
	if sigpath != "" {
		sigReader, sigErr := os.Open(sigpath)
		if sigErr != nil {
			return errors.Wrap(sigErr, 1)
		}
		defer sigReader.Close()

		signature, sigErr = pwr.ReadSignature(sigReader)
		if sigErr != nil {
			return errors.Wrap(sigErr, 1)
		}
	}

	actx := &pwr.ApplyContext{
		TargetPath: target,
		OutputPath: output,
		InPlace:    inplace,
		Signature:  signature,

		Consumer: comm.NewStateConsumer(),
	}

	comm.StartProgress()
	err = actx.ApplyPatch(patchReader)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	comm.EndProgress()

	container := actx.SourceContainer
	prettySize := humanize.IBytes(uint64(container.Size))
	perSecond := humanize.IBytes(uint64(float64(container.Size) / time.Since(startTime).Seconds()))

	if actx.InPlace {
		comm.Statf("patched %d, kept %d, deleted %d (%s stage)", actx.TouchedFiles, actx.NoopFiles, actx.DeletedFiles, humanize.IBytes(uint64(actx.StageSize)))
	}
	comm.Statf("%s (%s) @ %s/s\n", prettySize, container.Stats(), perSecond)

	return nil
}

func sign(output string, signature string, compression pwr.CompressionSettings, fixPerms bool) {
	must(doSign(output, signature, compression, fixPerms))
}

func doSign(output string, signature string, compression pwr.CompressionSettings, fixPerms bool) error {
	comm.Opf("Creating signature for %s", output)
	startTime := time.Now()

	container, err := tlc.Walk(output, filterPaths)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if fixPerms {
		container.FixPermissions(fspool.New(container, output))
	}

	signatureWriter, err := os.Create(signature)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	rawSigWire := wire.NewWriteContext(signatureWriter)
	rawSigWire.WriteMagic(pwr.SignatureMagic)

	rawSigWire.WriteMessage(&pwr.SignatureHeader{
		Compression: &compression,
	})

	sigWire, err := pwr.CompressWire(rawSigWire, &compression)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	sigWire.WriteMessage(container)

	comm.StartProgress()
	err = pwr.ComputeSignatureToWriter(container, fspool.New(container, output), comm.NewStateConsumer(), func(hash sync.BlockHash) error {
		return sigWire.WriteMessage(&pwr.BlockHash{
			WeakHash:   hash.WeakHash,
			StrongHash: hash.StrongHash,
		})
	})
	comm.EndProgress()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = sigWire.Close()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	prettySize := humanize.IBytes(uint64(container.Size))
	perSecond := humanize.IBytes(uint64(float64(container.Size) / time.Since(startTime).Seconds()))
	comm.Statf("%s (%s) @ %s/s\n", prettySize, container.Stats(), perSecond)

	return nil
}

func verify(signature string, output string) {
	must(doVerify(signature, output))
}

func doVerify(signature string, output string) error {
	comm.Opf("Verifying %s", output)
	startTime := time.Now()

	signatureReader, err := os.Open(signature)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	defer signatureReader.Close()

	refSignature, err := pwr.ReadSignature(signatureReader)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	refContainer := refSignature.Container
	refHashes := refSignature.Hashes

	comm.StartProgress()
	hashes, err := pwr.ComputeSignature(refContainer, fspool.New(refContainer, output), comm.NewStateConsumer())
	comm.EndProgress()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = pwr.CompareHashes(refHashes, hashes, refContainer)
	if err != nil {
		comm.Logf(err.Error())
		comm.Dief("Some checks failed after checking %d blocks.", len(refHashes))
	}

	prettySize := humanize.IBytes(uint64(refContainer.Size))
	perSecond := humanize.IBytes(uint64(float64(refContainer.Size) / time.Since(startTime).Seconds()))
	comm.Statf("%s (%s) @ %s/s\n", prettySize, refContainer.Stats(), perSecond)

	return nil
}
