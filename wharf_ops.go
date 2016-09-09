package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"

	"golang.org/x/crypto/sha3"

	"gopkg.in/kothar/brotli-go.v0/enc"

	"github.com/Datadog/zstd"
	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/splitfunc"
	"github.com/itchio/wharf/sync"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

func diff(target string, source string, patch string, compression pwr.CompressionSettings) {
	must(doDiff(target, source, patch, compression))
}

func doDiff(target string, source string, patch string, compression pwr.CompressionSettings) error {
	startTime := time.Now()

	var targetSignature []sync.BlockHash
	var targetContainer *tlc.Container

	if target == "/dev/null" {
		targetContainer = &tlc.Container{}
	} else {
		targetInfo, err := os.Lstat(target)
		if err != nil {
			return errors.Wrap(err, 1)
		}

		if targetInfo.IsDir() {
			comm.Opf("Hashing %s", target)
			targetContainer, err = tlc.Walk(target, filterPaths)
			if err != nil {
				return errors.Wrap(err, 1)
			}

			comm.StartProgress()
			targetSignature, err = pwr.ComputeSignature(targetContainer, targetContainer.NewFilePool(target), comm.NewStateConsumer())
			comm.EndProgress()
			if err != nil {
				return errors.Wrap(err, 1)
			}

			{
				prettySize := humanize.Bytes(uint64(targetContainer.Size))
				perSecond := humanize.Bytes(uint64(float64(targetContainer.Size) / time.Since(startTime).Seconds()))
				comm.Statf("%s (%s) @ %s/s\n", prettySize, targetContainer.Stats(), perSecond)
			}
		} else {
			signatureReader, err := os.Open(target)
			if err != nil {
				return errors.Wrap(err, 1)
			}

			targetContainer, targetSignature, err = pwr.ReadSignature(signatureReader)
			if err != nil {
				if err != wire.ErrFormat {
					return errors.Wrap(err, 1)
				}

				_, err = signatureReader.Seek(0, os.SEEK_SET)
				if err != nil {
					return errors.Wrap(err, 1)
				}

				stats, err := os.Lstat(target)
				if err != nil {
					return errors.Wrap(err, 1)
				}

				zr, err := zip.NewReader(signatureReader, stats.Size())
				if err != nil {
					return errors.Wrap(err, 1)
				}

				targetContainer, err = tlc.WalkZip(zr, filterPaths)
				if err != nil {
					return errors.Wrap(err, 1)
				}
				comm.Opf("Walking archive (%s)", targetContainer.Stats())

				comm.StartProgress()
				targetSignature, err = pwr.ComputeSignature(targetContainer, targetContainer.NewZipPool(zr), comm.NewStateConsumer())
				comm.EndProgress()
				if err != nil {
					return errors.Wrap(err, 1)
				}

				{
					prettySize := humanize.Bytes(uint64(targetContainer.Size))
					perSecond := humanize.Bytes(uint64(float64(targetContainer.Size) / time.Since(startTime).Seconds()))
					comm.Statf("%s (%s) @ %s/s\n", prettySize, targetContainer.Stats(), perSecond)
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
	var sourcePool sync.FilePool
	if source == "/dev/null" {
		sourceContainer = &tlc.Container{}
		sourcePool = sourceContainer.NewFilePool(source)
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
			sourcePool = sourceContainer.NewFilePool(source)
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

			sourcePool = sourceContainer.NewZipPool(zr)
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
		FilePool:        sourcePool,

		TargetContainer: targetContainer,
		TargetSignature: targetSignature,

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
		prettySize := humanize.Bytes(uint64(sourceContainer.Size))
		perSecond := humanize.Bytes(uint64(float64(sourceContainer.Size) / totalDuration.Seconds()))
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
		prettyPatchSize := humanize.Bytes(uint64(patchCounter.Count()))
		percReused := 100.0 * float64(dctx.ReusedBytes) / float64(dctx.FreshBytes+dctx.ReusedBytes)
		relToNew := 100.0 * float64(patchCounter.Count()) / float64(sourceContainer.Size)
		prettyFreshSize := humanize.Bytes(uint64(dctx.FreshBytes))

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

	comm.Opf("Patching %s", output)
	startTime := time.Now()

	patchReader, err := os.Open(patch)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	actx := &pwr.ApplyContext{
		TargetPath:        target,
		OutputPath:        output,
		InPlace:           inplace,
		SignatureFilePath: sigpath,

		Consumer: comm.NewStateConsumer(),
	}

	comm.StartProgress()
	err = actx.ApplyPatch(patchReader)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	comm.EndProgress()

	container := actx.SourceContainer
	prettySize := humanize.Bytes(uint64(container.Size))
	perSecond := humanize.Bytes(uint64(float64(container.Size) / time.Since(startTime).Seconds()))

	if actx.InPlace {
		comm.Statf("patched %d, kept %d, deleted %d (%s stage)", actx.TouchedFiles, actx.NoopFiles, actx.DeletedFiles, humanize.Bytes(uint64(actx.StageSize)))
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
		container.FixPermissions(container.NewFilePool(output))
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
	err = pwr.ComputeSignatureToWriter(container, container.NewFilePool(output), comm.NewStateConsumer(), func(hash sync.BlockHash) error {
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

	prettySize := humanize.Bytes(uint64(container.Size))
	perSecond := humanize.Bytes(uint64(float64(container.Size) / time.Since(startTime).Seconds()))
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

	refContainer, refHashes, err := pwr.ReadSignature(signatureReader)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.StartProgress()
	hashes, err := pwr.ComputeSignature(refContainer, refContainer.NewFilePool(output), comm.NewStateConsumer())
	comm.EndProgress()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = pwr.CompareHashes(refHashes, hashes, refContainer)
	if err != nil {
		comm.Logf(err.Error())
		comm.Dief("Some checks failed after checking %d block.", len(refHashes))
	}

	prettySize := humanize.Bytes(uint64(refContainer.Size))
	perSecond := humanize.Bytes(uint64(float64(refContainer.Size) / time.Since(startTime).Seconds()))
	comm.Statf("%s (%s) @ %s/s\n", prettySize, refContainer.Stats(), perSecond)

	return nil
}

func probe(target string) {
	must(doProbe(target))
}

func doProbe(target string) error {
	targetContainer, err := tlc.Walk(target, filterPaths)
	if err != nil {
		return err
	}

	bigBlockSize := 4 * 1024 * 1024 // 4MB blocks
	comm.Opf("Compressing %s as %s blocks", targetContainer.Stats(), humanize.Bytes(uint64(bigBlockSize)))

	pool := targetContainer.NewFilePool(target)

	brotliParams := enc.NewBrotliParams()
	brotliParams.SetQuality(*appArgs.compressionQuality)

	processedSize := int64(0)
	totalCompressedSize := int64(0)
	shake128 := sha3.NewShake128()

	seenBlocks := make(map[string]bool)
	duplicateBlocks := int64(0)
	hbuf := make([]byte, 32)

	comm.StartProgress()

	makeCompressedWriter := func(w io.Writer) io.WriteCloser {
		switch *probeArgs.algo {
		case "brotli":
			return enc.NewBrotliWriter(brotliParams, w)
		case "zstd":
			return zstd.NewWriterLevel(w, *appArgs.compressionQuality)
		default:
			panic(fmt.Sprintf("unknown compression algo %s", *probeArgs.algo))
		}
	}

	startTime := time.Now()

	for i := 0; i < len(targetContainer.Files); i++ {
		r, err := pool.GetReader(int64(i))
		if err != nil {
			return err
		}

		s := bufio.NewScanner(r)
		s.Buffer(make([]byte, bigBlockSize), 0)
		s.Split(splitfunc.New(bigBlockSize))

		for s.Scan() {
			shake128.Reset()
			cw := counter.NewWriter(shake128)
			bw := makeCompressedWriter(cw)

			block := s.Bytes()
			bw.Write(block)
			bw.Close()

			_, err := io.ReadFull(shake128, hbuf)
			if err != nil {
				return err
			}
			key := fmt.Sprintf("shake128-32/%d/%x", len(block), hbuf)
			if seenBlocks[key] {
				duplicateBlocks++
				comm.Debugf("%s block: duplicate", humanize.Bytes(uint64(len(block))))
			} else {
				seenBlocks[key] = true
				comm.Debugf("%s block compressed to %s bytes", humanize.Bytes(uint64(len(block))), humanize.Bytes(uint64(cw.Count())))
				totalCompressedSize += cw.Count()
			}

			processedSize += int64(len(block))
			comm.Progress(float64(processedSize) / float64(targetContainer.Size))
		}
	}

	comm.EndProgress()

	perSecond := humanize.Bytes(uint64(float64(targetContainer.Size) / time.Since(startTime).Seconds()))
	comm.Statf("%s => %s (%.3f) via %s blocks (%d duplicates), %s-q%d @ %s/s",
		humanize.Bytes(uint64(targetContainer.Size)),
		humanize.Bytes(uint64(totalCompressedSize)),
		float64(totalCompressedSize)/float64(targetContainer.Size),
		humanize.Bytes(uint64(bigBlockSize)),
		duplicateBlocks,
		*probeArgs.algo,
		*appArgs.compressionQuality,
		perSecond)

	comm.Opf("Now as a single archive...")

	startTime = time.Now()
	comm.StartProgress()

	cw := counter.NewWriter(nil)
	bw := makeCompressedWriter(cw)

	offset := int64(0)

	for i := 0; i < len(targetContainer.Files); i++ {
		r, err := pool.GetReader(int64(i))
		if err != nil {
			return err
		}

		cr := counter.NewReaderCallback(func(count int64) {
			comm.Progress(float64(offset+count) / float64(targetContainer.Size))
		}, r)

		_, err = io.Copy(bw, cr)
		if err != nil {
			return err
		}

		offset += targetContainer.Files[i].Size
	}

	bw.Close()
	totalCompressedSize = cw.Count()

	comm.EndProgress()

	perSecond = humanize.Bytes(uint64(float64(targetContainer.Size) / time.Since(startTime).Seconds()))
	comm.Statf("%s => %s (%.3f) as single archive, %s-q%d @ %s/s",
		humanize.Bytes(uint64(targetContainer.Size)),
		humanize.Bytes(uint64(totalCompressedSize)),
		float64(totalCompressedSize)/float64(targetContainer.Size),
		*probeArgs.algo,
		*appArgs.compressionQuality,
		perSecond)

	comm.Statf("(note: single-archive doesn't compute any hashes, so it's faster)")

	return nil
}
