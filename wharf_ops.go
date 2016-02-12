package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
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
var ignoredDirs = []string{
	".git",
	".cvs",
	".svn",
}

func filterDirs(fileInfo os.FileInfo) bool {
	name := fileInfo.Name()
	for _, dir := range ignoredDirs {
		if strings.HasPrefix(name, dir) {
			return false
		}
	}

	return true
}

func diff(target string, source string, recipe string, brotliQuality int) {
	startTime := time.Now()

	var targetSignature []sync.BlockHash
	var targetContainer *tlc.Container

	if target == "/dev/null" {
		targetContainer = &tlc.Container{}
	} else {
		targetInfo, err := os.Lstat(target)
		must(err)

		if targetInfo.IsDir() {
			comm.Logf("Computing signature of %s", target)
			targetContainer, err = tlc.Walk(target, filterDirs)
			must(err)

			comm.StartProgress()
			targetSignature, err = pwr.ComputeSignature(targetContainer, target, comm.NewStateConsumer())
			comm.EndProgress()
			must(err)

			{
				prettySize := humanize.Bytes(uint64(targetContainer.Size))
				perSecond := humanize.Bytes(uint64(float64(targetContainer.Size) / time.Since(startTime).Seconds()))
				comm.Logf("Signed %s @ %s/s", prettySize, perSecond)
			}
		} else {
			comm.Logf("Reading signature from file %s", target)
			signatureReader, err := os.Open(target)
			must(err)
			targetContainer, targetSignature, err = pwr.ReadSignature(signatureReader)
			must(err)
			must(signatureReader.Close())
		}

	}

	startTime = time.Now()

	sourceContainer, err := tlc.Walk(source, filterDirs)
	must(err)

	recipeWriter, err := os.Create(recipe)
	must(err)
	defer recipeWriter.Close()

	signaturePath := recipe + ".sig"
	signatureWriter, err := os.Create(signaturePath)
	must(err)
	defer signatureWriter.Close()

	recipeCounter := counter.NewWriter(recipeWriter)
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

	comm.Logf("Computing differences with %s", source)
	comm.StartProgress()
	must(dctx.WriteRecipe(recipeCounter, signatureCounter))
	comm.EndProgress()

	{
		prettySize := humanize.Bytes(uint64(sourceContainer.Size))
		prettyRecipeSize := humanize.Bytes(uint64(recipeCounter.Count()))
		perSecond := humanize.Bytes(uint64(float64(sourceContainer.Size) / time.Since(startTime).Seconds()))

		percReused := 100.0 * float64(dctx.ReusedBytes) / float64(dctx.FreshBytes+dctx.ReusedBytes)
		prettyFreshSize := humanize.Bytes(uint64(dctx.FreshBytes))
		percOfNew := float64(sourceContainer.Size) / float64(recipeCounter.Count())

		comm.Logf("Processed %s @ %s/s", prettySize, perSecond)
		comm.Logf("%s recipe (%.1fx smaller than new)", prettyRecipeSize, percOfNew)
		comm.Logf("%.2f%% re-used, %s fresh, compression: %s", percReused, prettyFreshSize, dctx.Compression.ToString())
	}

	if *diffArgs.verify {
		tmpDir, err := ioutil.TempDir(os.TempDir(), "pwr")
		must(err)
		defer os.RemoveAll(tmpDir)

		apply(recipe, target, tmpDir)

		verify(signaturePath, tmpDir)
	}
}

func apply(recipe string, target string, output string) {
	comm.Logf("Recreating new version into %s", output)
	startTime := time.Now()

	recipeReader, err := os.Open(recipe)
	must(err)

	actx := &pwr.ApplyContext{
		TargetPath: target,
		OutputPath: output,

		Consumer: comm.NewStateConsumer(),
	}

	comm.StartProgress()
	must(actx.ApplyRecipe(recipeReader))
	comm.EndProgress()

	container := actx.SourceContainer
	prettySize := humanize.Bytes(uint64(container.Size))
	perSecond := humanize.Bytes(uint64(float64(container.Size) / time.Since(startTime).Seconds()))
	comm.Logf("Rebuilt %s @ %s", prettySize, perSecond)
}

func sign(output string, signature string) {
	comm.Logf("Creating signature for %s", output)
	startTime := time.Now()

	container, err := tlc.Walk(output, nil)
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

	elapsedTime := time.Since(startTime)
	prettySize := humanize.Bytes(uint64(container.Size))
	perSecond := humanize.Bytes(uint64(float64(container.Size) / elapsedTime.Seconds()))
	comm.Logf("Hashed %s in %s (%s/s)", prettySize, elapsedTime.String(), perSecond)
}

func verify(signature string, output string) {
	comm.Logf("Verifying %s using %s", output, signature)
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

	if len(hashes) != len(refHashes) {
		must(fmt.Errorf("Expected %d blocks, got %d.", len(refHashes), len(hashes)))
	}

	for i, refHash := range refHashes {
		hash := hashes[i]

		if refHash.WeakHash != hash.WeakHash {
			comm.Dief("At block %d, expected weak hash %x, got %x", i, refHash.WeakHash, hash.WeakHash)
		}

		if !bytes.Equal(refHash.StrongHash, hash.StrongHash) {
			comm.Dief("At block %d, expected weak hash %x, got %x", i, refHash.StrongHash, hash.StrongHash)
		}
	}

	elapsedTime := time.Since(startTime)
	prettySize := humanize.Bytes(uint64(refContainer.Size))
	perSecond := humanize.Bytes(uint64(float64(refContainer.Size) / elapsedTime.Seconds()))
	comm.Logf("Verified %s in %s (%s/s)", prettySize, elapsedTime.String(), perSecond)
}
