package pwr

import (
	"os"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/wire"
)

type WoundsWriter struct {
	Wounds chan *Wound
}

func (ww *WoundsWriter) Go(signature *SignatureInfo, woundsPath string) error {
	var fw *os.File
	var wc *wire.WriteContext

	defer func() {
		if wc != nil {
			wc.Close()
		}

		if fw != nil {
			fw.Close()
		}
	}()

	var lastWound *Wound

	writeWound := func(wound *Wound) error {
		if wc == nil {
			var err error
			fw, err = os.Create(woundsPath)
			if err != nil {
				return errors.Wrap(err, 1)
			}

			wc = wire.NewWriteContext(fw)
			if err != nil {
				return errors.Wrap(err, 1)
			}

			err = wc.WriteMagic(WoundsMagic)
			if err != nil {
				return errors.Wrap(err, 1)
			}

			err = wc.WriteMessage(&WoundsHeader{})
			if err != nil {
				return errors.Wrap(err, 1)
			}

			err = wc.WriteMessage(signature.Container)
			if err != nil {
				return errors.Wrap(err, 1)
			}
		}

		err := wc.WriteMessage(wound)
		if err != nil {
			return errors.Wrap(err, 1)
		}

		return nil
	}

	for wound := range ww.Wounds {
		// try to aggregate input wounds into fewer, wider wounds
		if lastWound == nil {
			lastWound = wound
		} else {
			if lastWound.FileIndex == wound.FileIndex && lastWound.BlockIndex+lastWound.BlockSpan == wound.BlockIndex {
				lastWound.BlockSpan += wound.BlockSpan
			} else {
				err := writeWound(lastWound)
				if err != nil {
					return err
				}

				lastWound = wound
			}
		}
	}

	return nil
}
