package main

import (
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/kothar/brotli-go.v0/enc"

	"github.com/dustin/go-humanize"

	"github.com/itchio/butler/comm"
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
		comm.Logf("Computing TLC signature of %s", target)
	}

	targetContainer, err := tlc.Walk(target, filterDirs)
	must(err)

	sourceContainer, err := tlc.Walk(source, filterDirs)
	must(err)

	comm.StartProgress()
	targetSignature, err := pwr.ComputeDiffSignature(targetContainer, target, comm.NewStateConsumer())
	comm.EndProgress()
	must(err)

	// index + weak + strong
	sigBytes := len(targetSignature) * (4 + 16)
	comm.Debugf("Target signature size: %s", humanize.Bytes(uint64(sigBytes)))

	rawRecipeWriter, err := os.Create(recipe)
	must(err)

	recipeWriter := rawRecipeWriter

	signaturePath := recipe + ".sig"
	signatureWriter, err := os.Create(signaturePath)
	must(err)

	dctx := &pwr.DiffContext{
		SourceContainer: sourceContainer,
		SourcePath:      source,

		TargetContainer: targetContainer,
		TargetSignature: targetSignature,

		Consumer: comm.NewStateConsumer(),
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

	comm.StartProgress()
	err = dctx.WriteRecipe(recipeWriter, sourceSignatureWriter)
	must(err)
	comm.EndProgress()

	err = signatureWriter.Close()
	must(err)

	if *diffArgs.verify {
		tmpDir, err := ioutil.TempDir(os.TempDir(), "pwr")
		must(err)
		defer os.RemoveAll(tmpDir)

		comm.Logf("Verifying recipe by rebuilding source in %s", tmpDir)
		apply(recipe, target, tmpDir)

		verify(signaturePath, tmpDir)
	}
}

func apply(recipe string, target string, output string) {
	recipeReader, err := os.Open(recipe)
	must(err)

	actx := &pwr.ApplyContext{
		TargetPath: target,
		OutputPath: output,

		Consumer: comm.NewStateConsumer(),
	}

	comm.StartProgress()
	err = actx.ApplyRecipe(recipeReader)
	comm.EndProgress()
	must(err)

	comm.Debugf("Rebuilt source in %s", output)
}

func verify(signature string, output string) {
	comm.Logf("Verify: stub")
}
