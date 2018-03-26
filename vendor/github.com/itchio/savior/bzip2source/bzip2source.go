package bzip2source

import (
	"encoding/gob"
	"fmt"

	"github.com/itchio/kompress/bzip2"
	"github.com/itchio/savior"
	"github.com/pkg/errors"
)

type bzip2Source struct {
	// input
	source savior.Source

	// internal
	sr      bzip2.SaverReader
	offset  int64
	counter int64
	bytebuf []byte

	ssc              savior.SourceSaveConsumer
	sourceCheckpoint *savior.SourceCheckpoint
}

type Bzip2SourceCheckpoint struct {
	Offset           int64
	SourceCheckpoint *savior.SourceCheckpoint
	Bzip2Checkpoint  *bzip2.Checkpoint
}

var _ savior.Source = (*bzip2Source)(nil)

func New(source savior.Source) *bzip2Source {
	return &bzip2Source{
		source:  source,
		bytebuf: []byte{0x00},
	}
}

func (bs *bzip2Source) Features() savior.SourceFeatures {
	return savior.SourceFeatures{
		Name:          "bzip2",
		ResumeSupport: savior.ResumeSupportBlock,
	}
}

func (bs *bzip2Source) SetSourceSaveConsumer(ssc savior.SourceSaveConsumer) {
	savior.Debugf("bzip2: set source save consumer!")
	bs.ssc = ssc
	bs.source.SetSourceSaveConsumer(&savior.CallbackSourceSaveConsumer{
		OnSave: func(checkpoint *savior.SourceCheckpoint) error {
			savior.Debugf("bzip2: on save!")
			bs.sourceCheckpoint = checkpoint
			bs.sr.WantSave()
			return nil
		},
	})
}

func (bs *bzip2Source) WantSave() {
	savior.Debugf("bzip2: want save!")
	bs.source.WantSave()
}

func (bs *bzip2Source) Resume(checkpoint *savior.SourceCheckpoint) (int64, error) {
	savior.Debugf(`bzip2: asked to resume`)

	if checkpoint != nil {
		if ourCheckpoint, ok := checkpoint.Data.(*Bzip2SourceCheckpoint); ok {
			sourceOffset, err := bs.source.Resume(ourCheckpoint.SourceCheckpoint)
			if err != nil {
				return 0, errors.WithStack(err)
			}

			bc := ourCheckpoint.Bzip2Checkpoint
			if sourceOffset < bc.Roffset {
				delta := bc.Roffset - sourceOffset
				savior.Debugf(`bzip2source: discarding %d bytes to align source with decompressor`, delta)
				err = savior.DiscardByRead(bs.source, delta)
				if err != nil {
					return 0, errors.WithStack(err)
				}
				sourceOffset += delta
			}

			if sourceOffset == bc.Roffset {
				bs.sr, err = bc.Resume(bs.source)
				if err != nil {
					savior.Debugf(`bzip2source: could not use bzip2 checkpoint at R=%d`, bc.Roffset)
					// well, let's start over
					_, err = bs.source.Resume(nil)
					if err != nil {
						return 0, errors.WithStack(err)
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
		return 0, errors.WithStack(err)
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
	if bs.sr == nil {
		return 0, errors.WithStack(savior.ErrUninitializedSource)
	}

	n, err := bs.sr.Read(buf)
	bs.offset += int64(n)

	if err == bzip2.ReadyToSaveError {
		err = nil

		if bs.sourceCheckpoint == nil {
			savior.Debugf("bzip2source: can't save, sourceCheckpoint is nil!")
		} else if bs.ssc == nil {
			savior.Debugf("bzip2source: can't save, ssc is nil!")
		} else {
			bzip2Checkpoint, saveErr := bs.sr.Save()
			if saveErr != nil {
				return n, saveErr
			}

			savior.Debugf("bzip2source: saving, bzip2 rOffset = %d, sourceCheckpoint.Offset = %d", bzip2Checkpoint.Roffset, bs.sourceCheckpoint.Offset)

			checkpoint := &savior.SourceCheckpoint{
				Offset: bs.offset,
				Data: &Bzip2SourceCheckpoint{
					Offset:           bs.offset,
					Bzip2Checkpoint:  bzip2Checkpoint,
					SourceCheckpoint: bs.sourceCheckpoint,
				},
			}

			err = bs.ssc.Save(checkpoint)
			savior.Debugf("bzip2source: saved checkpoint at byte %d", bs.offset)
		}
	}

	return n, err
}

func (bs *bzip2Source) ReadByte() (byte, error) {
	if bs.sr == nil {
		return 0, errors.WithStack(savior.ErrUninitializedSource)
	}

	n, err := bs.Read(bs.bytebuf)
	if n == 0 {
		/* this happens when Read needs to save, but it swallows the error */
		/* we're not meant to surface them, but there's no way to handle a */
		/* short read from ReadByte, so we just read again */
		n, err = bs.Read(bs.bytebuf)
	}

	return bs.bytebuf[0], err
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
