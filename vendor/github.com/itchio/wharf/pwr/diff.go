package pwr

import (
	"fmt"
	"io"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
	"github.com/itchio/wharf/wsync"
)

// DiffContext holds the state during a diff operation
type DiffContext struct {
	Compression *CompressionSettings
	Consumer    *StateConsumer

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
func (dctx *DiffContext) WritePatch(patchWriter io.Writer, signatureWriter io.Writer) error {
	if dctx.Compression == nil {
		return errors.Wrap(fmt.Errorf("No compression settings specified, bailing out"), 1)
	}

	// signature header
	rawSigWire := wire.NewWriteContext(signatureWriter)
	err := rawSigWire.WriteMagic(SignatureMagic)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = rawSigWire.WriteMessage(&SignatureHeader{
		Compression: dctx.Compression,
	})
	if err != nil {
		return errors.Wrap(err, 1)
	}

	sigWire, err := CompressWire(rawSigWire, dctx.Compression)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = sigWire.WriteMessage(dctx.SourceContainer)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	// patch header
	rawPatchWire := wire.NewWriteContext(patchWriter)
	err = rawPatchWire.WriteMagic(PatchMagic)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	header := &PatchHeader{
		Compression: dctx.Compression,
	}

	err = rawPatchWire.WriteMessage(header)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	patchWire, err := CompressWire(rawPatchWire, dctx.Compression)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = patchWire.WriteMessage(dctx.TargetContainer)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = patchWire.WriteMessage(dctx.SourceContainer)
	if err != nil {
		return errors.Wrap(err, 1)
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
			err = errors.Wrap(fErr, 1)
		}
	}()

	for fileIndex, f := range dctx.SourceContainer.Files {
		dctx.Consumer.ProgressLabel(f.Path)
		dctx.Consumer.Debug(fmt.Sprintf("%s (%s)", f.Path, humanize.IBytes(uint64(f.Size))))
		fileOffset = f.Offset

		syncHeader.Reset()
		syncHeader.FileIndex = int64(fileIndex)
		err = patchWire.WriteMessage(syncHeader)
		if err != nil {
			return errors.Wrap(err, 1)
		}

		var sourceReader io.Reader
		sourceReader, err = pool.GetReader(int64(fileIndex))
		if err != nil {
			return errors.Wrap(err, 1)
		}

		//             / differ
		// source file +
		//             \ signer
		diffReader, diffWriter := io.Pipe()
		signReader, signWriter := io.Pipe()

		done := make(chan bool)
		errs := make(chan error)

		var preferredFileIndex int64 = -1
		if oldIndex, ok := targetContainerPathToIndex[f.Path]; ok {
			preferredFileIndex = oldIndex
		}

		go diffFile(diffContext, dctx, blockLibrary, diffReader, opsWriter, preferredFileIndex, errs, done)
		go signFile(signContext, fileIndex, signReader, sigWriter, errs, done)

		go func() {
			defer func() {
				if dErr := diffWriter.Close(); dErr != nil {
					errs <- errors.Wrap(dErr, 1)
				}
			}()
			defer func() {
				if sErr := signWriter.Close(); sErr != nil {
					errs <- errors.Wrap(sErr, 1)
				}
			}()

			mw := io.MultiWriter(diffWriter, signWriter)

			sourceReadCounter := counter.NewReaderCallback(onSourceRead, sourceReader)
			_, cErr := io.Copy(mw, sourceReadCounter)
			if cErr != nil {
				errs <- errors.Wrap(cErr, 1)
			}
		}()

		// wait until all are done
		// or an error occurs
		for c := 0; c < 2; c++ {
			select {
			case wErr := <-errs:
				return errors.Wrap(wErr, 1)
			case <-done:
			}
		}

		err = patchWire.WriteMessage(syncDelimiter)
		if err != nil {
			return errors.Wrap(err, 1)
		}
	}

	err = patchWire.Close()
	if err != nil {
		return errors.Wrap(err, 1)
	}
	err = sigWire.Close()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	return nil
}

func diffFile(sctx *wsync.Context, dctx *DiffContext, blockLibrary *wsync.BlockLibrary, reader io.Reader, opsWriter wsync.OperationWriter, preferredFileIndex int64, errs chan error, done chan bool) {
	err := sctx.ComputeDiff(reader, blockLibrary, opsWriter, preferredFileIndex)
	if err != nil {
		errs <- errors.Wrap(err, 1)
	}

	done <- true
}

func signFile(sctx *wsync.Context, fileIndex int, reader io.Reader, writeHash wsync.SignatureWriter, errs chan error, done chan bool) {
	err := sctx.CreateSignature(int64(fileIndex), reader, writeHash)
	if err != nil {
		errs <- errors.Wrap(err, 1)
	}

	done <- true
}

func makeSigWriter(wc *wire.WriteContext) wsync.SignatureWriter {
	return func(bl wsync.BlockHash) error {
		return wc.WriteMessage(&BlockHash{
			WeakHash:   bl.WeakHash,
			StrongHash: bl.StrongHash,
		})
	}
}

func numBlocks(fileSize int64) int64 {
	return 1 + (fileSize-1)/int64(BlockSize)
}

func lastBlockSize(fileSize int64) int64 {
	return 1 + (fileSize-1)%int64(BlockSize)
}

func makeOpsWriter(wc *wire.WriteContext, dctx *DiffContext) wsync.OperationWriter {
	numOps := 0
	wop := &SyncOp{}

	blockSize64 := int64(BlockSize)
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

			tailSize := blockSize64
			fileSize := files[op.FileIndex].Size

			if (op.BlockIndex + op.BlockSpan) >= numBlocks(fileSize) {
				tailSize = lastBlockSize(fileSize)
			}
			dctx.ReusedBytes += blockSize64*(op.BlockSpan-1) + tailSize

		case wsync.OpData:
			wop.Type = SyncOp_DATA
			wop.Data = op.Data

			dctx.FreshBytes += int64(len(op.Data))

		default:
			return errors.Wrap(fmt.Errorf("unknown rsync op type: %d", op.Type), 1)
		}

		err := wc.WriteMessage(wop)
		if err != nil {
			return errors.Wrap(err, 1)
		}

		return nil
	}
}
