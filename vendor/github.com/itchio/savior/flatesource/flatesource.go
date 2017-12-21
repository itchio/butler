package flatesource

import (
	"encoding/gob"
	"fmt"

	"github.com/go-errors/errors"
	"github.com/itchio/kompress/flate"
	"github.com/itchio/savior"
	"github.com/mohae/deepcopy"
)

type flateSource struct {
	// input
	source savior.Source

	// params
	threshold int64

	// internal
	sr      flate.SaverReader
	offset  int64
	counter int64

	checkpoint *FlateSourceCheckpoint
}

type FlateSourceCheckpoint struct {
	SourceCheckpoint *savior.SourceCheckpoint
	FlateCheckpoint  *flate.Checkpoint
}

var _ savior.Source = (*flateSource)(nil)

func New(source savior.Source, threshold int64) *flateSource {
	return &flateSource{
		source:    source,
		threshold: threshold,
	}
}

func (fs *flateSource) Save() (*savior.SourceCheckpoint, error) {
	if fs.checkpoint != nil {
		c := &savior.SourceCheckpoint{
			Offset: fs.offset,
			Data:   fs.checkpoint,
		}
		return c, nil
	}
	return nil, nil
}

func (fs *flateSource) Resume(checkpoint *savior.SourceCheckpoint) (int64, error) {
	fs.counter = 0
	fs.checkpoint = nil

	savior.Debugf(`flate: asked to resume`)

	if checkpoint != nil {
		if ourCheckpoint, ok := checkpoint.Data.(*FlateSourceCheckpoint); ok {
			sourceOffset, err := fs.source.Resume(ourCheckpoint.SourceCheckpoint)
			if err != nil {
				return 0, errors.Wrap(err, 0)
			}

			fc := ourCheckpoint.FlateCheckpoint
			if sourceOffset == fc.Roffset {
				fs.sr, err = fc.Resume(fs.source)
				if err != nil {
					savior.Debugf(`flatesource: could not use flate checkpoint at R=%d / W=%d`, fc.Roffset, fc.Woffset)
					// well, let's start over
					_, err = fs.source.Resume(nil)
					if err != nil {
						return 0, errors.Wrap(err, 0)
					}
				} else {
					fs.offset = fc.Woffset
					return fc.Woffset, nil
				}
			} else {
				savior.Debugf(`flatesource: expected source to resume at %d but got %d`, fc.Roffset, sourceOffset)
			}
		}
	}

	// start from beginning
	sourceOffset, err := fs.source.Resume(nil)
	if err != nil {
		return 0, errors.Wrap(err, 0)
	}

	if sourceOffset != 0 {
		msg := fmt.Sprintf("flatesource: expected source to resume at start but got %d", sourceOffset)
		return 0, errors.New(msg)
	}

	fs.sr = flate.NewSaverReader(fs.source)
	fs.offset = 0
	return 0, nil
}

func (fs *flateSource) Read(buf []byte) (int, error) {
	n, err := fs.sr.Read(buf)
	fs.offset += int64(n)
	fs.counter += int64(n)
	if fs.counter > fs.threshold {
		fs.sr.WantSave()
		fs.counter = 0
	}

	if err != nil {
		if err == flate.ReadyToSaveError {
			flateCheckpoint, saveErr := fs.sr.Save()
			if saveErr != nil {
				return n, saveErr
			}

			sourceCheckpoint, sourceErr := fs.source.Save()
			if saveErr != nil {
				return n, sourceErr
			}

			savior.Debugf("flatesource: saving, flate rOffset = %d, sourceCheckpoint.Offset = %d", flateCheckpoint.Roffset, sourceCheckpoint.Offset)

			fs.checkpoint = &FlateSourceCheckpoint{
				FlateCheckpoint:  deepcopy.Copy(flateCheckpoint).(*flate.Checkpoint),
				SourceCheckpoint: sourceCheckpoint,
			}

			savior.Debugf("flatesource: saved checkpoint at byte %d", fs.offset)
			err = nil
		}
	}

	return n, err
}

func (fs *flateSource) ReadByte() (byte, error) {
	buf := []byte{0}
	_, err := fs.Read(buf)
	return buf[0], err
}

func (fs *flateSource) Progress() float64 {
	// We can't tell how large the uncompressed stream is until we finish
	// decompressing it. The underlying's source progress is a good enough
	// approximation.
	return fs.source.Progress()
}

func init() {
	gob.Register(&FlateSourceCheckpoint{})
}
