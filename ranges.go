package main

import (
	"fmt"
	"math"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

func ranges(patch string) {
	must(doRanges(patch))
}

func doRanges(patch string) error {
	patchStats, err := os.Lstat(patch)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	patchReader, err := os.Open(patch)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	rawPatchWire := wire.NewReadContext(patchReader)
	err = rawPatchWire.ExpectMagic(pwr.PatchMagic)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	header := &pwr.PatchHeader{}
	err = rawPatchWire.ReadMessage(header)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	patchWire, err := pwr.DecompressWire(rawPatchWire, header.Compression)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	targetContainer := &tlc.Container{}
	err = patchWire.ReadMessage(targetContainer)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	sourceContainer := &tlc.Container{}
	err = patchWire.ReadMessage(sourceContainer)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.Opf("Showing ranges for %s patch", humanize.IBytes(uint64(patchStats.Size())))
	comm.Statf("Old version: %s in %s", humanize.IBytes(uint64(targetContainer.Size)), targetContainer.Stats())
	comm.Statf("New version: %s in %s", humanize.IBytes(uint64(sourceContainer.Size)), sourceContainer.Stats())
	deltaOp := "+"
	if sourceContainer.Size < targetContainer.Size {
		deltaOp = "-"
	}
	delta := math.Abs(float64(sourceContainer.Size - targetContainer.Size))
	comm.Statf("Delta: %s%s (%s%.2f%%)", deltaOp, humanize.IBytes(uint64(delta)), deltaOp, delta/float64(targetContainer.Size)*100.0)
	comm.Log("")

	numDatas := 0
	numBlockRanges := 0
	blockSize := int64(pwr.BlockSize)
	unchangedBytes := int64(0)
	movedBytes := int64(0)
	freshBytes := int64(0)

	sh := &pwr.SyncHeader{}
	for fileIndex, sourceFile := range sourceContainer.Files {
		sh.Reset()
		err := patchWire.ReadMessage(sh)
		if err != nil {
			return errors.Wrap(err, 1)
		}

		if sh.FileIndex != int64(fileIndex) {
			fmt.Printf("expected fileIndex = %d, got fileIndex %d\n", fileIndex, sh.FileIndex)
			return errors.Wrap(pwr.ErrMalformedPatch, 1)
		}

		rop := &pwr.SyncOp{}

		err = (func() error {
			sourceOffset := int64(0)

			for {
				rop.Reset()
				pErr := patchWire.ReadMessage(rop)
				if pErr != nil {
					return errors.Wrap(pErr, 1)
				}

				switch rop.Type {
				case pwr.SyncOp_BLOCK_RANGE:
					targetOffset := blockSize * rop.BlockIndex
					targetFile := targetContainer.Files[rop.FileIndex]

					size := blockSize * rop.BlockSpan

					alignedSize := blockSize * (rop.BlockIndex + rop.BlockSpan)
					if alignedSize > targetFile.Size {
						size -= blockSize
						size += targetFile.Size % blockSize
					}

					if targetFile.Path == sourceFile.Path && targetOffset == sourceOffset {
						// comm.Statf("%d unchanged blocks %d bytes into %s", rop.BlockSpan, sourceOffset, targetFile.Path)
						unchangedBytes += size
					} else {
						movedBytes += size
					}

					numBlockRanges++
					sourceOffset += size
				case pwr.SyncOp_DATA:
					size := int64(len(rop.Data))
					sourceOffset += size
					freshBytes += size
					numDatas++
				case pwr.SyncOp_HEY_YOU_DID_IT:
					return nil
				}
			}
		})()
		if err != nil {
			return errors.Wrap(err, 1)
		}
	}

	comm.Statf("%d BlockRange ops, %d Data ops", numBlockRanges, numDatas)
	comm.Statf("Unchanged bytes: %s", humanize.IBytes(uint64(unchangedBytes)))
	comm.Statf("Moved bytes    : %s", humanize.IBytes(uint64(movedBytes)))
	comm.Statf("Fresh bytes    : %s", humanize.IBytes(uint64(freshBytes)))

	return nil
}
