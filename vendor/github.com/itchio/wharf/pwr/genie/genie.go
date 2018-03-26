package genie

import (
	"fmt"

	"github.com/itchio/savior"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
	"github.com/pkg/errors"
)

type CompositionListener func(comp *Composition)

// A Genie analyzes a patch to figure out which parts of the target container
// are used to build individual blocks of the source container.
type Genie struct {
	BlockSize int64

	PatchWire *wire.ReadContext

	TargetContainer *tlc.Container
	SourceContainer *tlc.Container
}

// ParseHeader is the first step of the genie's operation - it reads both
// containers, leaving the caller a chance to use them later, when parsing
// the contents
func (g *Genie) ParseHeader(patchReader savior.SeekSource) error {
	rawPatchWire := wire.NewReadContext(patchReader)
	err := rawPatchWire.ExpectMagic(pwr.PatchMagic)
	if err != nil {
		return errors.WithStack(err)
	}

	header := &pwr.PatchHeader{}
	err = rawPatchWire.ReadMessage(header)
	if err != nil {
		return errors.WithStack(err)
	}

	patchWire, err := pwr.DecompressWire(rawPatchWire, header.Compression)
	if err != nil {
		return errors.WithStack(err)
	}
	g.PatchWire = patchWire

	g.TargetContainer = &tlc.Container{}
	err = patchWire.ReadMessage(g.TargetContainer)
	if err != nil {
		return errors.WithStack(err)
	}

	g.SourceContainer = &tlc.Container{}
	err = patchWire.ReadMessage(g.SourceContainer)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// ParseContents sends a Composition for each block of the source container
func (g *Genie) ParseContents(onComp CompositionListener) error {
	patchWire := g.PatchWire

	// for each file, the patch contains a SyncHeader followed by a series of
	// operations, always ending in HEY_YOU_DID_IT
	sh := &pwr.SyncHeader{}
	for fileIndex, f := range g.SourceContainer.Files {
		sh.Reset()
		err := patchWire.ReadMessage(sh)
		if err != nil {
			return errors.WithStack(err)
		}

		if sh.FileIndex != int64(fileIndex) {
			fmt.Printf("expected fileIndex = %d, got fileIndex %d\n", fileIndex, sh.FileIndex)
			return errors.WithStack(pwr.ErrMalformedPatch)
		}

		err = g.analyzeFile(patchWire, int64(fileIndex), f.Size, onComp)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (g *Genie) analyzeFile(patchWire *wire.ReadContext, fileIndex int64, fileSize int64, onComp CompositionListener) error {
	rop := &pwr.SyncOp{}

	smallBlockSize := int64(pwr.BlockSize)
	bigBlockSize := g.BlockSize

	comp := &Composition{
		FileIndex: int64(fileIndex),
	}

	// infinite loop, explicitly "break"'d out of
	for {
		rop.Reset()
		pErr := patchWire.ReadMessage(rop)
		if pErr != nil {
			return errors.WithStack(pErr)
		}

		switch rop.Type {
		case pwr.SyncOp_BLOCK_RANGE:
			// SyncOps operate in terms of small blocks, we want byte offsets
			bo := &BlockOrigin{
				FileIndex: rop.FileIndex,
				Offset:    rop.BlockIndex * smallBlockSize,
				Size:      rop.BlockSpan * smallBlockSize,
			}

			// As long as the block origin would span beyond the end of the
			// big block we're currently analyzing, split it into {A, B},
			// where A fits into the current big block, and B is the rest
			for comp.Size+bo.Size > bigBlockSize {
				truncatedSize := bigBlockSize - comp.Size

				// truncatedSize may be 0 if `comp.Size == bigBlockSize`, ie. comp already
				// explains all the contents of the current big block - in this case,
				// we keep this BlockOrigin intact for the next iteration of the loop
				// (during which comp.Size will == 0)
				if truncatedSize > 0 {
					// this is A
					comp.Append(&BlockOrigin{
						FileIndex: rop.FileIndex,
						Offset:    bo.Offset,
						Size:      truncatedSize,
					})

					// and bo becomes B
					bo.Offset += truncatedSize
					bo.Size -= truncatedSize
				}

				onComp(comp)

				// after sending over the composition, we allocate a new one - same file, next block
				// (sent comps should not be modified afterwards)
				comp = &Composition{
					FileIndex:  int64(fileIndex),
					BlockIndex: comp.BlockIndex + 1,
				}
			}

			// after all the splitting, there might still be some data left over
			// (that's smaller than bigBlockSize)
			if bo.Size > 0 {
				comp.Append(bo)
			}
		case pwr.SyncOp_DATA:
			// Data SyncOps are not aligned either in target or source. Since genie
			// works in byte offsets, this suits us just fine.
			fo := &FreshOrigin{
				Size: int64(len(rop.Data)),
			}

			for comp.Size+fo.Size > bigBlockSize {
				truncatedSize := bigBlockSize - comp.Size

				// only if we can fit some of the data in this block, otherwise, clear
				// the current comp, wait for next loop iteration where comp.Size will be 0
				if truncatedSize > 0 {
					comp.Append(&FreshOrigin{
						Size: truncatedSize,
					})

					fo.Size -= truncatedSize
				}

				onComp(comp)

				// allocate a new comp after sending, so we don't write to the previous one
				comp = &Composition{
					FileIndex:  int64(fileIndex),
					BlockIndex: comp.BlockIndex + 1,
				}
			}

			if fo.Size > 0 {
				comp.Append(fo)
			}
		case pwr.SyncOp_HEY_YOU_DID_IT:
			if comp.Size > 0 && fileSize > 0 {
				onComp(comp)
			}
			return nil
		}
	}
}
