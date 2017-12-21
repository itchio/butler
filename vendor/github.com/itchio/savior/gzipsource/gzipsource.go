package gzipsource

import (
	"encoding/gob"
	"fmt"

	"github.com/go-errors/errors"
	"github.com/itchio/kompress/flate"
	"github.com/itchio/kompress/gzip"
	"github.com/itchio/savior"
	"github.com/mohae/deepcopy"
)

type gzipSource struct {
	// input
	source savior.Source

	// params
	threshold int64

	// internal
	sr      gzip.SaverReader
	offset  int64
	counter int64

	checkpoint *GzipSourceCheckpoint
}

type GzipSourceCheckpoint struct {
	Offset           int64
	SourceCheckpoint *savior.SourceCheckpoint
	GzipCheckpoint   *gzip.Checkpoint
}

var _ savior.Source = (*gzipSource)(nil)

func New(source savior.Source, threshold int64) *gzipSource {
	return &gzipSource{
		source:    source,
		threshold: threshold,
	}
}

func (gs *gzipSource) Save() (*savior.SourceCheckpoint, error) {
	if gs.checkpoint != nil {
		c := &savior.SourceCheckpoint{
			Offset: gs.offset,
			Data:   gs.checkpoint,
		}
		return c, nil
	}
	return nil, nil
}

func (gs *gzipSource) Resume(checkpoint *savior.SourceCheckpoint) (int64, error) {
	gs.counter = 0
	gs.checkpoint = nil

	if checkpoint != nil {
		if ourCheckpoint, ok := checkpoint.Data.(*GzipSourceCheckpoint); ok {
			sourceOffset, err := gs.source.Resume(ourCheckpoint.SourceCheckpoint)
			if err != nil {
				return 0, errors.Wrap(err, 0)
			}

			gc := ourCheckpoint.GzipCheckpoint
			if sourceOffset == gc.Roffset {
				gs.sr, err = gc.Resume(gs.source)
				if err != nil {
					savior.Debugf(`gzipsource: could not use gzip checkpoint at R=%d`, gc.Roffset)
					// well, let's start over
					_, err = gs.source.Resume(nil)
					if err != nil {
						return 0, errors.Wrap(err, 0)
					}
				} else {
					gs.offset = ourCheckpoint.Offset
					return gs.offset, nil
				}
			} else {
				savior.Debugf(`gzipsource: expected source to resume at %d but got %d`, gc.Roffset, sourceOffset)
			}
		}
	}

	// start from beginning
	sourceOffset, err := gs.source.Resume(nil)
	if err != nil {
		return 0, errors.Wrap(err, 0)
	}

	if sourceOffset != 0 {
		msg := fmt.Sprintf("gzipsource: expected source to resume at start but got %d", sourceOffset)
		return 0, errors.New(msg)
	}

	gs.sr, err = gzip.NewSaverReader(gs.source)
	if err != nil {
		return 0, err
	}

	gs.offset = 0
	return 0, nil
}

func (gs *gzipSource) Read(buf []byte) (int, error) {
	n, err := gs.sr.Read(buf)
	gs.offset += int64(n)
	gs.counter += int64(n)
	if gs.counter > gs.threshold {
		gs.sr.WantSave()
		gs.counter = 0
	}

	if err != nil {
		if err == flate.ReadyToSaveError {
			gzipCheckpoint, saveErr := gs.sr.Save()
			if saveErr != nil {
				return n, saveErr
			}

			sourceCheckpoint, sourceErr := gs.source.Save()
			if saveErr != nil {
				return n, sourceErr
			}

			savior.Debugf("gzipsource: saving, gzip rOffset = %d, sourceCheckpoint.Offset = %d", gzipCheckpoint.Roffset, sourceCheckpoint.Offset)

			gs.checkpoint = &GzipSourceCheckpoint{
				Offset:           gs.offset,
				GzipCheckpoint:   deepcopy.Copy(gzipCheckpoint).(*gzip.Checkpoint),
				SourceCheckpoint: sourceCheckpoint,
			}

			savior.Debugf("gzipsource: saved checkpoint at byte %d", gs.offset)
			err = nil
		}
	}

	return n, err
}

func (gs *gzipSource) ReadByte() (byte, error) {
	buf := []byte{0}
	_, err := gs.Read(buf)
	return buf[0], err
}

func (gs *gzipSource) Progress() float64 {
	// We can't tell how large the uncompressed stream is until we finish
	// decompressing it. The underlying's source progress is a good enough
	// approximation.
	return gs.source.Progress()
}

func init() {
	gob.Register(&GzipSourceCheckpoint{})
}
