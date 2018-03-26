package brotlisource

import (
	"encoding/gob"
	"fmt"

	"github.com/itchio/dskompress/brotli"
	"github.com/itchio/savior"
	"github.com/pkg/errors"
)

type brotliSource struct {
	// input
	source savior.Source

	// internal
	br      brotli.SaverReader
	offset  int64
	bytebuf []byte

	ssc              savior.SourceSaveConsumer
	sourceCheckpoint *savior.SourceCheckpoint
}

type BrotliSourceCheckpoint struct {
	SourceCheckpoint *savior.SourceCheckpoint
	BrotliCheckpoint *brotli.Checkpoint
}

var _ savior.Source = (*brotliSource)(nil)

func New(source savior.Source) *brotliSource {
	return &brotliSource{
		source:  source,
		bytebuf: []byte{0x00},
	}
}

func (bs *brotliSource) Features() savior.SourceFeatures {
	return savior.SourceFeatures{
		Name:          "brotli",
		ResumeSupport: savior.ResumeSupportBlock,
	}
}

func (bs *brotliSource) SetSourceSaveConsumer(ssc savior.SourceSaveConsumer) {
	savior.Debugf("brotlisource: set source save consumer!")
	bs.ssc = ssc
	bs.source.SetSourceSaveConsumer(&savior.CallbackSourceSaveConsumer{
		OnSave: func(checkpoint *savior.SourceCheckpoint) error {
			savior.Debugf("brotlisource: underlying source gave us checkpoint!")
			bs.sourceCheckpoint = checkpoint
			bs.br.WantSave()
			return nil
		},
	})
}

func (bs *brotliSource) WantSave() {
	savior.Debugf("brotlisource: want save!")
	bs.source.WantSave()
}

func (bs *brotliSource) Resume(checkpoint *savior.SourceCheckpoint) (int64, error) {
	savior.Debugf(`brotlisource: asked to resume`)

	if checkpoint != nil {
		if ourCheckpoint, ok := checkpoint.Data.(*BrotliSourceCheckpoint); ok {
			sourceOffset, err := bs.source.Resume(ourCheckpoint.SourceCheckpoint)
			if err != nil {
				return 0, errors.WithStack(err)
			}

			bc := ourCheckpoint.BrotliCheckpoint
			if sourceOffset < bc.InputOffset {
				delta := bc.InputOffset - sourceOffset
				savior.Debugf(`brotlisource: discarding %d bytes to align source with decompressor`, delta)
				err = savior.DiscardByRead(bs.source, delta)
				if err != nil {
					return 0, errors.WithStack(err)
				}
				sourceOffset += delta
			}

			if sourceOffset == bc.InputOffset {
				bs.br, err = bc.Resume(bs.source)
				if err != nil {
					savior.Debugf(`brotlisource: could not use brotli checkpoint at R=%d / W=%d`, bc.InputOffset, bc.OutputOffset)
					// well, let's start over
					_, err = bs.source.Resume(nil)
					if err != nil {
						return 0, errors.WithStack(err)
					}
				} else {
					bs.offset = bc.OutputOffset
					return bc.OutputOffset, nil
				}
			} else {
				savior.Debugf(`brotlisource: expected source to resume at %d but got %d`, bc.InputOffset, sourceOffset)
			}
		}
	}

	// start from beginning
	sourceOffset, err := bs.source.Resume(nil)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	if sourceOffset != 0 {
		msg := fmt.Sprintf("brotlisource: expected source to resume at start but got %d", sourceOffset)
		return 0, errors.New(msg)
	}

	br, err := brotli.NewSaverReader(bs.source)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	bs.br = br
	bs.offset = 0

	return 0, nil
}

func (bs *brotliSource) Read(buf []byte) (int, error) {
	if bs.br == nil {
		return 0, errors.WithStack(savior.ErrUninitializedSource)
	}

	n, err := bs.br.Read(buf)
	bs.offset += int64(n)

	if err == brotli.ReadyToSaveError {
		err = nil

		if bs.sourceCheckpoint == nil {
			savior.Debugf("brotlisource: can't save, sourceCheckpoint is nil!")
		} else if bs.ssc == nil {
			savior.Debugf("brotlisource: can't save, ssc is nil!")
		} else {
			brotliCheckpoint, saveErr := bs.br.Save()
			if saveErr != nil {
				return n, saveErr
			}

			savior.Debugf("brotlisource: saving, brotli InputOffset = %d, sourceCheckpoint.Offset = %d", brotliCheckpoint.InputOffset, bs.sourceCheckpoint.Offset)

			checkpoint := &savior.SourceCheckpoint{
				Offset: bs.offset,
				Data: &BrotliSourceCheckpoint{
					BrotliCheckpoint: brotliCheckpoint,
					SourceCheckpoint: bs.sourceCheckpoint,
				},
			}
			bs.sourceCheckpoint = nil

			err = bs.ssc.Save(checkpoint)
			savior.Debugf("brotlisource: saved checkpoint at byte %d", bs.offset)
		}
	}

	return n, err
}

func (bs *brotliSource) ReadByte() (byte, error) {
	if bs.br == nil {
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

func (bs *brotliSource) Progress() float64 {
	// We can't tell how large the uncompressed stream is until we finish
	// decompressing it. The underlying's source progress is a good enough
	// approximation.
	return bs.source.Progress()
}

func init() {
	gob.Register(&BrotliSourceCheckpoint{})
}
