package blockpool

import (
	"fmt"
	"io"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

func ReadManifest(manifestReader io.Reader) (*tlc.Container, BlockAddressMap, error) {
	container := &tlc.Container{}
	blockAddresses := make(BlockAddressMap)

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

			size := BigBlockSize
			if (blockIndex+1)*BigBlockSize > f.Size {
				size = f.Size % BigBlockSize
			}

			address := fmt.Sprintf("shake128-32/%x/%d", mbh.Hash, size)
			blockAddresses.Set(BlockLocation{FileIndex: int64(fileIndex), BlockIndex: blockIndex}, address)
		}
	}

	return container, blockAddresses, nil
}
