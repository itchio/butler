package lrufile

import (
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/golang-lru/simplelru"

	"github.com/pkg/errors"
)

var lruFileDumpStats = os.Getenv("LRU_FILE_DUMP_STATS") == "1"

type File interface {
	io.ReadSeeker
	Reset(rs io.ReadSeeker) error
	Stats() Stats
}

type Stats struct {
	Hits   int64
	Misses int64
}

type lruFile struct {
	chunkSize  int64
	numEntries int

	rs   io.ReadSeeker
	size int64

	offset int64

	storage     []byte
	allocations []int64

	// maps chunkIndex to storageIndex
	lru simplelru.LRUCache

	stats Stats
}

var _ File = (*lruFile)(nil)

func New(chunkSize int64, numEntries int) (File, error) {
	storageSize := chunkSize * int64(numEntries)
	allocations := make([]int64, numEntries)

	lf := &lruFile{
		chunkSize:  chunkSize,
		numEntries: numEntries,

		storage:     make([]byte, storageSize),
		allocations: allocations,
	}
	lru, err := simplelru.NewLRU(numEntries, lf.onEvict)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	lf.lru = lru

	return lf, nil
}

func (lf *lruFile) Reset(rs io.ReadSeeker) error {
	size, err := rs.Seek(0, io.SeekEnd)
	if err != nil {
		return errors.WithStack(err)
	}
	lf.size = size
	lf.offset = 0
	lf.rs = rs

	for i := range lf.allocations {
		lf.allocations[i] = -1
	}
	lf.lru.Purge()

	lf.stats.Hits = 0
	lf.stats.Misses = 0

	// N.B: no need to clear storage!
	// think of a "non-secure" file removal - we just
	// remove the allocation information, not the actual data.

	return nil
}

func (lf *lruFile) Read(buf []byte) (int, error) {
	remaining := int64(len(buf))

	var readBytes int
	var eof = false

	for remaining > 0 {
		chunkIndex := lf.offset / lf.chunkSize

		chunk, err := lf.getChunk(chunkIndex)
		if err != nil {
			return readBytes, errors.WithStack(err)
		}

		start := lf.offset % lf.chunkSize
		end := start + remaining

		chunkStart := chunkIndex * lf.chunkSize
		chunkEnd := chunkStart + lf.chunkSize
		lastChunk := false
		if chunkEnd > lf.size {
			chunkEnd = lf.size
			lastChunk = true
		}
		chunkSize := chunkEnd - chunkStart

		if end > chunkSize {
			end = chunkSize
			if lastChunk {
				eof = true
			}
		}

		copied := copy(buf[readBytes:], chunk[start:end])
		readBytes += copied
		remaining -= int64(copied)
		lf.offset += int64(copied)
		if eof {
			return readBytes, io.EOF
		}
	}

	return readBytes, nil
}

func (lf *lruFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		lf.offset = offset
	case io.SeekCurrent:
		lf.offset += offset
	case io.SeekEnd:
		lf.offset = lf.size + offset
	default:
		return lf.offset, fmt.Errorf("invalid whence: %d", whence)
	}

	if lf.offset < 0 || lf.offset > lf.size {
		invalid := lf.offset
		lf.offset = 0
		return lf.offset, fmt.Errorf("invalid seek to %d, must be in [%d,%d]", invalid, 0, lf.size)
	}

	return lf.offset, nil
}

func (lf *lruFile) Stats() Stats {
	return lf.stats
}

func (lf *lruFile) onEvict(key interface{}, value interface{}) {
	// clear allocation!
	storageIndex := value.(int)
	lf.allocations[storageIndex] = -1
}

func (lf *lruFile) getChunk(chunkIndex int64) ([]byte, error) {
	// do we already have it?
	{
		if v, ok := lf.lru.Get(chunkIndex); ok {
			storageIndex := v.(int)
			lf.stats.Hits++

			// it's a cache hit. cool!
			start := storageIndex * int(lf.chunkSize)
			end := (storageIndex + 1) * int(lf.chunkSize)
			slice := lf.storage[start:end]
			return slice, nil
		}
	}

	lf.stats.Misses++

	// first add -1 to the cache, so it can evict as needed
	lf.lru.Add(chunkIndex, -1)

	var storageIndex = -1

	// find a free storageIndex
	for k, v := range lf.allocations {
		if v < 0 {
			// yay, we found one!
			storageIndex = k
			break
		}
	}
	if storageIndex < 0 {
		return nil, errors.New("internal error: could not find room in lrufile cache")
	}

	// now store actual storage index in cache
	lf.lru.Add(chunkIndex, storageIndex)
	// and mark it as used in allocations
	lf.allocations[storageIndex] = chunkIndex

	// now let's actually read it
	start := storageIndex * int(lf.chunkSize)
	end := (storageIndex + 1) * int(lf.chunkSize)
	slice := lf.storage[start:end]

	inputOffset := chunkIndex * lf.chunkSize

	_, err := lf.rs.Seek(inputOffset, io.SeekStart)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// by contract, an io.Reader reads the entire buffer
	// if it returns less, it should return an error
	_, err = lf.rs.Read(slice)
	if err != nil {
		if err == io.EOF {
			// that's expected! input is rarely a multiple of chunk size
		} else {
			return nil, errors.WithStack(err)
		}
	}

	// slice points to storage, so all the data's there.
	// muffin left to do but return it!
	return slice, nil
}
