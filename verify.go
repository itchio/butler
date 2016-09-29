package main

import (
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/pwr"
)

func verify(signature string, output string) {
	must(doVerify(signature, output))
}

func doVerify(signature string, output string) error {
	comm.Opf("Verifying %s", output)
	startTime := time.Now()

	signatureReader, err := os.Open(signature)
	if err != nil {
		return errors.Wrap(err, 1)
	}
	defer signatureReader.Close()

	refSignature, err := pwr.ReadSignature(signatureReader)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	refContainer := refSignature.Container
	refHashes := refSignature.Hashes

	comm.StartProgress()
	hashes, err := pwr.ComputeSignature(refContainer, fspool.New(refContainer, output), comm.NewStateConsumer())
	comm.EndProgress()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = pwr.CompareHashes(refHashes, hashes, refContainer)
	if err != nil {
		comm.Logf(err.Error())
		comm.Dief("Some checks failed after checking %d blocks.", len(refHashes))
	}

	prettySize := humanize.IBytes(uint64(refContainer.Size))
	perSecond := humanize.IBytes(uint64(float64(refContainer.Size) / time.Since(startTime).Seconds()))
	comm.Statf("%s (%s) @ %s/s\n", prettySize, refContainer.Stats(), perSecond)

	return nil
}
