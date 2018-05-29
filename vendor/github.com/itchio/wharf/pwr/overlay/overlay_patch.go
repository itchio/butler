package overlay

import (
	"io"

	"github.com/itchio/savior"
	"github.com/itchio/wharf/wire"

	"github.com/pkg/errors"
)

type OverlayPatchContext struct{}

const overlayPatchBufSize = 32 * 1024

func (ctx *OverlayPatchContext) Patch(r savior.Source, w io.WriteSeeker) error {
	// it's imperative that we buffer here, or gob.Decoder will
	// make its own bufio.Reader and everything will break
	rctx := wire.NewReadContext(r)
	op := &OverlayOp{}

	err := rctx.ExpectMagic(OverlayMagic)
	if err != nil {
		return err
	}

	for {
		op.Reset()

		err := rctx.ReadMessage(op)
		if err != nil {
			return errors.WithStack(err)
		}

		switch op.Type {
		case OverlayOp_HEY_YOU_DID_IT:
			// cool, we're done!
			return nil

		case OverlayOp_SKIP:
			_, err = w.Seek(op.Len, io.SeekCurrent)
			if err != nil {
				return errors.WithStack(err)
			}

		case OverlayOp_FRESH:
			_, err := w.Write(op.Data)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}
}
