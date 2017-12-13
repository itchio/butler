package szah

import "io"

type onCloseFunc func(totalBytes int64) error

type notifyCloser struct {
	Writer  io.Writer
	OnClose onCloseFunc

	totalBytes int64
}

var _ io.WriteCloser = (*notifyCloser)(nil)

func (nc *notifyCloser) Write(buf []byte) (int, error) {
	written, err := nc.Writer.Write(buf)
	nc.totalBytes += int64(written)
	return written, err
}

func (nc *notifyCloser) Close() error {
	if closer, ok := nc.Writer.(io.Closer); ok {
		err := closer.Close()
		if err != nil {
			return err
		}
	}

	return nc.OnClose(nc.totalBytes)
}
