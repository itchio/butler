package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/sync"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

// TODO: make this customizable
var ignoredPaths = []string{
	".git",
	".hg",
	".svn",
	".DS_Store",
	"._*",
	"Thumbs.db",
}

func filterPaths(fileInfo os.FileInfo) bool {
	name := fileInfo.Name()
	for _, pattern := range ignoredPaths {
		match, err := filepath.Match(pattern, name)
		if err != nil {
			panic(fmt.Sprintf("Malformed ignore pattern '%s': %s", pattern, err.Error()))
		}
		if match {
			if *appArgs.verbose {
				fmt.Printf("Ignoring '%s' because of pattern '%s'\n", fileInfo.Name(), pattern)
			}
			return false
		}
	}

	return true
}

func diff(target string, source string, patch string, brotliQuality int) {
	startTime := time.Now()

	var targetSignature []sync.BlockHash
	var targetContainer *tlc.Container

	if target == "/dev/null" {
		targetContainer = &tlc.Container{}
	} else {
		targetInfo, err := os.Lstat(target)
		must(err)

		if targetInfo.IsDir() {
			comm.Opf("Hashing %s", target)
			targetContainer, err = tlc.Walk(target, filterPaths)
			must(err)

			comm.StartProgress()
			targetSignature, err = pwr.ComputeSignature(targetContainer, target, comm.NewStateConsumer())
			comm.EndProgress()
			must(err)

			{
				prettySize := humanize.Bytes(uint64(targetContainer.Size))
				perSecond := humanize.Bytes(uint64(float64(targetContainer.Size) / time.Since(startTime).Seconds()))
				comm.Statf("%s (%s) @ %s/s\n", prettySize, targetContainer.Stats(), perSecond)
			}
		} else {
			comm.Opf("Reading signature from %s", target)
			signatureReader, err := os.Open(target)
			must(err)
			targetContainer, targetSignature, err = pwr.ReadSignature(signatureReader)
			must(err)
			must(signatureReader.Close())
		}

	}

	startTime = time.Now()

	sourceContainer, err := tlc.Walk(source, filterPaths)
	must(err)

	patchWriter, err := os.Create(patch)
	must(err)
	defer patchWriter.Close()

	signaturePath := patch + ".sig"
	signatureWriter, err := os.Create(signaturePath)
	must(err)
	defer signatureWriter.Close()

	patchCounter := counter.NewWriter(patchWriter)
	signatureCounter := counter.NewWriter(signatureWriter)

	dctx := &pwr.DiffContext{
		SourceContainer: sourceContainer,
		SourcePath:      source,

		TargetContainer: targetContainer,
		TargetSignature: targetSignature,

		Consumer: comm.NewStateConsumer(),
		Compression: &pwr.CompressionSettings{
			Algorithm: pwr.CompressionAlgorithm_BROTLI,
			Quality:   int32(*diffArgs.quality),
		},
	}

	comm.Opf("Diffing %s", source)
	comm.StartProgress()
	must(dctx.WritePatch(patchCounter, signatureCounter))
	comm.EndProgress()

	{
		prettySize := humanize.Bytes(uint64(sourceContainer.Size))
		perSecond := humanize.Bytes(uint64(float64(sourceContainer.Size) / time.Since(startTime).Seconds()))
		comm.Statf("%s (%s) @ %s/s\n", prettySize, sourceContainer.Stats(), perSecond)
	}

	if *diffArgs.verify {
		tmpDir, err := ioutil.TempDir(os.TempDir(), "pwr")
		must(err)
		defer os.RemoveAll(tmpDir)

		apply(patch, target, tmpDir, false)

		verify(signaturePath, tmpDir)
	}

	{
		prettyPatchSize := humanize.Bytes(uint64(patchCounter.Count()))
		percReused := 100.0 * float64(dctx.ReusedBytes) / float64(dctx.FreshBytes+dctx.ReusedBytes)
		relToNew := 100.0 * float64(patchCounter.Count()) / float64(sourceContainer.Size)
		prettyFreshSize := humanize.Bytes(uint64(dctx.FreshBytes))

		comm.Statf("Re-used %.2f%% of old, added %s fresh data", percReused, prettyFreshSize)
		comm.Statf("%s patch (%.2f%% of the full size)", prettyPatchSize, relToNew)
	}
}

func apply(patch string, target string, output string, inplace bool) {
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
	must(err)

	actx := &pwr.ApplyContext{
		TargetPath: target,
		OutputPath: output,
		InPlace:    inplace,

		Consumer: comm.NewStateConsumer(),
	}

	comm.StartProgress()
	must(actx.ApplyPatch(patchReader))
	comm.EndProgress()

	container := actx.SourceContainer
	prettySize := humanize.Bytes(uint64(container.Size))
	perSecond := humanize.Bytes(uint64(float64(container.Size) / time.Since(startTime).Seconds()))
	comm.Statf("%s (%s) @ %s/s (touched %d files)\n", prettySize, container.Stats(), perSecond, actx.TouchedFiles)
}

func sign(output string, signature string) {
	comm.Opf("Creating signature for %s", output)
	startTime := time.Now()

	container, err := tlc.Walk(output, filterPaths)
	must(err)

	signatureWriter, err := os.Create(signature)
	must(err)

	compression := pwr.CompressionDefault()

	rawSigWire := wire.NewWriteContext(signatureWriter)
	rawSigWire.WriteMagic(pwr.SignatureMagic)

	rawSigWire.WriteMessage(&pwr.SignatureHeader{
		Compression: compression,
	})

	sigWire, err := pwr.CompressWire(rawSigWire, compression)
	must(err)
	sigWire.WriteMessage(container)

	comm.StartProgress()
	err = pwr.ComputeSignatureToWriter(container, output, comm.NewStateConsumer(), func(hash sync.BlockHash) error {
		return sigWire.WriteMessage(&pwr.BlockHash{
			WeakHash:   hash.WeakHash,
			StrongHash: hash.StrongHash,
		})
	})
	comm.EndProgress()
	must(err)

	must(sigWire.Close())

	prettySize := humanize.Bytes(uint64(container.Size))
	perSecond := humanize.Bytes(uint64(float64(container.Size) / time.Since(startTime).Seconds()))
	comm.Statf("%s (%s) @ %s/s\n", prettySize, container.Stats(), perSecond)
}

func verify(signature string, output string) {
	comm.Opf("Verifying %s", output)
	startTime := time.Now()

	signatureReader, err := os.Open(signature)
	must(err)
	defer signatureReader.Close()

	refContainer, refHashes, err := pwr.ReadSignature(signatureReader)
	must(err)

	comm.StartProgress()
	hashes, err := pwr.ComputeSignature(refContainer, output, comm.NewStateConsumer())
	comm.EndProgress()
	must(err)

	err = pwr.CompareHashes(refHashes, hashes)
	if err != nil {
		comm.Logf(err.Error())
		comm.Dief("Some checks failed after checking %d block.", len(refHashes))
	}

	prettySize := humanize.Bytes(uint64(refContainer.Size))
	perSecond := humanize.Bytes(uint64(float64(refContainer.Size) / time.Since(startTime).Seconds()))
	comm.Statf("%s (%s) @ %s/s\n", prettySize, refContainer.Stats(), perSecond)
}
