package pwr

import (
	"bytes"
	"fmt"
	"io"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/sync"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

// A SignatureInfo contains all the hashes for small-blocks of a given container
type SignatureInfo struct {
	Container *tlc.Container
	Hashes    []sync.BlockHash
}

// ComputeSignature compute the signature of all blocks of all files in a given container,
// by reading them from disk, relative to `basePath`, and notifying `consumer` of its
// progress
func ComputeSignature(container *tlc.Container, pool sync.Pool, consumer *StateConsumer) ([]sync.BlockHash, error) {
	var signature []sync.BlockHash

	err := ComputeSignatureToWriter(container, pool, consumer, func(bl sync.BlockHash) error {
		signature = append(signature, bl)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	return signature, nil
}

// ComputeSignatureToWriter is a variant of ComputeSignature that writes hashes
// to a callback
func ComputeSignatureToWriter(container *tlc.Container, pool sync.Pool, consumer *StateConsumer, sigWriter sync.SignatureWriter) error {
	var err error

	defer func() {
		if pErr := pool.Close(); pErr != nil && err == nil {
			err = errors.Wrap(pErr, 1)
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
			return errors.Wrap(err, 1)
		}

		cr := counter.NewReaderCallback(onRead, reader)
		err = sctx.CreateSignature(int64(fileIndex), cr, sigWriter)
		if err != nil {
			return errors.Wrap(err, 1)
		}
	}

	if err != nil {
		return errors.Wrap(err, 1)
	}
	return nil
}

// ReadSignature reads the hashes from all files of a given container, from a
// wharf signature file.
func ReadSignature(signatureReader io.Reader) (*SignatureInfo, error) {
	rawSigWire := wire.NewReadContext(signatureReader)
	err := rawSigWire.ExpectMagic(SignatureMagic)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	header := &SignatureHeader{}
	err = rawSigWire.ReadMessage(header)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	sigWire, err := DecompressWire(rawSigWire, header.Compression)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	container := &tlc.Container{}
	err = sigWire.ReadMessage(container)
	if err != nil {
		if errors.Is(err, io.EOF) {
			// ok
		} else {
			return nil, errors.Wrap(err, 1)
		}
	}

	var hashes []sync.BlockHash
	hash := &BlockHash{}

	blockSize64 := int64(BlockSize)

	for fileIndex, f := range container.Files {
		numBlocks := (f.Size + blockSize64 - 1) / blockSize64
		if numBlocks == 0 {
			hash.Reset()
			err = sigWire.ReadMessage(hash)

			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, errors.Wrap(err, 1)
			}

			// empty files have a 0-length shortblock for historical reasons.
			blockHash := sync.BlockHash{
				FileIndex:  int64(fileIndex),
				BlockIndex: 0,

				WeakHash:   hash.WeakHash,
				StrongHash: hash.StrongHash,

				ShortSize: 0,
			}
			hashes = append(hashes, blockHash)
		}

		for blockIndex := int64(0); blockIndex < numBlocks; blockIndex++ {
			hash.Reset()
			err = sigWire.ReadMessage(hash)

			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, errors.Wrap(err, 1)
			}

			// full blocks have a shortSize of 0, for more compact storage
			shortSize := int32(0)
			if (blockIndex+1)*blockSize64 > f.Size {
				shortSize = int32(f.Size % blockSize64)
			}

			blockHash := sync.BlockHash{
				FileIndex:  int64(fileIndex),
				BlockIndex: blockIndex,

				WeakHash:   hash.WeakHash,
				StrongHash: hash.StrongHash,

				ShortSize: shortSize,
			}
			hashes = append(hashes, blockHash)
		}
	}

	signature := &SignatureInfo{
		Container: container,
		Hashes:    hashes,
	}
	return signature, nil
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
