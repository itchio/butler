package main

import (
	"bytes"

	"io/ioutil"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/pools/nullpool"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
)

func rediff(target string, source string) {
	must(doRediff(target, source))
}

func doRediff(target string, source string) error {
	// workdir := filepath.Join(".", "workdir")

	targetContainer, err := tlc.WalkAny(target, filterPaths)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	sourceContainer, err := tlc.WalkAny(source, filterPaths)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	targetPool := fspool.New(targetContainer, source)
	sourcePool := fspool.New(sourceContainer, source)

	patchBuffer := new(bytes.Buffer)

	targetSignature, err := pwr.ComputeSignature(targetContainer, targetPool, comm.NewStateConsumer())

	compression := butlerCompressionSettings()

	comm.Opf("Diffing...")

	comm.StartProgress()
	dc := &pwr.DiffContext{
		TargetContainer: targetContainer,
		TargetSignature: targetSignature,

		SourceContainer: sourceContainer,
		Pool:            sourcePool,
		Compression:     &compression,

		Consumer: comm.NewStateConsumer(),
	}

	sigBuffer := new(bytes.Buffer)

	err = dc.WritePatch(patchBuffer, sigBuffer)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	comm.EndProgress()

	comm.Statf("Original patch: %s", humanize.IBytes(uint64(patchBuffer.Len())))

	err = ioutil.WriteFile("patch.pwr", patchBuffer.Bytes(), 0644)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	rc := &pwr.RediffContext{
		TargetPool: targetPool,
		SourcePool: sourcePool,

		ForceMapAll: true,
		Partitions:  6,

		Consumer: comm.NewStateConsumer(),
	}

	comm.Opf("Analyzing patch...")
	err = rc.AnalyzePatch(bytes.NewReader(patchBuffer.Bytes()))
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Statf("Found %d diff mappings", len(rc.DiffMappings))

	newPatchBuffer := new(bytes.Buffer)

	comm.Opf("Optimizing patch...")

	comm.StartProgress()
	err = rc.OptimizePatch(bytes.NewReader(patchBuffer.Bytes()), newPatchBuffer)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	comm.EndProgress()

	comm.Statf("New patch: %s", humanize.IBytes(uint64(newPatchBuffer.Len())))

	err = ioutil.WriteFile("newpatch.pwr", newPatchBuffer.Bytes(), 0644)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	whitelist := make(map[int64]bool)
	for sourceIndex := range rc.DiffMappings {
		whitelist[sourceIndex] = true
	}

	sigInfo, err := pwr.ReadSignature(bytes.NewReader(sigBuffer.Bytes()))
	if err != nil {
		return errors.Wrap(err, 0)
	}

	validatingPool := &pwr.ValidatingPool{
		Pool:      nullpool.New(sigInfo.Container),
		Container: sigInfo.Container,
		Signature: sigInfo,
	}

	comm.Opf("Validating patch...")

	comm.StartProgress()
	actx := &pwr.ApplyContext{
		TargetPool: targetPool,
		OutputPool: validatingPool,
		Consumer:   comm.NewStateConsumer(),

		SourceIndexWhiteList: whitelist,
	}

	actx.ApplyPatch(bytes.NewReader(newPatchBuffer.Bytes()))
	comm.EndProgress()

	comm.Statf("Patch validates!")

	return nil
}
