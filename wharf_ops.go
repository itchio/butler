package main

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/sync"
	"github.com/itchio/wharf/tlc"
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
		} else {
			comm.Logf("Reading signature from file %s", target)
			signatureReader, err := os.Open(target)
			must(err)
			targetContainer, targetSignature, err = pwr.ReadSignature(signatureReader)
			must(err)
			must(signatureReader.Close())
		}

	}

	sourceContainer, err := tlc.Walk(source, filterDirs)
	must(err)

	recipeWriter, err := os.Create(recipe)
	must(err)
	defer recipeWriter.Close()

	signaturePath := recipe + ".sig"
	signatureWriter, err := os.Create(signaturePath)
	must(err)
	defer signatureWriter.Close()

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

	comm.StartProgress()
	must(dctx.WriteRecipe(recipeWriter, signatureWriter))
	comm.EndProgress()

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
	must(actx.ApplyRecipe(recipeReader))
	comm.EndProgress()

	comm.Debugf("Rebuilt source in %s", output)
}

func sign() {

}

func verify(signature string, output string) {
	comm.Dief("Verify: stub")
}
