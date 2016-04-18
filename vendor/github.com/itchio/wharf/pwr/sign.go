package pwr

import (
	"bytes"
	"fmt"
	"io"

	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/sync"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

// ComputeSignature compute the signature of all blocks of all files in a given container,
// by reading them from disk, relative to `basePath`, and notifying `consumer` of its
// progress
func ComputeSignature(container *tlc.Container, pool sync.FilePool, consumer *StateConsumer) ([]sync.BlockHash, error) {
	var signature []sync.BlockHash

	err := ComputeSignatureToWriter(container, pool, consumer, func(bl sync.BlockHash) error {
		signature = append(signature, bl)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return signature, nil
}

// ComputeSignatureToWriter is a variant of ComputeSignature that writes hashes
// to a callback
func ComputeSignatureToWriter(container *tlc.Container, pool sync.FilePool, consumer *StateConsumer, sigWriter sync.SignatureWriter) error {
	var err error

	defer func() {
		if pErr := pool.Close(); pErr != nil && err == nil {
			err = pErr
		}
	}()

	sctx := mksync()

	totalBytes := container.Size
	fileOffset := int64(0)

	onRead := func(count int64) {
		consumer.Progress(float64(fileOffset+count) / float64(totalBytes))
	}

	for fileIndex, f := range container.Files {
		consumer.ProgressLabel(f.Path)
		fileOffset = f.Offset

		var reader io.Reader
		reader, err = pool.GetReader(int64(fileIndex))
		if err != nil {
			return err
		}

		cr := counter.NewReaderCallback(onRead, reader)
		err = sctx.CreateSignature(int64(fileIndex), cr, sigWriter)
		if err != nil {
			return err
		}
	}

	return err
}

// ReadSignature reads the hashes from all files of a given container, from a
// wharf signature file.
func ReadSignature(signatureReader io.Reader) (*tlc.Container, []sync.BlockHash, error) {
	rawSigWire := wire.NewReadContext(signatureReader)
	err := rawSigWire.ExpectMagic(SignatureMagic)
	if err != nil {
		return nil, nil, err
	}

	header := &SignatureHeader{}
	err = rawSigWire.ReadMessage(header)
	if err != nil {
		return nil, nil, fmt.Errorf("While reading signature header: %s", err.Error())
	}

	sigWire, err := DecompressWire(rawSigWire, header.Compression)
	if err != nil {
		return nil, nil, fmt.Errorf("While decompressing signature wire: %s", err.Error())
	}

	container := &tlc.Container{}
	err = sigWire.ReadMessage(container)
	if err != nil {
		if err == io.EOF {
			// ok
		} else {
			return nil, nil, fmt.Errorf("While reading signature container: %s", err.Error())
		}
	}

	var signature []sync.BlockHash
	hash := &BlockHash{}

	blockIndex := int64(0)
	fileIndex := int64(0)
	byteOffset := int64(0)
	blockSize64 := int64(BlockSize)

	for {
		hash.Reset()
		err = sigWire.ReadMessage(hash)

		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, err
		}

		sizeDiff := container.Files[fileIndex].Size - byteOffset
		shortSize := int32(0)

		if sizeDiff < 0 {
			byteOffset = 0
			blockIndex = 0
			fileIndex++
			sizeDiff = container.Files[fileIndex].Size
		}

		// last block
		if sizeDiff < blockSize64 {
			shortSize = int32(sizeDiff)
		} else {
			shortSize = 0
		}

		blockHash := sync.BlockHash{
			FileIndex:  fileIndex,
			BlockIndex: blockIndex,

			WeakHash:   hash.WeakHash,
			StrongHash: hash.StrongHash,

			ShortSize: shortSize,
		}
		signature = append(signature, blockHash)

		// still in same file
		byteOffset += blockSize64
		blockIndex++
	}

	return container, signature, nil
}

// CompareHashes returns an error if the signatures aren't exactly the same
func CompareHashes(refHashes []sync.BlockHash, actualHashes []sync.BlockHash, refContainer *tlc.Container) error {
	if len(actualHashes) != len(refHashes) {
		return fmt.Errorf("Expected %d blocks, got %d.", len(refHashes), len(actualHashes))
	}

	location := func(i int) string {
		var hash = refHashes[i]
		var fileIndex = hash.FileIndex
		var file = refContainer.Files[fileIndex]
		byteOffset := (hash.BlockIndex * int64(BlockSize))
		return fmt.Sprintf("At block %d / %d (%d/%d bytes into %s)", i, len(refHashes), byteOffset, file.Size, file.Path)
	}

	for i, refHash := range refHashes {
		hash := actualHashes[i]

		if refHash.WeakHash != hash.WeakHash {
			return fmt.Errorf("%s, expected weak hash %x, got %x", location(i), refHash.WeakHash, hash.WeakHash)
		}

		if !bytes.Equal(refHash.StrongHash, hash.StrongHash) {
			return fmt.Errorf("%s, expected strong hash %v, got %v", location(i), refHash.StrongHash, hash.StrongHash)
		}

		if refHash.ShortSize != hash.ShortSize {
			return fmt.Errorf("%s, expected short size %d, got %d", location(i), refHash.ShortSize, hash.ShortSize)
		}

		if refHash.BlockIndex != hash.BlockIndex {
			return fmt.Errorf("%s, expected block index %d, got %d", location(i), refHash.BlockIndex, hash.BlockIndex)
		}

		if refHash.FileIndex != hash.FileIndex {
			return fmt.Errorf("%s, expected file index %d, got %d", location(i), refHash.FileIndex, hash.FileIndex)
		}
	}

	return nil
}
