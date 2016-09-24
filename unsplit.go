package main

import (
	"os"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/pools/blockpool"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/pwr"
)

func unsplit(sourcePath string, manifest string) {
	must(doUnsplit(sourcePath, manifest))
}

func doUnsplit(sourcePath string, manifest string) error {
	manifestReader, err := os.Open(manifest)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	container, blockHashes, err := blockpool.ReadManifest(manifestReader)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	blockAddresses, err := blockHashes.ToAddressMap(container, pwr.HashAlgorithm_SHAKE128_32)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	var source blockpool.Source
	source = &blockpool.DiskSource{
		BasePath:       "blocks",
		BlockAddresses: blockAddresses,
		Container:      container,
	}

	source = &blockpool.DecompressingSource{
		Source: source,
	}

	inPool := &blockpool.BlockPool{
		Container: container,
		Upstream:  source,
	}

	outPool := fspool.New(container, sourcePath)

	err = container.Prepare(sourcePath)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.StartProgress()

	err = pwr.CopyContainer(container, outPool, inPool, comm.NewStateConsumer())
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.EndProgress()

	return nil
}
