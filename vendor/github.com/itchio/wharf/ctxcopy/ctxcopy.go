package ctxcopy

import (
	"context"
	"io"

	"github.com/itchio/wharf/werrors"
)

const threshold int64 = 256 * 1024

func Do(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	return DoBuffer(ctx, dst, src, nil)
}

func DoBuffer(ctx context.Context, dst io.Writer, src io.Reader, buf []byte) (int64, error) {
	if buf == nil {
		buf = make([]byte, 16384)
	}

	var copied int64
	var counter int64

	var eof bool
	for !eof {
		n, err := src.Read(buf)
		if err != nil {
			if err == io.EOF {
				eof = true
			} else {
				return copied, err
			}
		}

		nn, err := dst.Write(buf[:n])
		copied += int64(nn)
		if err != nil {
			return copied, err
		}

		counter += int64(nn)
		if counter > threshold {
			counter = 0
			select {
			case <-ctx.Done():
				return copied, werrors.ErrCancelled
			default:
				// keep going
			}
		}
	}

	return copied, nil
}
