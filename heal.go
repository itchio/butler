package main

import (
	"os"

	"io"

	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

func heal(dir string, wounds string, spec string) {
	must(doHeal(dir, wounds, spec))
}

func doHeal(dir string, woundsPath string, spec string) error {
	reader, err := os.Open(woundsPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	healer, err := pwr.NewHealer(spec, dir)
	if err != nil {
		return err
	}

	consumer := comm.NewStateConsumer()

	healer.SetConsumer(consumer)

	rctx := wire.NewReadContext(reader)

	err = rctx.ExpectMagic(pwr.WoundsMagic)
	if err != nil {
		return err
	}

	wh := &pwr.WoundsHeader{}
	err = rctx.ReadMessage(wh)
	if err != nil {
		return err
	}

	container := &tlc.Container{}
	err = rctx.ReadMessage(container)
	if err != nil {
		return err
	}

	wounds := make(chan *pwr.Wound)
	errs := make(chan error)

	comm.StartProgress()

	go func() {
		errs <- healer.Do(container, wounds)
	}()

	wound := &pwr.Wound{}
	for {
		wound.Reset()
		err = rctx.ReadMessage(wound)
		if err != nil {
			if err == io.EOF {
				// all good
				break
			}
		}

		select {
		case wounds <- wound:
			// all good
		case healErr := <-errs:
			return healErr
		}
	}

	close(wounds)
	healErr := <-errs

	comm.EndProgress()

	if healErr != nil {
		return healErr
	}

	comm.Opf("All healed!")
	return nil
}
