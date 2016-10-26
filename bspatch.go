package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

func bspatch(patch string, target string, output string) {
	must(doBspatch(patch, target, output))
}

func doBspatch(patch string, target string, output string) error {
	targetReader, err := os.Open(target)
	if err != nil {
		return err
	}

	defer targetReader.Close()

	patchReader, err := os.Open(patch)
	if err != nil {
		return err
	}

	defer patchReader.Close()

	err = os.MkdirAll(filepath.Dir(output), 0755)
	if err != nil {
		return err
	}

	outputWriter, err := os.Create(output)
	if err != nil {
		return err
	}

	defer outputWriter.Close()

	rctx := wire.NewReadContext(patchReader)

	err = rctx.ExpectMagic(pwr.PatchMagic)
	if err != nil {
		return err
	}

	ph := &pwr.PatchHeader{}

	err = rctx.ReadMessage(ph)
	if err != nil {
		return err
	}

	compression := ph.GetCompression()

	rctx, err = pwr.DecompressWire(rctx, compression)
	if err != nil {
		return err
	}

	targetContainer := &tlc.Container{}
	err = rctx.ReadMessage(targetContainer)
	if err != nil {
		return err
	}

	sourceContainer := &tlc.Container{}
	err = rctx.ReadMessage(sourceContainer)
	if err != nil {
		return err
	}

	if len(targetContainer.Files) != 1 {
		return fmt.Errorf("expected only one file in target container")
	}

	if len(sourceContainer.Files) != 1 {
		return fmt.Errorf("expected only one file in source container")
	}

	sh := &pwr.SyncHeader{}
	err = rctx.ReadMessage(sh)
	if err != nil {
		return err
	}

	if sh.FileIndex != 0 {
		return fmt.Errorf("wrong sync header")
	}

	op := &pwr.SyncOp{}
	err = rctx.ReadMessage(op)
	if err != nil {
		return err
	}

	if op.Type != pwr.SyncOp_BSDIFF {
		return fmt.Errorf("expected bsdiff syncop")
	}

	if op.FileIndex != 0 {
		return fmt.Errorf("expected bsdiff syncop to operate on only file")
	}

	outputSize := sourceContainer.Files[op.FileIndex].Size

	err = pwr.BSPatch(targetReader, outputWriter, outputSize, rctx)
	if err != nil {
		return err
	}

	return nil
}
