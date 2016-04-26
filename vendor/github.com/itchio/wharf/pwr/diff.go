package pwr

import (
	"fmt"
	"io"

	"github.com/dustin/go-humanize"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/sync"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

type DataLookupFunction func([]byte) (string, error)

// DiffContext holds the state during a diff operation
type DiffContext struct {
	Compression *CompressionSettings
	Consumer    *StateConsumer

	SourceContainer *tlc.Container
	FilePool        sync.FilePool

	TargetContainer *tlc.Container
	TargetSignature []sync.BlockHash

	ReusedBytes int64
	FreshBytes  int64

	AddedBytes int64
	SavedBytes int64

	DataLookup DataLookupFunction
}

// WritePatch outputs a pwr patch to patchWriter
func (dctx *DiffContext) WritePatch(patchWriter io.Writer, signatureWriter io.Writer) error {
	if dctx.Compression == nil {
		return fmt.Errorf("No compression settings specified, bailing out")
	}

	// signature header
	rawSigWire := wire.NewWriteContext(signatureWriter)
	err := rawSigWire.WriteMagic(SignatureMagic)
	if err != nil {
		return err
	}

	err = rawSigWire.WriteMessage(&SignatureHeader{
		Compression: dctx.Compression,
	})
	if err != nil {
		return err
	}

	sigWire, err := CompressWire(rawSigWire, dctx.Compression)
	if err != nil {
		return err
	}

	err = sigWire.WriteMessage(dctx.SourceContainer)
	if err != nil {
		return err
	}

	// patch header
	rawPatchWire := wire.NewWriteContext(patchWriter)
	err = rawPatchWire.WriteMagic(PatchMagic)
	if err != nil {
		return err
	}

	header := &PatchHeader{
		Compression: dctx.Compression,
	}

	err = rawPatchWire.WriteMessage(header)
	if err != nil {
		return err
	}

	patchWire, err := CompressWire(rawPatchWire, dctx.Compression)
	if err != nil {
		return err
	}

	err = patchWire.WriteMessage(dctx.TargetContainer)
	if err != nil {
		return err
	}

	err = patchWire.WriteMessage(dctx.SourceContainer)
	if err != nil {
		return err
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
	blockLibrary := sync.NewBlockLibrary(dctx.TargetSignature)

	targetContainerPathToIndex := make(map[string]int64)
	for index, f := range dctx.TargetContainer.Files {
		targetContainerPathToIndex[f.Path] = int64(index)
	}

	// re-used messages
	syncHeader := &SyncHeader{}
	syncDelimiter := &SyncOp{
		Type: SyncOp_HEY_YOU_DID_IT,
	}

	filePool := dctx.FilePool
	defer func() {
		if fErr := filePool.Close(); fErr != nil && err == nil {
			err = fErr
		}
	}()

	for fileIndex, f := range dctx.SourceContainer.Files {
		dctx.Consumer.ProgressLabel(f.Path)
		dctx.Consumer.Debug(fmt.Sprintf("%s (%s)", f.Path, humanize.Bytes(uint64(f.Size))))
		fileOffset = f.Offset

		syncHeader.Reset()
		syncHeader.FileIndex = int64(fileIndex)
		err = patchWire.WriteMessage(syncHeader)
		if err != nil {
			return err
		}

		var sourceReader io.Reader
		sourceReader, err = filePool.GetReader(int64(fileIndex))
		if err != nil {
			return err
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
					errs <- dErr
				}
			}()
			defer func() {
				if sErr := signWriter.Close(); sErr != nil {
					errs <- sErr
				}
			}()

			mw := io.MultiWriter(diffWriter, signWriter)

			sourceReadCounter := counter.NewReaderCallback(onSourceRead, sourceReader)
			_, cErr := io.Copy(mw, sourceReadCounter)
			if cErr != nil {
				errs <- cErr
			}
		}()

		// wait until all are done
		// or an error occurs
		for c := 0; c < 2; c++ {
			select {
			case err := <-errs:
				return err
			case <-done:
			}
		}

		err = patchWire.WriteMessage(syncDelimiter)
		if err != nil {
			return err
		}
	}

	err = patchWire.Close()
	if err != nil {
		return err
	}
	err = sigWire.Close()

	return err
}

func diffFile(sctx *sync.Context, dctx *DiffContext, blockLibrary *sync.BlockLibrary, reader io.Reader, opsWriter sync.OperationWriter, preferredFileIndex int64, errs chan error, done chan bool) {
	writeOp := func(op sync.Operation) error {
		if op.Type == sync.OpData {
			if dctx.DataLookup != nil {
				key, err := dctx.DataLookup(op.Data)
				if err != nil {
					return err
				}

				if key == "" {
					dctx.AddedBytes += int64(len(op.Data))
				} else {
					// TODO: new op type
					dctx.SavedBytes += int64(len(op.Data))
					return nil
				}
			}
		}
		return opsWriter(op)
	}

	err := sctx.ComputeDiff(reader, blockLibrary, writeOp, preferredFileIndex)
	if err != nil {
		errs <- err
	}

	done <- true
}

func signFile(sctx *sync.Context, fileIndex int, reader io.Reader, writeHash sync.SignatureWriter, errs chan error, done chan bool) {
	err := sctx.CreateSignature(int64(fileIndex), reader, writeHash)
	if err != nil {
		errs <- err
	}

	done <- true
}

func makeSigWriter(wc *wire.WriteContext) sync.SignatureWriter {
	return func(bl sync.BlockHash) error {
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

func makeOpsWriter(wc *wire.WriteContext, dctx *DiffContext) sync.OperationWriter {
	numOps := 0
	wop := &SyncOp{}

	blockSize64 := int64(BlockSize)
	files := dctx.TargetContainer.Files

	return func(op sync.Operation) error {
		numOps++
		wop.Reset()

		switch op.Type {
		case sync.OpBlockRange:
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

		case sync.OpData:
			wop.Type = SyncOp_DATA
			wop.Data = op.Data

			dctx.FreshBytes += int64(len(op.Data))

		case sync.OpRemoteData:
			wop.Type = SyncOp_REMOTE_DATA
			wop.RemoteSize = op.RemoteSize
			wop.RemotePath = op.RemotePath

			dctx.FreshBytes += int64(op.RemoteSize)

		default:
			return fmt.Errorf("unknown rsync op type: %d", op.Type)
		}

		err := wc.WriteMessage(wop)
		if err != nil {
			return err
		}

		return nil
	}
}
