package genie

import (
	"fmt"
	"io"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

type BlockOrigin struct {
	FileIndex int64
	Offset    int64
	Size      int64
}

func (bo *BlockOrigin) GetSize() int64 {
	return bo.Size
}

type FreshOrigin struct {
	Size int64
}

func (fo *FreshOrigin) GetSize() int64 {
	return fo.Size
}

type Origin interface {
	GetSize() int64
}

type Composition struct {
	FileIndex  int64
	BlockIndex int64
	Origins    []Origin
	Size       int64
}

func (comp *Composition) Append(origin Origin) {
	comp.Origins = append(comp.Origins, origin)
	comp.Size += origin.GetSize()
}

func (comp *Composition) String() string {
	res := fmt.Sprintf("file %d, block %d (%s) is composed of: ", comp.FileIndex, comp.BlockIndex, humanize.IBytes(uint64(comp.Size)))
	for i, origin := range comp.Origins {
		if i > 0 {
			res += ", "
		}
		res += fmt.Sprintf("%+v", origin)
	}
	return res
}

type Genie struct {
	BlockSize int64

	PatchWire *wire.ReadContext

	TargetContainer *tlc.Container
	SourceContainer *tlc.Container
}

func (g *Genie) ParseHeader(patchReader io.Reader) error {
	rawPatchWire := wire.NewReadContext(patchReader)
	err := rawPatchWire.ExpectMagic(pwr.PatchMagic)
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
	g.PatchWire = patchWire

	g.TargetContainer = &tlc.Container{}
	err = patchWire.ReadMessage(g.TargetContainer)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	g.SourceContainer = &tlc.Container{}
	err = patchWire.ReadMessage(g.SourceContainer)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	return nil
}

func (g *Genie) ParseContents(comps chan *Composition) error {
	patchWire := g.PatchWire

	smallBlockSize := int64(pwr.BlockSize)
	bigBlockSize := g.BlockSize

	sh := &pwr.SyncHeader{}
	for fileIndex, _ := range g.SourceContainer.Files {
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
			comp := &Composition{
				FileIndex: int64(fileIndex),
			}

			for {
				rop.Reset()
				pErr := patchWire.ReadMessage(rop)
				if pErr != nil {
					return errors.Wrap(pErr, 1)
				}

				switch rop.Type {
				case pwr.SyncOp_BLOCK_RANGE:
					bo := &BlockOrigin{
						FileIndex: rop.FileIndex,
						Offset:    rop.BlockIndex * smallBlockSize,
						Size:      rop.BlockSpan * smallBlockSize,
					}

					for comp.Size+bo.Size > bigBlockSize {
						truncatedSize := bigBlockSize - comp.Size

						if truncatedSize > 0 {
							comp.Append(&BlockOrigin{
								FileIndex: rop.FileIndex,
								Offset:    bo.Offset,
								Size:      truncatedSize,
							})
							bo.Offset += truncatedSize
							bo.Size -= truncatedSize
						}

						comps <- comp
						comp = &Composition{
							FileIndex:  int64(fileIndex),
							BlockIndex: comp.BlockIndex + 1,
						}
					}

					if bo.Size > 0 {
						comp.Append(bo)
					}
				case pwr.SyncOp_DATA:
					fo := &FreshOrigin{
						Size: int64(len(rop.Data)),
					}

					for comp.Size+fo.Size > bigBlockSize {
						truncatedSize := bigBlockSize - comp.Size
						if truncatedSize > 0 {
							comp.Append(&FreshOrigin{
								Size: truncatedSize,
							})
							fo.Size -= truncatedSize
						}

						comps <- comp
						comp = &Composition{
							FileIndex:  int64(fileIndex),
							BlockIndex: comp.BlockIndex + 1,
						}
					}

					if fo.Size > 0 {
						comp.Append(fo)
					}
				case pwr.SyncOp_HEY_YOU_DID_IT:
					if comp.Size > 0 {
						comps <- comp
					}
					return nil
				}
			}
		})()
		if err != nil {
			return errors.Wrap(err, 1)
		}
	}

	return nil
}
