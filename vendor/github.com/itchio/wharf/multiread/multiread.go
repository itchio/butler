package multiread

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/itchio/wharf/ctxcopy"
)

type multiread struct {
	upstream  io.Reader
	writers   []*io.PipeWriter
	doing     bool
	doingLock sync.Mutex
}

// Multiread lets multiple readers read the same data
type Multiread interface {
	Reader() io.Reader
	Do(ctx context.Context) error
}

// New returns a new instance of Multiread
// reading from upstream
func New(upstream io.Reader) Multiread {
	return &multiread{upstream: upstream}
}

func (m *multiread) Reader() io.Reader {
	m.doingLock.Lock()
	defer m.doingLock.Unlock()
	if m.doing {
		return &errReader{err: errors.New("multiread: cannot call Reader() after Do()")}
	}

	r, w := io.Pipe()
	m.writers = append(m.writers, w)
	return r
}

func (m *multiread) Do(ctx context.Context) error {
	m.doingLock.Lock()
	m.doing = true
	m.doingLock.Unlock()

	var closeOnce sync.Once

	defer closeOnce.Do(func() {
		for _, w := range m.writers {
			w.Close()
		}
	})

	ww := make([]io.Writer, 0, len(m.writers))
	for _, w := range m.writers {
		ww = append(ww, w)
	}
	mw := io.MultiWriter(ww...)

	_, err := ctxcopy.Do(ctx, mw, m.upstream)
	if err != nil {
		closeOnce.Do(func() {
			for _, w := range m.writers {
				w.CloseWithError(err)
			}
		})
	}
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
