package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/kothar/brotli-go.v0/dec"
	"gopkg.in/kothar/brotli-go.v0/enc"

	"github.com/dustin/go-humanize"

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
	if *appArgs.verbose {
		Logf("Computing TLC signature of %s", target)
	}

	targetContainer, err := tlc.Walk(target, filterDirs)
	must(err)

	sourceContainer, err := tlc.Walk(source, filterDirs)
	must(err)

	StartProgress()
	targetSignature, err := pwr.ComputeDiffSignature(targetContainer, target, Progress)
	EndProgress()
	must(err)

	// index + weak + strong
	sigBytes := len(targetSignature) * (4 + 16)
	if *appArgs.verbose {
		Logf("Target signature size: %s", humanize.Bytes(uint64(sigBytes)))
	}

	rawRecipeWriter, err := os.Create(recipe)
	must(err)

	// recipeWriter := enc.NewBrotliWriter(brotliParams, rawRecipeWriter)
	recipeWriter := rawRecipeWriter

	signatureWriter, err := os.Create(recipe + ".sig")
	must(err)

	dctx := &pwr.DiffContext{
		SourceContainer: sourceContainer,
		SourcePath:      source,

		TargetContainer: targetContainer,
		TargetSignature: targetSignature,
	}

	hswc := wire.NewWriteContext(signatureWriter)

	err = hswc.WriteMessage(&pwr.SignatureHeader{})
	must(err)

	brotliParams := enc.NewBrotliParams()
	brotliParams.SetQuality(1)

	signatureCompressedWriter := enc.NewBrotliWriter(brotliParams, signatureWriter)
	swc := wire.NewWriteContext(signatureCompressedWriter)

	sourceSignatureWriter := func(bl sync.BlockHash) error {
		swc.WriteMessage(&pwr.BlockHash{
			WeakHash:   bl.WeakHash,
			StrongHash: bl.StrongHash,
		})
		return nil
	}

	StartProgress()
	err = dctx.WriteRecipe(recipeWriter, Progress, sourceSignatureWriter)
	must(err)
	EndProgress()

	err = recipeWriter.Close()
	must(err)

	err = signatureWriter.Close()
	must(err)

	if *diffArgs.verify {
		tmpDir, err := ioutil.TempDir(os.TempDir(), "megadiff")
		must(err)
		defer os.RemoveAll(tmpDir)

		Logf("Verifying recipe by rebuilding source in %s", tmpDir)
		apply(recipe, target, tmpDir)

		tmpInfo, err := tlc.Walk(tmpDir, filterDirs)
		must(err)
		fmt.Printf("tmpInfo: %+v", tmpInfo)
	}
}

func apply(recipe string, target string, output string) {
	rawRecipeReader, err := os.Open(recipe)
	must(err)

	recipeReader := dec.NewBrotliReader(rawRecipeReader)
	must(err)
	// recipeReader := rawRecipeReader

	StartProgress()
	err = pwr.ApplyRecipe(recipeReader, target, output, Progress)
	EndProgress()
	must(err)

	if *appArgs.verbose {
		Logf("Rebuilt source in %s", output)
	}
}
