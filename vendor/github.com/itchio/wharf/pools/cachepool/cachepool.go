package cachepool

import (
	"io"

	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
)

type CachePool struct {
	container *tlc.Container
	source    wsync.Pool
	cache     wsync.WritablePool

	fileChans []chan struct{}
}

var _ wsync.Pool = (*CachePool)(nil)

// New creates a cachepool that reads from source and stores in
// cache as an intermediary
func New(c *tlc.Container, source wsync.Pool, cache wsync.WritablePool) *CachePool {
	cp := &CachePool{
		container: c,
		source:    source,
		cache:     cache,
		fileChans: make([]chan struct{}, len(c.Files)),
	}

	for i := range cp.fileChans {
		cp.fileChans[i] = make(chan struct{})
	}

	return cp
}

// Preload immediately starts copying from source to cache.
// if it returns nil, all future GetRead{Seek,}er calls for
// this index will succeed (and all pending calls will unblock)
func (cp *CachePool) Preload(fileIndex int64) error {
	channel := cp.fileChans[int(fileIndex)]

	select {
	case <-channel:
		// already preloaded, all done!
		return nil
	default:
		// need to preload now
	}

	success := false
	defer func() {
		if success {
			close(channel)
		}
	}()

	reader, err := cp.source.GetReader(fileIndex)
	if err != nil {
		return errors.WithStack(err)
	}
	defer cp.source.Close()

	writer, err := cp.cache.GetWriter(fileIndex)
	if err != nil {
		return errors.WithStack(err)
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	if err != nil {
		return errors.WithStack(err)
	}

	success = true

	return nil
}

// GetReader returns a reader for the file at index fileIndex,
// once the file has been preloaded successfully.
func (cp *CachePool) GetReader(fileIndex int64) (io.Reader, error) {
	// this will block until the channel is closed,
	// or immediately succeed if it's already closed
	<-cp.fileChans[fileIndex]

	return cp.cache.GetReader(fileIndex)
}

// GetReadSeeker is a version of GetReader that returns an io.ReadSeeker
func (cp *CachePool) GetReadSeeker(fileIndex int64) (io.ReadSeeker, error) {
	// this will block until the channel is closed,
	// or immediately succeed if it's already closed
	<-cp.fileChans[fileIndex]

	return cp.cache.GetReadSeeker(fileIndex)
}

func (cp *CachePool) GetSize(fileIndex int64) int64 {
	return cp.container.Files[fileIndex].Size
}

// Close attempts to close both the source and the cache
// and relays any error it encounters
func (cp *CachePool) Close() error {
	err := cp.source.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	err = cp.cache.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
