package overlay

import (
	"bufio"
	"encoding/gob"
	"io"

	"github.com/go-errors/errors"
)

type OverlayPatchContext struct {
	buf []byte
}

const overlayPatchBufSize = 32 * 1024

func (ctx *OverlayPatchContext) Patch(r io.Reader, w io.WriteSeeker) error {
	br := bufio.NewReader(r)

	// it's imperative that we buffer here, or gob.Decoder will
	// make its own bufio.Reader and everything will break
	decoder := gob.NewDecoder(br)
	op := &OverlayOp{}

	for {
		// reset op
		op.Skip = 0
		op.Fresh = 0
		op.Eof = false

		err := decoder.Decode(op)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		switch {
		case op.Eof:
			// cool, we're done!
			return nil

		case op.Skip > 0:
			_, err = w.Seek(op.Skip, io.SeekCurrent)
			if err != nil {
				return errors.Wrap(err, 0)
			}

		case op.Fresh > 0:
			if len(ctx.buf) < overlayPatchBufSize {
				ctx.buf = make([]byte, overlayPatchBufSize)
			}

			_, err = io.CopyBuffer(w, io.LimitReader(br, op.Fresh), ctx.buf)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
	}
}
