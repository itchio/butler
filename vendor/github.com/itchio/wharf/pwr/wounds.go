package pwr

import (
	"fmt"
	"os"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

///////////////////////////////
// Writer
///////////////////////////////

type WoundsWriter struct {
	Wounds chan *Wound
}

func (ww *WoundsWriter) Do(signature *SignatureInfo, woundsPath string) error {
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
		err := writeWound(wound)
		if err != nil {
			return err
		}
	}

	return nil
}

///////////////////////////////
// Writer
///////////////////////////////

type WoundsPrinter struct {
	Wounds chan *Wound
}

func (wp *WoundsPrinter) Do(signature *SignatureInfo, consumer *StateConsumer) error {
	if consumer == nil {
		return fmt.Errorf("Missing Consumer in WoundsPrinter")
	}

	for wound := range wp.Wounds {
		consumer.Infof(wound.PrettyString(signature.Container))
	}

	return nil
}

///////////////////////////////
// Utils
///////////////////////////////

func AggregateWounds(outWounds chan *Wound, maxSize int64) chan *Wound {
	var lastWound *Wound
	inWounds := make(chan *Wound)

	go func() {
		for wound := range inWounds {
			// try to aggregate input wounds into fewer, wider wounds
			if lastWound == nil {
				lastWound = wound
			} else {
				if lastWound.End <= wound.Start && wound.Start >= lastWound.Start {
					lastWound.End = wound.End

					if lastWound.End-lastWound.Start >= maxSize {
						outWounds <- lastWound
						lastWound = nil
					}
				} else {
					outWounds <- lastWound
					lastWound = wound
				}
			}
		}

		if lastWound != nil {
			outWounds <- lastWound
		}

		close(outWounds)
	}()

	return inWounds
}

func (w *Wound) PrettyString(container *tlc.Container) string {
	file := container.Files[w.FileIndex]
	woundSize := humanize.IBytes(uint64(w.End - w.Start))
	offset := humanize.IBytes(uint64(w.Start))
	return fmt.Sprintf("~%s wound %s into %s", woundSize, offset, file.Path)
}
