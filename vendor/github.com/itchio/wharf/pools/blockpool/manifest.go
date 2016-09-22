package blockpool

import (
	"fmt"
	"io"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

// WriteManifest writes container info and block addresses in wharf's manifest format
// Does not close manifestWriter.
func WriteManifest(manifestWriter io.Writer, compression *pwr.CompressionSettings, container *tlc.Container, blockHashes *BlockHashMap) error {
	rawWire := wire.NewWriteContext(manifestWriter)
	err := rawWire.WriteMagic(pwr.ManifestMagic)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = rawWire.WriteMessage(&pwr.ManifestHeader{
		Compression: compression,
		Algorithm:   pwr.HashAlgorithm_SHAKE128_32,
	})
	if err != nil {
		return errors.Wrap(err, 1)
	}

	wire, err := pwr.CompressWire(rawWire, compression)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = wire.WriteMessage(container)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	sh := &pwr.SyncHeader{}
	mbh := &pwr.ManifestBlockHash{}

	for fileIndex, f := range container.Files {
		sh.Reset()
		sh.FileIndex = int64(fileIndex)
		err = wire.WriteMessage(sh)
		if err != nil {
			return errors.Wrap(err, 1)
		}

		numBlocks := (f.Size + BigBlockSize - 1) / BigBlockSize
		for blockIndex := int64(0); blockIndex < numBlocks; blockIndex++ {
			loc := BlockLocation{FileIndex: int64(fileIndex), BlockIndex: blockIndex}
			hash := blockHashes.Get(loc)
			if hash == nil {
				err = fmt.Errorf("missing BlockHash for block %+v", loc)
				return errors.Wrap(err, 1)
			}

			mbh.Reset()
			mbh.Hash = hash

			err = wire.WriteMessage(mbh)
			if err != nil {
				return errors.Wrap(err, 1)
			}
		}
	}

	err = wire.Close()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	return nil
}

// ReadManifest reads container info and block addresses from a wharf manifest file.
// Does not close manifestReader.
func ReadManifest(manifestReader io.Reader) (*tlc.Container, *BlockHashMap, error) {
	container := &tlc.Container{}
	blockHashes := NewBlockHashMap()

	rawWire := wire.NewReadContext(manifestReader)
	err := rawWire.ExpectMagic(pwr.ManifestMagic)
	if err != nil {
		return nil, nil, errors.Wrap(err, 1)
	}

	mh := &pwr.ManifestHeader{}
	err = rawWire.ReadMessage(mh)
	if err != nil {
		return nil, nil, errors.Wrap(err, 1)
	}

	if mh.Algorithm != pwr.HashAlgorithm_SHAKE128_32 {
		err = fmt.Errorf("Manifest has unsupported hash algorithm %d, expected %d", mh.Algorithm, pwr.HashAlgorithm_SHAKE128_32)
		return nil, nil, errors.Wrap(err, 1)
	}

	wire, err := pwr.DecompressWire(rawWire, mh.GetCompression())
	if err != nil {
		return nil, nil, errors.Wrap(err, 1)
	}

	err = wire.ReadMessage(container)
	if err != nil {
		return nil, nil, errors.Wrap(err, 1)
	}

	sh := &pwr.SyncHeader{}
	mbh := &pwr.ManifestBlockHash{}

	for fileIndex, f := range container.Files {
		sh.Reset()
		err = wire.ReadMessage(sh)
		if err != nil {
			return nil, nil, errors.Wrap(err, 1)
		}

		if int64(fileIndex) != sh.FileIndex {
			err = fmt.Errorf("manifest format error: expected file %d, got %d", fileIndex, sh.FileIndex)
			return nil, nil, errors.Wrap(err, 1)
		}

		numBlocks := (f.Size + BigBlockSize - 1) / BigBlockSize
		for blockIndex := int64(0); blockIndex < numBlocks; blockIndex++ {
			mbh.Reset()
			err = wire.ReadMessage(mbh)
			if err != nil {
				return nil, nil, errors.Wrap(err, 1)
			}

			loc := BlockLocation{FileIndex: int64(fileIndex), BlockIndex: blockIndex}
			blockHashes.Set(loc, append([]byte{}, mbh.Hash...))
		}
	}

	return container, blockHashes, nil
}
