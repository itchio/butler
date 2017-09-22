package main

import (
	"fmt"
	"path"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
)

func apply(patch string, target string, output string, inplace bool, signaturePath string, woundsPath string, healSpec string, stagePath string) {
	must(doApply(patch, target, output, inplace, signaturePath, woundsPath, healSpec, stagePath))
}

func doApply(patch string, target string, output string, inplace bool, signaturePath string, woundsPath string, healSpec string, stagePath string) error {
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

	if signaturePath == "" {
		comm.Opf("Patching %s", output)
	} else {
		comm.Opf("Patching %s with validation", output)
	}

	startTime := time.Now()

	patchReader, err := eos.Open(patch)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	var signature *pwr.SignatureInfo
	if signaturePath != "" {
		sigReader, sigErr := eos.Open(signaturePath)
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
		WoundsPath: woundsPath,
		StagePath:  stagePath,
		HealPath:   healSpec,

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
		statStr := ""
		if actx.Stats.TouchedFiles > 0 {
			statStr += fmt.Sprintf("patched %d, ", actx.Stats.TouchedFiles)
		}
		if actx.Stats.MovedFiles > 0 {
			statStr += fmt.Sprintf("renamed %d, ", actx.Stats.MovedFiles)
		}
		if actx.Stats.DeletedFiles > 0 {
			statStr += fmt.Sprintf("deleted %d, ", actx.Stats.DeletedFiles)
		}

		comm.Statf("%s (%s stage)", statStr, humanize.IBytes(uint64(actx.Stats.StageSize)))
	}
	comm.Statf("%s (%s) @ %s/s\n", prettySize, container.Stats(), perSecond)

	if actx.WoundsConsumer != nil && actx.WoundsConsumer.HasWounds() {
		extra := ""
		if actx.WoundsPath != "" {
			extra = fmt.Sprintf(" (written to %s)", actx.WoundsPath)
		}

		totalCorrupted := actx.WoundsConsumer.TotalCorrupted()

		verb := "has"
		totalHealed := int64(0)
		if healer, ok := actx.WoundsConsumer.(pwr.Healer); ok {
			verb = "had"
			totalHealed = healer.TotalHealed()
		}

		comm.Logf("Result %s wounds, %s corrupted data, %s healed%s", verb, humanize.IBytes(uint64(totalCorrupted)), humanize.IBytes(uint64(totalHealed)), extra)
	}

	return nil
}
