package pwr

import (
	"bytes"
	"fmt"
	"io"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/pwr/drip"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

// A ValidatingPool will check files against their hashes, but doesn't
// check directories or symlinks
type ValidatingPool struct {
	// required

	Pool wsync.WritablePool
	// Container must match Pool - may have different file indices than Signature.Container
	Container *tlc.Container
	Signature *SignatureInfo

	Wounds chan *Wound

	// private //

	hashGroups map[int64][]wsync.BlockHash
	sctx       *wsync.Context
}

var _ wsync.WritablePool = (*ValidatingPool)(nil)

func (vp *ValidatingPool) GetReader(fileIndex int64) (io.Reader, error) {
	return vp.GetReadSeeker(fileIndex)
}

func (vp *ValidatingPool) GetReadSeeker(fileIndex int64) (io.ReadSeeker, error) {
	return vp.Pool.GetReadSeeker(fileIndex)
}

func (vp *ValidatingPool) GetWriter(fileIndex int64) (io.WriteCloser, error) {
	if vp.hashGroups == nil {
		err := vp.makeHashGroups()
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}
		vp.sctx = wsync.NewContext(int(BlockSize))
	}

	w, err := vp.Pool.GetWriter(fileIndex)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	hashGroup := vp.hashGroups[fileIndex]
	blockIndex := int64(0)
	fileSize := vp.Container.Files[fileIndex].Size

	validate := func(data []byte) error {
		bh := hashGroup[blockIndex]

		weakHash, strongHash := vp.sctx.HashBlock(data)

		if bh.WeakHash != weakHash {
			if vp.Wounds == nil {
				err := fmt.Errorf("at %d/%d, expected weak hash %x, got %x", fileIndex, blockIndex, bh.WeakHash, weakHash)
				return errors.Wrap(err, 1)
			} else {
				size := ComputeBlockSize(fileSize, blockIndex)
				start := blockIndex * BlockSize
				vp.Wounds <- &Wound{
					Kind:  WoundKind_FILE,
					Index: fileIndex,
					Start: start,
					End:   start + size,
				}
			}
		} else if !bytes.Equal(bh.StrongHash, strongHash) {
			if vp.Wounds == nil {
				err := fmt.Errorf("at %d/%d, expected strong hash %x, got %x", fileIndex, blockIndex, bh.StrongHash, strongHash)
				return errors.Wrap(err, 1)
			} else {
				size := ComputeBlockSize(fileSize, blockIndex)
				start := blockIndex * BlockSize
				vp.Wounds <- &Wound{
					Kind:  WoundKind_FILE,
					Index: fileIndex,
					Start: start,
					End:   start + size,
				}
			}
		}

		blockIndex++
		return nil
	}

	dw := &drip.Writer{
		Writer:   w,
		Buffer:   make([]byte, BlockSize),
		Validate: validate,
	}

	return dw, nil
}

func (vp *ValidatingPool) makeHashGroups() error {
	// see blockpool's validator for a slightly different take on this
	pathToFileIndex := make(map[string]int64)
	for fileIndex, f := range vp.Container.Files {
		pathToFileIndex[f.Path] = int64(fileIndex)
	}

	vp.hashGroups = make(map[int64][]wsync.BlockHash)
	hashIndex := int64(0)

	for _, f := range vp.Signature.Container.Files {
		fileIndex := pathToFileIndex[f.Path]

		if f.Size == 0 {
			// empty files have a 0-length shortblock for historical reasons.
			hashIndex++
			continue
		}

		numBlocks := ComputeNumBlocks(f.Size)
		vp.hashGroups[fileIndex] = vp.Signature.Hashes[hashIndex : hashIndex+numBlocks]
		hashIndex += numBlocks
	}

	if hashIndex != int64(len(vp.Signature.Hashes)) {
		err := fmt.Errorf("expected to have %d hashes in signature, had %d", hashIndex, len(vp.Signature.Hashes))
		return errors.Wrap(err, 1)
	}

	return nil
}

func (vp *ValidatingPool) Close() error {
	return vp.Pool.Close()
}
