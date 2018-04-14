package multiread

import (
	"errors"
	"io"
)

type multiread struct {
	upstream io.Reader
	writers  []io.WriteCloser
	doing    bool
}

// Multiread lets multiple readers read the same data
type Multiread interface {
	Reader() io.Reader
	Do() error
}

// New returns a new instance of Multiread
// reading from upstream
func New(upstream io.Reader) Multiread {
	return &multiread{upstream: upstream}
}

func (m *multiread) Reader() io.Reader {
	if m.doing {
		return &errReader{err: errors.New("multiread: cannot call Reader() after Do()")}
	}

	r, w := io.Pipe()
	m.writers = append(m.writers, w)
	return r
}

func (m *multiread) Do() error {
	m.doing = true

	defer func() {
		for _, w := range m.writers {
			w.Close()
		}
	}()

	ww := make([]io.Writer, 0, len(m.writers))
	for _, w := range m.writers {
		ww = append(ww, w)
	}
	mw := io.MultiWriter(ww...)

	_, err := io.Copy(mw, m.upstream)
	return err
}

// errReader

type errReader struct {
	err error
}

var _ io.Reader = (*errReader)(nil)

func (er *errReader) Read(buf []byte) (int, error) {
	return 0, er.err
}
