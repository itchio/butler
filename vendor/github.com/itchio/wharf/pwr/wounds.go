package pwr

import (
	"context"
	"fmt"
	"os"

	"github.com/itchio/httpkit/progress"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
	"github.com/pkg/errors"
)

// A WoundsConsumer takes file corruption information as input,
// and does something with it: print it, write it to a file, heal
// the corrupted files.
type WoundsConsumer interface {
	// Do starts receiving wounds from the given channel, and returns
	// on error or when wound processing is done.
	Do(ctx context.Context, container *tlc.Container, wounds chan *Wound) error

	// TotalCorrupted returns the total size of corrupted data seen by this consumer.
	// If the only wounds are dir and symlink wounds, this may be 0, but HasWounds might
	// still be true
	TotalCorrupted() int64

	// HasWounds returns true if any wounds were received by this consumer
	HasWounds() bool
}

///////////////////////////////
// Writer
///////////////////////////////

// WoundsGuardian is a wounds consumer that returns an error on the first wound received.
type WoundsGuardian struct {
	totalCorrupted int64
	hasWounds      bool
}

var _ WoundsConsumer = (*WoundsGuardian)(nil)

type ErrHasWound struct {
	Wound     *Wound
	Container *tlc.Container
}

var _ error = (*ErrHasWound)(nil)

func (e *ErrHasWound) Error() string {
	return e.Wound.PrettyString(e.Container)
}

// Do returns an error on the first wound received. If no wounds are ever received,
// it returns nil (no error)
func (wg *WoundsGuardian) Do(ctx context.Context, container *tlc.Container, wounds chan *Wound) error {
	for {
		select {
		case wound := <-wounds:
			if wound == nil {
				// channel closed
				return nil
			}

			if wound.Healthy() {
				continue
			}

			wg.hasWounds = true
			wg.totalCorrupted += wound.Size()
			return &ErrHasWound{
				Wound:     wound,
				Container: container,
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// TotalCorrupted is only ever 0 or the size of the first wound, since a guardian
// doesn't keep track of any wounds beyond that
func (wg *WoundsGuardian) TotalCorrupted() int64 {
	return wg.totalCorrupted
}

// HasWounds returns true if the guardian has seen a wound
func (wg *WoundsGuardian) HasWounds() bool {
	return wg.hasWounds
}

///////////////////////////////
// Writer
///////////////////////////////

// WoundsWriter writes wounds to a .pww (wharf wounds file format) file
type WoundsWriter struct {
	WoundsPath string

	totalCorrupted int64
	hasWounds      bool
}

var _ WoundsConsumer = (*WoundsWriter)(nil)

// Do only create a file at WoundsPath when it receives the first wound.
// If no wounds are ever received, Do will effectively be a no-op.
func (ww *WoundsWriter) Do(ctx context.Context, container *tlc.Container, wounds chan *Wound) error {
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
				return errors.WithStack(err)
			}

			wc = wire.NewWriteContext(fw)
			if err != nil {
				return errors.WithStack(err)
			}

			err = wc.WriteMagic(WoundsMagic)
			if err != nil {
				return errors.WithStack(err)
			}

			err = wc.WriteMessage(&WoundsHeader{})
			if err != nil {
				return errors.WithStack(err)
			}

			err = wc.WriteMessage(container)
			if err != nil {
				return errors.WithStack(err)
			}
		}

		err := wc.WriteMessage(wound)
		if err != nil {
			return errors.WithStack(err)
		}

		return nil
	}

	for {
		select {
		case wound := <-wounds:
			if wound == nil {
				// channel's closed, let's go home!
				return nil
			}

			if wound.Healthy() {
				continue
			}

			ww.hasWounds = true
			err := writeWound(wound)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			// cancelled, let's go home too
			return nil
		}
	}
}

// TotalCorrupted returns the total size of wounds received by this wounds writer
func (ww *WoundsWriter) TotalCorrupted() int64 {
	return ww.totalCorrupted
}

// HasWounds returns true if this wounds writer has received any wounds at all
func (ww *WoundsWriter) HasWounds() bool {
	return ww.hasWounds
}

///////////////////////////////
// Writer
///////////////////////////////

// WoundsPrinter prints all received wounds as a Debug message to the given consumer.
type WoundsPrinter struct {
	Consumer *state.Consumer

	totalCorrupted int64
	hasWounds      bool
}

var _ WoundsConsumer = (*WoundsPrinter)(nil)

// Do starts printing wounds. It will return an error if a Consumer is not given
func (wp *WoundsPrinter) Do(ctx context.Context, container *tlc.Container, wounds chan *Wound) error {
	if wp.Consumer == nil {
		return fmt.Errorf("Missing Consumer in WoundsPrinter")
	}

	for {
		select {
		case wound := <-wounds:
			if wound == nil {
				// channel's closed
				return nil
			}

			if wound.Healthy() {
				continue
			}

			wp.totalCorrupted += wound.Size()
			wp.hasWounds = true
			wp.Consumer.Debugf(wound.PrettyString(container))
		case <-ctx.Done():
			return nil
		}
	}
}

// TotalCorrupted returns the total size of wounds received by this wounds printer
func (wp *WoundsPrinter) TotalCorrupted() int64 {
	return wp.totalCorrupted
}

// HasWounds returns true if this wounds printer has received any wounds at all
func (wp *WoundsPrinter) HasWounds() bool {
	return wp.hasWounds
}

///////////////////////////////
// Utils
///////////////////////////////

// AggregateWounds returns a channel that it'll receive wounds from,
// try to aggregate them into bigger wounds (for example: 250 contiguous 16KB wounds = one 4MB wound),
// and send to outWounds. It may return wounds bigger than maxSize, since it
// doesn't do any wound splitting, and it may return wounds smaller than maxSize,
// since it should relay all input wounds, no matter what size.
func AggregateWounds(outWounds chan *Wound, maxSize int64) chan *Wound {
	var lastWound *Wound
	inWounds := make(chan *Wound)

	go func() {
		for wound := range inWounds {
			if wound.Kind == WoundKind_FILE {
				// try to aggregate input file wounds into fewer, wider wounds
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
			} else {
				if lastWound != nil {
					// clear out any waiting lastWound
					outWounds <- lastWound
					lastWound = nil
				}
				outWounds <- wound
			}
		}

		if lastWound != nil {
			outWounds <- lastWound
		}

		close(outWounds)
	}()

	return inWounds
}

// PrettyString returns a human-readable English string for a given wound
func (w *Wound) PrettyString(container *tlc.Container) string {
	switch w.Kind {
	case WoundKind_DIR:
		dir := container.Dirs[w.Index]
		return fmt.Sprintf("directory wound (%s should exist)", dir.Path)
	case WoundKind_SYMLINK:
		symlink := container.Symlinks[w.Index]
		return fmt.Sprintf("symlink wound (%s should point to %s)", symlink.Path, symlink.Dest)
	case WoundKind_FILE:
		file := container.Files[w.Index]
		woundSize := progress.FormatBytes(w.End - w.Start)
		offset := progress.FormatBytes(w.Start)
		return fmt.Sprintf("~%s wound %s into %s", woundSize, offset, file.Path)
	default:
		return fmt.Sprintf("unknown wound (%d)", w.Kind)
	}
}

// Size returns the size of the wound, ie. how many bytes are corrupted
func (w *Wound) Size() int64 {
	return w.End - w.Start
}

// Healthy returns true if the wound is not a wound, but simply a progress
// indicator used when validating files. It should not count towards HasWounds()
func (w *Wound) Healthy() bool {
	return w.Kind == WoundKind_CLOSED_FILE
}
