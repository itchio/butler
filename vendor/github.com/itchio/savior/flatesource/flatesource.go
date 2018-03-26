package flatesource

import (
	"encoding/gob"
	"fmt"

	"github.com/itchio/kompress/flate"
	"github.com/itchio/savior"
	"github.com/pkg/errors"
)

type flateSource struct {
	// input
	source savior.Source

	// internal
	sr      flate.SaverReader
	offset  int64
	counter int64
	bytebuf []byte

	ssc              savior.SourceSaveConsumer
	sourceCheckpoint *savior.SourceCheckpoint
}

type FlateSourceCheckpoint struct {
	SourceCheckpoint *savior.SourceCheckpoint
	FlateCheckpoint  *flate.Checkpoint
}

var _ savior.Source = (*flateSource)(nil)

func New(source savior.Source) *flateSource {
	return &flateSource{
		source:  source,
		bytebuf: []byte{0x00},
	}
}

func (fs *flateSource) Features() savior.SourceFeatures {
	return savior.SourceFeatures{
		Name:          "flate",
		ResumeSupport: savior.ResumeSupportBlock,
	}
}

func (fs *flateSource) SetSourceSaveConsumer(ssc savior.SourceSaveConsumer) {
	fs.ssc = ssc
	fs.source.SetSourceSaveConsumer(&savior.CallbackSourceSaveConsumer{
		OnSave: func(checkpoint *savior.SourceCheckpoint) error {
			fs.sourceCheckpoint = checkpoint
			fs.sr.WantSave()
			return nil
		},
	})
}

func (fs *flateSource) WantSave() {
	fs.source.WantSave()
}

func (fs *flateSource) Resume(checkpoint *savior.SourceCheckpoint) (int64, error) {
	savior.Debugf(`flate: asked to resume`)

	if checkpoint != nil {
		if ourCheckpoint, ok := checkpoint.Data.(*FlateSourceCheckpoint); ok {
			sourceOffset, err := fs.source.Resume(ourCheckpoint.SourceCheckpoint)
			if err != nil {
				return 0, errors.WithStack(err)
			}

			fc := ourCheckpoint.FlateCheckpoint
			if sourceOffset < fc.Roffset {
				delta := fc.Roffset - sourceOffset
				savior.Debugf(`flatesource: discarding %d bytes to align source with decompressor`, delta)
				err = savior.DiscardByRead(fs.source, delta)
				if err != nil {
					return 0, errors.WithStack(err)
				}
				sourceOffset += delta
			}

			if sourceOffset == fc.Roffset {
				fs.sr, err = fc.Resume(fs.source)
				if err != nil {
					savior.Debugf(`flatesource: could not use flate checkpoint at R=%d / W=%d`, fc.Roffset, fc.Woffset)
					// well, let's start over
					_, err = fs.source.Resume(nil)
					if err != nil {
						return 0, errors.WithStack(err)
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
		return 0, errors.WithStack(err)
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
	if fs.sr == nil {
		return 0, errors.WithStack(savior.ErrUninitializedSource)
	}

	n, err := fs.sr.Read(buf)
	fs.offset += int64(n)

	if err == flate.ReadyToSaveError {
		err = nil

		if fs.sourceCheckpoint == nil {
			savior.Debugf("flatesource: can't save, sourceCheckpoint is nil!")
		} else if fs.ssc == nil {
			savior.Debugf("flatesource: can't save, ssc is nil!")
		} else {
			flateCheckpoint, saveErr := fs.sr.Save()
			if saveErr != nil {
				return n, saveErr
			}

			savior.Debugf("flatesource: saving, flate rOffset = %d, sourceCheckpoint.Offset = %d", flateCheckpoint.Roffset, fs.sourceCheckpoint.Offset)

			checkpoint := &savior.SourceCheckpoint{
				Offset: fs.offset,
				Data: &FlateSourceCheckpoint{
					FlateCheckpoint:  flateCheckpoint,
					SourceCheckpoint: fs.sourceCheckpoint,
				},
			}
			fs.sourceCheckpoint = nil

			err = fs.ssc.Save(checkpoint)
			savior.Debugf("flatesource: saved checkpoint at byte %d", fs.offset)
		}
	}

	return n, err
}

func (fs *flateSource) ReadByte() (byte, error) {
	if fs.sr == nil {
		return 0, errors.WithStack(savior.ErrUninitializedSource)
	}

	n, err := fs.Read(fs.bytebuf)
	if n == 0 {
		/* this happens when Read needs to save, but it swallows the error */
		/* we're not meant to surface them, but there's no way to handle a */
		/* short read from ReadByte, so we just read again */
		n, err = fs.Read(fs.bytebuf)
	}

	return fs.bytebuf[0], err
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
