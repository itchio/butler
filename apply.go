package main

import (
	"os"
	"path"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/pwr"
)

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
