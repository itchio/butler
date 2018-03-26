package blockpool

import (
	"fmt"
	"io"

	"github.com/itchio/savior"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
	"github.com/pkg/errors"
)

// WriteManifest writes container info and block addresses in wharf's manifest format
// Does not close manifestWriter.
func WriteManifest(manifestWriter io.Writer, compression *pwr.CompressionSettings, container *tlc.Container, blockHashes *BlockHashMap) error {
	rawWire := wire.NewWriteContext(manifestWriter)
	err := rawWire.WriteMagic(pwr.ManifestMagic)
	if err != nil {
		return errors.WithStack(err)
	}

	err = rawWire.WriteMessage(&pwr.ManifestHeader{
		Compression: compression,
		Algorithm:   pwr.HashAlgorithm_SHAKE128_32,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	wire, err := pwr.CompressWire(rawWire, compression)
	if err != nil {
		return errors.WithStack(err)
	}

	err = wire.WriteMessage(container)
	if err != nil {
		return errors.WithStack(err)
	}

	sh := &pwr.SyncHeader{}
	mbh := &pwr.ManifestBlockHash{}

	for fileIndex, f := range container.Files {
		sh.Reset()
		sh.FileIndex = int64(fileIndex)
		err = wire.WriteMessage(sh)
		if err != nil {
			return errors.WithStack(err)
		}

		numBlocks := ComputeNumBlocks(f.Size)

		for blockIndex := int64(0); blockIndex < numBlocks; blockIndex++ {
			loc := BlockLocation{FileIndex: int64(fileIndex), BlockIndex: blockIndex}
			hash := blockHashes.Get(loc)
			if hash == nil {
				err = fmt.Errorf("missing BlockHash for block %+v", loc)
				return errors.WithStack(err)
			}

			mbh.Reset()
			mbh.Hash = hash

			err = wire.WriteMessage(mbh)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	err = wire.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// ReadManifest reads container info and block addresses from a wharf manifest file.
func ReadManifest(manifestReader savior.SeekSource) (*tlc.Container, *BlockHashMap, error) {
	container := &tlc.Container{}
	blockHashes := NewBlockHashMap()

	rawWire := wire.NewReadContext(manifestReader)
	err := rawWire.ExpectMagic(pwr.ManifestMagic)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	mh := &pwr.ManifestHeader{}
	err = rawWire.ReadMessage(mh)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	if mh.Algorithm != pwr.HashAlgorithm_SHAKE128_32 {
		err = fmt.Errorf("Manifest has unsupported hash algorithm %d, expected %d", mh.Algorithm, pwr.HashAlgorithm_SHAKE128_32)
		return nil, nil, errors.WithStack(err)
	}

	wire, err := pwr.DecompressWire(rawWire, mh.GetCompression())
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	err = wire.ReadMessage(container)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	sh := &pwr.SyncHeader{}
	mbh := &pwr.ManifestBlockHash{}

	for fileIndex, f := range container.Files {
		sh.Reset()
		err = wire.ReadMessage(sh)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}

		if int64(fileIndex) != sh.FileIndex {
			err = fmt.Errorf("manifest format error: expected file %d, got %d", fileIndex, sh.FileIndex)
			return nil, nil, errors.WithStack(err)
		}

		numBlocks := ComputeNumBlocks(f.Size)
		for blockIndex := int64(0); blockIndex < numBlocks; blockIndex++ {
			mbh.Reset()
			err = wire.ReadMessage(mbh)
			if err != nil {
				return nil, nil, errors.WithStack(err)
			}

			loc := BlockLocation{FileIndex: int64(fileIndex), BlockIndex: blockIndex}
			blockHashes.Set(loc, append([]byte{}, mbh.Hash...))
		}
	}

	return container, blockHashes, nil
}
