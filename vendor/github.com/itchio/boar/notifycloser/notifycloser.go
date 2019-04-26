package notifycloser

import "io"

type OnCloseFunc func(totalBytes int64) error

type NotifyCloser struct {
	Writer  io.Writer
	OnClose OnCloseFunc

	totalBytes int64
}

var _ io.WriteCloser = (*NotifyCloser)(nil)

func (nc *NotifyCloser) Write(buf []byte) (int, error) {
	written, err := nc.Writer.Write(buf)
	nc.totalBytes += int64(written)
	return written, err
}

func (nc *NotifyCloser) Close() error {
	if closer, ok := nc.Writer.(io.Closer); ok {
		err := closer.Close()
		if err != nil {
			return err
		}
	}

	return nc.OnClose(nc.totalBytes)
}
