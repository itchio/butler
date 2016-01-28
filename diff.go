package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/kothar/brotli-go.v0/enc"

	"github.com/dustin/go-humanize"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
)

func diff(target string, source string, recipe string, brotliQuality int) {
	if *appArgs.verbose {
		Logf("Computing TLC signature of %s", target)
	}

	targetInfo, err := tlc.Walk(target, pwr.BlockSize)
	must(err)

	sourceInfo, err := tlc.Walk(source, pwr.BlockSize)
	must(err)

	sourceReader := sourceInfo.NewReader(source)
	defer sourceReader.Close()

	StartProgress()
	signature, err := pwr.ComputeSignature(target, targetInfo, Progress)
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
	err = pwr.WriteRecipe(patchWriter, sourceInfo, sourceReader, targetInfo, signature, Progress, brotliParams)
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
