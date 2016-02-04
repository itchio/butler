package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

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

	brotliParams := enc.NewBrotliParams()
	brotliParams.SetQuality(brotliQuality)

	rawRecipeWriter, err := os.Create(recipe + ".br")
	must(err)

	recipeWriter := enc.NewBrotliWriter(brotliParams, rawRecipeWriter)

	rawSignatureWriter, err := os.Create(recipe + ".sig.br")
	must(err)

	dctx := &pwr.DiffContext{
		SourceContainer: sourceContainer,
		SourcePath:      source,

		TargetContainer: targetContainer,
		TargetSignature: targetSignature,
	}

	signatureWriter := enc.NewBrotliWriter(brotliParams, rawSignatureWriter)
	swc := wire.NewWriteContext(signatureWriter)

	err = swc.WriteMessage(&pwr.SignatureHeader{})
	must(err)

	sourceSignatureWriter := func(bl sync.BlockHash) error {
		// swc.WriteMessage(&pwr.BlockHash{
		// 	WeakHash:   bl.WeakHash,
		// 	StrongHash: bl.StrongHash,
		// })
		return nil
	}

	StartProgress()
	err = dctx.WriteRecipe(recipeWriter, Progress, brotliParams, sourceSignatureWriter)
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
	recipeReader, err := os.Open(recipe)
	must(err)

	StartProgress()
	err = pwr.ApplyRecipe(recipeReader, target, output, Progress)
	EndProgress()
	must(err)

	if *appArgs.verbose {
		Logf("Rebuilt source in %s", output)
	}
}
