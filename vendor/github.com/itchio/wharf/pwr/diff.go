package pwr

import (
	"context"
	"fmt"
	"io"

	"github.com/itchio/wharf/counter"

	"github.com/itchio/wharf/multiread"
	"github.com/itchio/wharf/taskgroup"

	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
)

// DiffContext holds the state during a diff operation
type DiffContext struct {
	Compression *CompressionSettings
	Consumer    *state.Consumer

	SourceContainer *tlc.Container
	Pool            wsync.Pool

	TargetContainer *tlc.Container
	TargetSignature []wsync.BlockHash

	ReusedBytes int64
	FreshBytes  int64

	AddedBytes int64
	SavedBytes int64
}

// WritePatch outputs a pwr patch to patchWriter
func (dctx *DiffContext) WritePatch(ctx context.Context, patchWriter io.Writer, signatureWriter io.Writer) error {
	if dctx.Compression == nil {
		return errors.WithStack(fmt.Errorf("No compression settings specified, bailing out"))
	}

	// signature header
	rawSigWire := wire.NewWriteContext(signatureWriter)
	err := rawSigWire.WriteMagic(SignatureMagic)
	if err != nil {
		return errors.WithStack(err)
	}

	err = rawSigWire.WriteMessage(&SignatureHeader{
		Compression: dctx.Compression,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	sigWire, err := CompressWire(rawSigWire, dctx.Compression)
	if err != nil {
		return errors.WithStack(err)
	}

	err = sigWire.WriteMessage(dctx.SourceContainer)
	if err != nil {
		return errors.WithStack(err)
	}

	// patch header
	rawPatchWire := wire.NewWriteContext(patchWriter)
	err = rawPatchWire.WriteMagic(PatchMagic)
	if err != nil {
		return errors.WithStack(err)
	}

	header := &PatchHeader{
		Compression: dctx.Compression,
	}

	err = rawPatchWire.WriteMessage(header)
	if err != nil {
		return errors.WithStack(err)
	}

	patchWire, err := CompressWire(rawPatchWire, dctx.Compression)
	if err != nil {
		return errors.WithStack(err)
	}

	err = patchWire.WriteMessage(dctx.TargetContainer)
	if err != nil {
		return errors.WithStack(err)
	}

	err = patchWire.WriteMessage(dctx.SourceContainer)
	if err != nil {
		return errors.WithStack(err)
	}

	sourceBytes := dctx.SourceContainer.Size
	fileOffset := int64(0)

	onSourceRead := func(count int64) {
		dctx.Consumer.Progress(float64(fileOffset+count) / float64(sourceBytes))
	}

	sigWriter := makeSigWriter(sigWire)
	opsWriter := makeOpsWriter(patchWire, dctx)

	diffContext := mksync()
	signContext := mksync()
	blockLibrary := wsync.NewBlockLibrary(dctx.TargetSignature)

	targetContainerPathToIndex := make(map[string]int64)
	for index, f := range dctx.TargetContainer.Files {
		targetContainerPathToIndex[f.Path] = int64(index)
	}

	// re-used messages
	syncHeader := &SyncHeader{}
	syncDelimiter := &SyncOp{
		Type: SyncOp_HEY_YOU_DID_IT,
	}

	pool := dctx.Pool
	defer func() {
		if fErr := pool.Close(); fErr != nil && err == nil {
			err = errors.WithStack(fErr)
		}
	}()

	for fileIndex, f := range dctx.SourceContainer.Files {
		dctx.Consumer.ProgressLabel(f.Path)
		fileOffset = f.Offset

		syncHeader.Reset()
		syncHeader.FileIndex = int64(fileIndex)
		err = patchWire.WriteMessage(syncHeader)
		if err != nil {
			return errors.WithStack(err)
		}

		var sourceReader io.Reader
		sourceReader, err = pool.GetReader(int64(fileIndex))
		if err != nil {
			return errors.WithStack(err)
		}

		var preferredFileIndex int64 = -1
		if oldIndex, ok := targetContainerPathToIndex[f.Path]; ok {
			preferredFileIndex = oldIndex
		}

		mr := multiread.New(counter.NewReaderCallback(onSourceRead, sourceReader))
		diffReader := mr.Reader()
		signReader := mr.Reader()

		err := taskgroup.Do(
			ctx,
			func() error {
				return diffContext.ComputeDiff(diffReader, blockLibrary, opsWriter, preferredFileIndex)
			},
			func() error {
				return signContext.CreateSignature(ctx, int64(fileIndex), signReader, sigWriter)
			},
			func() error {
				return mr.Do(ctx)
			},
		)
		if err != nil {
			return errors.WithStack(err)
		}

		err = patchWire.WriteMessage(syncDelimiter)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	err = patchWire.Close()
	if err != nil {
		return errors.WithStack(err)
	}
	err = sigWire.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func makeSigWriter(wc *wire.WriteContext) wsync.SignatureWriter {
	return func(bl wsync.BlockHash) error {
		return wc.WriteMessage(&BlockHash{
			WeakHash:   bl.WeakHash,
			StrongHash: bl.StrongHash,
		})
	}
}

// ComputeNumBlocks returns the number of small blocks a file is made up of.
// It returns a correct result even when the file's size is not a multiple of BlockSize
func ComputeNumBlocks(fileSize int64) int64 {
	return (fileSize + BlockSize - 1) / BlockSize
}

// ComputeBlockSize returns the size of one of the file's blocks, given the size of the file
// and the position of the block in the file. It'll return BlockSize for all blocks except
// the last one, if the file size is not a multiple of BlockSize
func ComputeBlockSize(fileSize int64, blockIndex int64) int64 {
	if BlockSize*(blockIndex+1) > fileSize {
		return fileSize % BlockSize
	}
	return BlockSize
}

func makeOpsWriter(wc *wire.WriteContext, dctx *DiffContext) wsync.OperationWriter {
	numOps := 0
	wop := &SyncOp{}

	files := dctx.TargetContainer.Files

	return func(op wsync.Operation) error {
		numOps++
		wop.Reset()

		switch op.Type {
		case wsync.OpBlockRange:
			wop.Type = SyncOp_BLOCK_RANGE
			wop.FileIndex = op.FileIndex
			wop.BlockIndex = op.BlockIndex
			wop.BlockSpan = op.BlockSpan

			fileSize := files[op.FileIndex].Size
			lastBlockIndex := op.BlockIndex + op.BlockSpan - 1
			tailSize := ComputeBlockSize(fileSize, lastBlockIndex)
			dctx.ReusedBytes += BlockSize*(op.BlockSpan-1) + tailSize

		case wsync.OpData:
			wop.Type = SyncOp_DATA
			wop.Data = op.Data

			dctx.FreshBytes += int64(len(op.Data))

		default:
			return errors.WithStack(fmt.Errorf("unknown rsync op type: %d", op.Type))
		}

		err := wc.WriteMessage(wop)
		if err != nil {
			return errors.WithStack(err)
		}

		return nil
	}
}
