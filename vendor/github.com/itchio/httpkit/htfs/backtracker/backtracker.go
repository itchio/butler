package backtracker

import (
	"bufio"
	"io"

	"github.com/pkg/errors"
)

// Backtracker allows reads from an io.Reader while remembering
// the last N bytes of data. Backtrack() can then be called, to
// read those bytes again.
type Backtracker interface {
	io.Reader

	// Returns the current offset (doesn't count backtracking)
	Offset() int64

	// Return amount of bytes that can be backtracked
	Cached() int64

	// Backtrack n bytes
	Backtrack(n int64) error

	// Advance n bytes
	Discard(n int64) error

	NumCacheHits() int64
	NumCacheMiss() int64

	CachedBytesServed() int64
}

// New returns a Backtracker reading from upstream
func New(offset int64, upstream io.Reader, cacheSize int64) Backtracker {
	return &backtracker{
		upstream:   bufio.NewReader(upstream),
		discardBuf: make([]byte, 256*1024),
		cache:      make([]byte, cacheSize),
		cached:     0,
		backtrack:  0,
		offset:     offset,
	}
}

type backtracker struct {
	upstream   *bufio.Reader
	cache      []byte
	discardBuf []byte
	cached     int
	backtrack  int
	offset     int64

	numCacheHits      int64
	numCacheMiss      int64
	cachedBytesServed int64
}

func (bt *backtracker) NumCacheHits() int64 {
	return bt.numCacheHits
}

func (bt *backtracker) NumCacheMiss() int64 {
	return bt.numCacheMiss
}

func (bt *backtracker) CachedBytesServed() int64 {
	return bt.cachedBytesServed
}

var _ Backtracker = (*backtracker)(nil)

func (bt *backtracker) Read(buf []byte) (int, error) {
	readlen := len(buf)
	cachesize := len(bt.cache)

	// read from cache
	if bt.backtrack > 0 {
		if readlen > bt.backtrack {
			readlen = bt.backtrack
		}

		cache := bt.cache[cachesize-bt.backtrack:]

		copy(buf[:readlen], cache[:readlen])
		bt.backtrack -= readlen
		bt.numCacheHits++
		bt.cachedBytesServed += int64(readlen)
		return readlen, nil
	}

	bt.numCacheMiss++

	// read from upstream
	readlen, err := bt.upstream.Read(buf)

	if readlen > 0 {
		bt.offset += int64(readlen)

		// cache data
		remainingOldCacheSize := cachesize - readlen
		if remainingOldCacheSize > 0 {
			copy(bt.cache[:remainingOldCacheSize], bt.cache[readlen:])
			copy(bt.cache[remainingOldCacheSize:], buf[:readlen])
		} else {
			readbytes := buf[:readlen]
			copy(bt.cache, readbytes[readlen-cachesize:readlen])
		}

		bt.cached += readlen
		if bt.cached > cachesize {
			bt.cached = cachesize
		}
	}

	return readlen, err
}

func (bt *backtracker) Discard(n int64) error {
	discardlen := int64(len(bt.discardBuf))

	for n > 0 {
		readlen := n
		if readlen > discardlen {
			readlen = discardlen
		}

		discarded, err := bt.Read(bt.discardBuf[:readlen])
		if err != nil {
			return errors.WithMessage(err, "discarding")
		}

		n -= int64(discarded)
	}
	return nil
}

func (bt *backtracker) Cached() int64 {
	return int64(bt.cached)
}

func (bt *backtracker) Backtrack(n int64) error {
	if int64(bt.cached) < n {
		return errors.Errorf("only %d cached, can't backtrack by %d", bt.cached, n)
	}
	bt.backtrack = int(n)
	return nil
}

func (bt *backtracker) Offset() int64 {
	return bt.offset
}
