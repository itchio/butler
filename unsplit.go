package main

import (
	"os"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/pools/blockpool"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/pwr"
)

func unsplit(source string, manifest string) {
	must(doUnsplit(source, manifest))
}

func doUnsplit(source string, manifest string) error {
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

	ds := &blockpool.DiskSource{
		BasePath:       "blocks",
		BlockAddresses: blockAddresses,
		Container:      container,
	}

	inPool := &blockpool.BlockPool{
		Container: container,
		Upstream:  ds,
	}

	outPool := fspool.New(container, source)

	err = container.Prepare(source)
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
