package main

import (
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/sync"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

func sign(output string, signature string, compression pwr.CompressionSettings, fixPerms bool) {
	must(doSign(output, signature, compression, fixPerms))
}

func doSign(output string, signature string, compression pwr.CompressionSettings, fixPerms bool) error {
	comm.Opf("Creating signature for %s", output)
	startTime := time.Now()

	container, err := tlc.Walk(output, filterPaths)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if fixPerms {
		container.FixPermissions(fspool.New(container, output))
	}

	signatureWriter, err := os.Create(signature)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	rawSigWire := wire.NewWriteContext(signatureWriter)
	rawSigWire.WriteMagic(pwr.SignatureMagic)

	rawSigWire.WriteMessage(&pwr.SignatureHeader{
		Compression: &compression,
	})

	sigWire, err := pwr.CompressWire(rawSigWire, &compression)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	sigWire.WriteMessage(container)

	comm.StartProgress()
	err = pwr.ComputeSignatureToWriter(container, fspool.New(container, output), comm.NewStateConsumer(), func(hash sync.BlockHash) error {
		return sigWire.WriteMessage(&pwr.BlockHash{
			WeakHash:   hash.WeakHash,
			StrongHash: hash.StrongHash,
		})
	})
	comm.EndProgress()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = sigWire.Close()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	prettySize := humanize.IBytes(uint64(container.Size))
	perSecond := humanize.IBytes(uint64(float64(container.Size) / time.Since(startTime).Seconds()))
	comm.Statf("%s (%s) @ %s/s\n", prettySize, container.Stats(), perSecond)

	return nil
}
