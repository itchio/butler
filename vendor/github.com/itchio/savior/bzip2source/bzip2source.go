package bzip2source

import (
	"encoding/gob"
	"fmt"

	"github.com/go-errors/errors"
	"github.com/itchio/kompress/bzip2"
	"github.com/itchio/savior"
	"github.com/mohae/deepcopy"
)

type bzip2Source struct {
	// input
	source savior.Source

	// params
	threshold int64

	// internal
	sr      bzip2.SaverReader
	offset  int64
	counter int64

	checkpoint *Bzip2SourceCheckpoint
}

type Bzip2SourceCheckpoint struct {
	Offset           int64
	SourceCheckpoint *savior.SourceCheckpoint
	Bzip2Checkpoint  *bzip2.Checkpoint
}

var _ savior.Source = (*bzip2Source)(nil)

func New(source savior.Source, threshold int64) *bzip2Source {
	return &bzip2Source{
		source:    source,
		threshold: threshold,
	}
}

func (bs *bzip2Source) Save() (*savior.SourceCheckpoint, error) {
	if bs.checkpoint != nil {
		c := &savior.SourceCheckpoint{
			Offset: bs.offset,
			Data:   bs.checkpoint,
		}
		return c, nil
	}
	return nil, nil
}

func (bs *bzip2Source) Resume(checkpoint *savior.SourceCheckpoint) (int64, error) {
	bs.counter = 0
	bs.checkpoint = nil

	if checkpoint != nil {
		if ourCheckpoint, ok := checkpoint.Data.(*Bzip2SourceCheckpoint); ok {
			sourceOffset, err := bs.source.Resume(ourCheckpoint.SourceCheckpoint)
			if err != nil {
				return 0, errors.Wrap(err, 0)
			}

			bc := ourCheckpoint.Bzip2Checkpoint
			if sourceOffset == bc.Roffset {
				bs.sr, err = bc.Resume(bs.source)
				if err != nil {
					savior.Debugf(`bzip2source: could not use bzip2 checkpoint at R=%d`, bc.Roffset)
					// well, let's start over
					_, err = bs.source.Resume(nil)
					if err != nil {
						return 0, errors.Wrap(err, 0)
					}
				} else {
					bs.offset = ourCheckpoint.Offset
					return bs.offset, nil
				}
			} else {
				savior.Debugf(`bzip2source: expected source to resume at %d but got %d`, bc.Roffset, sourceOffset)
			}
		}
	}

	// start from beginning
	sourceOffset, err := bs.source.Resume(nil)
	if err != nil {
		return 0, errors.Wrap(err, 0)
	}

	if sourceOffset != 0 {
		msg := fmt.Sprintf("bzip2source: expected source to resume at start but got %d", sourceOffset)
		return 0, errors.New(msg)
	}

	bs.sr = bzip2.NewSaverReader(bs.source)

	bs.offset = 0
	return 0, nil
}

func (bs *bzip2Source) Read(buf []byte) (int, error) {
	n, err := bs.sr.Read(buf)
	bs.offset += int64(n)
	bs.counter += int64(n)
	if bs.counter > bs.threshold {
		bs.sr.WantSave()
		bs.counter = 0
	}

	if err != nil {
		if err == bzip2.ReadyToSaveError {
			bzip2Checkpoint, saveErr := bs.sr.Save()
			if saveErr != nil {
				return n, saveErr
			}

			sourceCheckpoint, sourceErr := bs.source.Save()
			if saveErr != nil {
				return n, sourceErr
			}

			savior.Debugf("bzip2source: saving, bzip2 rOffset = %d, sourceCheckpoint.Offset = %d", bzip2Checkpoint.Roffset, sourceCheckpoint.Offset)

			bs.checkpoint = &Bzip2SourceCheckpoint{
				Offset:           bs.offset,
				Bzip2Checkpoint:  deepcopy.Copy(bzip2Checkpoint).(*bzip2.Checkpoint),
				SourceCheckpoint: sourceCheckpoint,
			}

			savior.Debugf("bzip2source: saved checkpoint at byte %d", bs.offset)
			err = nil
		}
	}

	return n, err
}

func (bs *bzip2Source) ReadByte() (byte, error) {
	buf := []byte{0}
	_, err := bs.Read(buf)
	return buf[0], err
}

func (bs *bzip2Source) Progress() float64 {
	// We can't tell how large the uncompressed stream is until we finish
	// decompressing it. The underlying's source progress is a good enough
	// approximation.
	return bs.source.Progress()
}

func init() {
	gob.Register(&Bzip2SourceCheckpoint{})
}
