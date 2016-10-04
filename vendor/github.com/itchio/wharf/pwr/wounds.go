package pwr

import (
	"fmt"
	"os"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

type WoundsConsumer interface {
	Do(*tlc.Container, chan *Wound) error
	TotalCorrupted() int64
}

///////////////////////////////
// Writer
///////////////////////////////

type WoundsGuardian struct {
	totalCorrupted int64
}

var _ WoundsConsumer = (*WoundsGuardian)(nil)

func (wg *WoundsGuardian) Do(container *tlc.Container, wounds chan *Wound) error {
	for wound := range wounds {
		wg.totalCorrupted += wound.Size()
		return fmt.Errorf(wound.PrettyString(container))
	}

	return nil
}

func (wg *WoundsGuardian) TotalCorrupted() int64 {
	return wg.totalCorrupted
}

///////////////////////////////
// Writer
///////////////////////////////

type WoundsWriter struct {
	WoundsPath string

	totalCorrupted int64
}

var _ WoundsConsumer = (*WoundsWriter)(nil)

func (ww *WoundsWriter) Do(container *tlc.Container, wounds chan *Wound) error {
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
		ww.totalCorrupted += wound.Size()

		if wc == nil {
			var err error
			fw, err = os.Create(ww.WoundsPath)
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

			err = wc.WriteMessage(container)
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

	for wound := range wounds {
		err := writeWound(wound)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ww *WoundsWriter) TotalCorrupted() int64 {
	return ww.totalCorrupted
}

///////////////////////////////
// Writer
///////////////////////////////

type WoundsPrinter struct {
	Consumer *StateConsumer

	totalCorrupted int64
}

var _ WoundsConsumer = (*WoundsPrinter)(nil)

func (wp *WoundsPrinter) Do(container *tlc.Container, wounds chan *Wound) error {
	if wp.Consumer == nil {
		return fmt.Errorf("Missing Consumer in WoundsPrinter")
	}

	for wound := range wounds {
		wp.totalCorrupted += wound.Size()
		wp.Consumer.Debugf(wound.PrettyString(container))
	}

	return nil
}

func (wp *WoundsPrinter) TotalCorrupted() int64 {
	return wp.totalCorrupted
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

func (w *Wound) Size() int64 {
	return w.End - w.Start
}
