package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/dustin/go-humanize"
	"gopkg.in/kothar/brotli-go.v0/enc"

	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
)

func diff(target string, source string, recipe string, brotliQuality int) {
	if *appArgs.verbose {
		Logf("Computing TLC signature of %s", target)
	}

	targetContainer, err := tlc.Walk(target, pwr.BlockSize)
	must(err)

	sourceContainer, err := tlc.Walk(source, pwr.BlockSize)
	must(err)

	StartProgress()
	signature, err := pwr.ComputeDiffSignature(targetContainer, Progress)
	EndProgress()
	must(err)

	// index + weak + strong
	sigBytes := len(signature) * (4 + 16)
	if *appArgs.verbose {
		Logf("Signature size: %s", humanize.Bytes(uint64(sigBytes)))
	}

	brotliParams := enc.NewBrotliParams()
	brotliParams.SetQuality(brotliQuality)

	patchWriter, err := os.Create(recipe)
	must(err)

	StartProgress()
	err = pwr.WriteRecipe(patchWriter, sourceContainer, targetContainer, signature, Progress, brotliParams)
	must(err)
	EndProgress()

	if *diffArgs.verify {
		tmpDir, err := ioutil.TempDir(os.TempDir(), "megadiff")
		must(err)
		defer os.RemoveAll(tmpDir)

		Logf("Verifying recipe by rebuilding source in %s", tmpDir)
		apply(recipe, target, tmpDir)

		tmpInfo, err := tlc.Walk(tmpDir, pwr.BlockSize)
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
