package pwr

import (
	"io"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
	"github.com/itchio/wharf/wsync"
)

// A SignatureInfo contains all the hashes for small-blocks of a given container
type SignatureInfo struct {
	Container *tlc.Container
	Hashes    []wsync.BlockHash
}

// ComputeSignature compute the signature of all blocks of all files in a given container,
// by reading them from disk, relative to `basePath`, and notifying `consumer` of its
// progress
func ComputeSignature(container *tlc.Container, pool wsync.Pool, consumer *StateConsumer) ([]wsync.BlockHash, error) {
	var signature []wsync.BlockHash

	err := ComputeSignatureToWriter(container, pool, consumer, func(bl wsync.BlockHash) error {
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
func ComputeSignatureToWriter(container *tlc.Container, pool wsync.Pool, consumer *StateConsumer, sigWriter wsync.SignatureWriter) error {
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

	var hashes []wsync.BlockHash
	hash := &BlockHash{}

	for fileIndex, f := range container.Files {
		numBlocks := ComputeNumBlocks(f.Size)
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
			blockHash := wsync.BlockHash{
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
			if (blockIndex+1)*BlockSize > f.Size {
				shortSize = int32(f.Size % BlockSize)
			}

			blockHash := wsync.BlockHash{
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
