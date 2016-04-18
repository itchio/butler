// Package wdec provides a NewBrotliReader that works around
// https://github.com/kothar/brotli-go/issues/32 until the fix is pulled
package wdec

import (
	"io"

	"gopkg.in/kothar/brotli-go.v0/dec"
)

type fixedBrotliReader struct {
	reader io.Reader
}

var _ io.Reader = (*fixedBrotliReader)(nil)

// NewBrotliReader returns a brotli reader with proper EOF behavior
func NewBrotliReader(reader io.Reader) io.Reader {
	return &fixedBrotliReader{
		reader: dec.NewBrotliReader(reader),
	}
}

func (fbr *fixedBrotliReader) Read(buffer []byte) (int, error) {
	n, err := fbr.reader.Read(buffer)
	if err == io.EOF && n > 0 {
		return n, nil
	}
	return n, err
}
