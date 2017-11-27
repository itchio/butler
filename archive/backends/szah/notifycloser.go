package szah

import "io"

type OnCloseFunc func(totalBytes int64)

type notifyCloser struct {
	Writer  io.WriteCloser
	OnClose OnCloseFunc

	totalBytes int64
}

var _ io.WriteCloser = (*notifyCloser)(nil)

func (nc *notifyCloser) Write(buf []byte) (int, error) {
	written, err := nc.Writer.Write(buf)
	nc.totalBytes += int64(written)
	return written, err
}

func (nc *notifyCloser) Close() error {
	nc.OnClose(nc.totalBytes)
	return nc.Writer.Close()
}
