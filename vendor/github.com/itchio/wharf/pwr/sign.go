package pwr

import (
	"context"
	"io"

	"github.com/itchio/savior"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/werrors"
	"github.com/itchio/wharf/wire"
	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
)

// A SignatureInfo contains all the hashes for small-blocks of a given container
type SignatureInfo struct {
	Container *tlc.Container
	Hashes    []wsync.BlockHash
}

// ComputeSignature compute the signature of all blocks of all files in a given container,
// by reading them from disk, relative to `basePath`, and notifying `consumer` of its
// progress
func ComputeSignature(ctx context.Context, container *tlc.Container, pool wsync.Pool, consumer *state.Consumer) ([]wsync.BlockHash, error) {
	var signature []wsync.BlockHash

	err := ComputeSignatureToWriter(ctx, container, pool, consumer, func(bl wsync.BlockHash) error {
		signature = append(signature, bl)
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return signature, nil
}

// ComputeSignatureToWriter is a variant of ComputeSignature that writes hashes
// to a callback
func ComputeSignatureToWriter(ctx context.Context, container *tlc.Container, pool wsync.Pool, consumer *state.Consumer, sigWriter wsync.SignatureWriter) error {
	var err error

	defer func() {
		if pErr := pool.Close(); pErr != nil && err == nil {
			err = errors.WithStack(pErr)
		}
	}()

	sctx := mksync()

	totalBytes := container.Size
	fileOffset := int64(0)

	onRead := func(count int64) {
		consumer.Progress(float64(fileOffset+count) / float64(totalBytes))
	}

	for fileIndex, f := range container.Files {
		select {
		case <-ctx.Done():
			return werrors.ErrCancelled
		default:
			// keep going!
		}

		consumer.ProgressLabel(f.Path)
		fileOffset = f.Offset

		var reader io.Reader
		reader, err = pool.GetReader(int64(fileIndex))
		if err != nil {
			return errors.WithStack(err)
		}

		cr := counter.NewReaderCallback(onRead, reader)
		err = sctx.CreateSignature(ctx, int64(fileIndex), cr, sigWriter)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// ReadSignature reads the hashes from all files of a given container, from a
// wharf signature file.
func ReadSignature(ctx context.Context, signatureReader savior.SeekSource) (*SignatureInfo, error) {
	rawSigWire := wire.NewReadContext(signatureReader)
	err := rawSigWire.ExpectMagic(SignatureMagic)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	header := &SignatureHeader{}
	err = rawSigWire.ReadMessage(header)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	sigWire, err := DecompressWire(rawSigWire, header.Compression)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	container := &tlc.Container{}
	err = sigWire.ReadMessage(container)
	if err != nil {
		if errors.Cause(err) == io.EOF {
			// ok
		} else {
			return nil, errors.WithStack(err)
		}
	}

	var hashes []wsync.BlockHash
	hash := &BlockHash{}

	for fileIndex, f := range container.Files {
		select {
		case <-ctx.Done():
			return nil, werrors.ErrCancelled
		default:
			// keep going!
		}

		numBlocks := ComputeNumBlocks(f.Size)
		if numBlocks == 0 {
			hash.Reset()
			err = sigWire.ReadMessage(hash)

			if err != nil {
				if errors.Cause(err) == io.EOF {
					break
				}
				return nil, errors.WithStack(err)
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
				if errors.Cause(err) == io.EOF {
					break
				}
				return nil, errors.WithStack(err)
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
