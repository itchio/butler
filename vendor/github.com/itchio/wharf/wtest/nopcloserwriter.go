package wtest

import "io"

type nopWriteCloser struct {
	writer io.Writer
}

var _ io.WriteCloser = (*nopWriteCloser)(nil)

func (nwc *nopWriteCloser) Write(buf []byte) (int, error) {
	return nwc.writer.Write(buf)
}

func (nwc *nopWriteCloser) Close() error {
	return nil
}

// NopWriteCloser returns an io.WriteCloser that does
// nothing on Close - never returns an error, doesn't call
// the underlying writer's Close method, even if it did implement it
func NopWriteCloser(w io.Writer) io.WriteCloser {
	return &nopWriteCloser{w}
}
