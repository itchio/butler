package fspool

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/hashicorp/golang-lru/simplelru"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

const (
	// ModeMask is or'd with the permission files being opened
	ModeMask = 0644
)

type fsEntryReader interface {
	io.ReadSeeker
	io.Closer
}

// FsPool is a filesystem-backed Pool+WritablePool
type FsPool struct {
	container *tlc.Container
	basePath  string

	fileIndex int64
	reader    fsEntryReader

	UniqueReader fsEntryReader
}

var _ wsync.Pool = (*FsPool)(nil)
var _ wsync.WritablePool = (*FsPool)(nil)

// ReadCloseSeeker unifies io.Reader, io.Seeker, and io.Closer
type ReadCloseSeeker interface {
	io.Reader
	io.Seeker
	io.Closer
}

// NewFsPool creates a new FsPool from the given Container
// metadata and a base path on-disk to allow reading from files.
func New(c *tlc.Container, basePath string) *FsPool {
	return &FsPool{
		container: c,
		basePath:  basePath,

		fileIndex: int64(-1),
		reader:    nil,
	}
}

// GetSize returns the size of the file at index fileIndex
func (cfp *FsPool) GetSize(fileIndex int64) int64 {
	return cfp.container.Files[fileIndex].Size
}

// GetRelativePath returns the slashed path of a file, relative to
// the container's root.
func (cfp *FsPool) GetRelativePath(fileIndex int64) string {
	return cfp.container.Files[fileIndex].Path
}

// GetPath returns the native path of a file (with slashes or backslashes)
// on-disk, based on the FsPool's base path
func (cfp *FsPool) GetPath(fileIndex int64) string {
	path := filepath.FromSlash(cfp.container.Files[fileIndex].Path)
	fullPath := filepath.Join(cfp.basePath, path)
	return fullPath
}

// GetReader returns an io.Reader for the file at index fileIndex
// Successive calls to `GetReader` will attempt to re-use the last
// returned reader if the file index is similar. The cache size is 1, so
// reading in parallel from different files is not supported.
func (cfp *FsPool) GetReader(fileIndex int64) (io.Reader, error) {
	return cfp.GetReadSeeker(fileIndex)
}

// GetReadSeeker is like GetReader but the returned object allows seeking
func (cfp *FsPool) GetReadSeeker(fileIndex int64) (io.ReadSeeker, error) {
	if cfp.UniqueReader != nil {
		return cfp.UniqueReader, nil
	}

	if cfp.fileIndex != fileIndex {
		if cfp.reader != nil {
			err := cfp.reader.Close()
			if err != nil {
				return nil, errors.Wrap(err, 1)
			}
			cfp.reader = nil
		}

		ra, err := eos.Open(cfp.GetPath(fileIndex))
		if err != nil {
			return nil, err
		}

		stats, err := ra.Stat()
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		reader := &readSeekerAt{
			r:    ra,
			size: stats.Size(),
		}

		cfp.reader = reader
		cfp.fileIndex = fileIndex
	}

	return cfp.reader, nil
}

// Close closes all reader belonging to this FsPool
func (cfp *FsPool) Close() error {
	if cfp.reader != nil {
		err := cfp.reader.Close()
		if err != nil {
			return errors.Wrap(err, 1)
		}

		cfp.reader = nil
		cfp.fileIndex = -1
	}

	return nil
}

func (cfp *FsPool) GetWriter(fileIndex int64) (io.WriteCloser, error) {
	path := cfp.GetPath(fileIndex)

	err := os.MkdirAll(filepath.Dir(path), os.FileMode(0755))
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	outputFile := cfp.container.Files[fileIndex]
	f, oErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(outputFile.Mode)|ModeMask)
	if oErr != nil {
		return nil, oErr
	}

	return f, nil
}

//

var rsaChunkSize int64 = 16 * 1024 // 16KiB!
var rsaLruSize int = 1
var rsaPrintThreshold float64 = 0.1

func init() {
	if cs, err := strconv.ParseFloat(os.Getenv("RSA_CHUNK_SIZE"), 64); err == nil {
		rsaChunkSize = int64(cs * 1024.0)
		fmt.Printf("setting chunk size to %s\n", humanize.IBytes(uint64(rsaChunkSize)))
	}

	if ls, err := strconv.ParseInt(os.Getenv("RSA_LRU_SIZE"), 10, 64); err == nil {
		rsaLruSize = int(ls)
		fmt.Printf("setting lru size to %d\n", rsaLruSize)
	}

	if ps, err := strconv.ParseFloat(os.Getenv("RSA_PRINT_THRESHOLD"), 64); err == nil {
		rsaPrintThreshold = ps
		fmt.Printf("setting print threshold to %f seconds\n", rsaPrintThreshold)
	}
}

type readSeekerAt struct {
	r    io.ReaderAt
	size int64

	offset int64

	smallestRead int
	largestRead  int
	totalRead    float64
	totalReads   int64

	smallestJumpback int
	largestJumpback  int
	totalJumpback    float64
	jumpBacks        int64

	chunkReads      int64
	totalChunkReads int64
	lru             *simplelru.LRU

	firstRead time.Time
}

var _ fsEntryReader = (*readSeekerAt)(nil)

func (rsa *readSeekerAt) Read(buf []byte) (int, error) {
	if rsa.firstRead.IsZero() {
		rsa.firstRead = time.Now()
		var err error
		rsa.lru, err = simplelru.NewLRU(rsaLruSize, nil)
		if err != nil {
			return 0, errors.Wrap(err, 0)
		}
	}

	min := rsa.offset
	max := rsa.offset + int64(len(buf))

	minChunk := min / rsaChunkSize
	maxChunk := (max - 1) / rsaChunkSize

	for ch := minChunk; ch <= maxChunk; ch++ {
		rsa.totalChunkReads++
		if !rsa.lru.Contains(ch) {
			rsa.chunkReads++
			rsa.lru.Add(ch, true)
		}
	}

	if rsa.smallestRead == 0 || len(buf) < rsa.smallestRead {
		rsa.smallestRead = len(buf)
	}
	if rsa.largestRead == 0 || len(buf) > rsa.largestRead {
		rsa.largestRead = len(buf)
	}
	rsa.totalRead += float64(len(buf))
	rsa.totalReads++

	n, err := rsa.r.ReadAt(buf, rsa.offset)

	rsa.offset += int64(n)
	return n, err
}

func (rsa *readSeekerAt) Seek(offset int64, whence int) (int64, error) {
	oldoffset := rsa.offset

	switch whence {
	case io.SeekStart:
		rsa.offset = offset
	case io.SeekCurrent:
		rsa.offset += offset
	case io.SeekEnd:
		rsa.offset = rsa.size + offset
	}

	if rsa.offset < oldoffset {
		delta := int(oldoffset - rsa.offset)
		if rsa.smallestJumpback == 0 || delta < rsa.smallestJumpback {
			rsa.smallestJumpback = delta
		}
		if rsa.largestJumpback == 0 || delta > rsa.largestJumpback {
			rsa.largestJumpback = delta
		}
		rsa.totalJumpback += float64(delta)
		rsa.jumpBacks++
	}

	return rsa.offset, nil
}

func (rsa *readSeekerAt) Close() error {
	duration := time.Since(rsa.firstRead)
	if !rsa.firstRead.IsZero() && duration.Seconds() > rsaPrintThreshold {
		// rmean := rsa.totalRead / float64(rsa.totalReads)
		// jmean := rsa.totalJumpback / float64(rsa.jumpBacks)
		// fmt.Printf("%10d reads\t%10d rmin\t%10d rmax\t%10.0f rmean\t%10d jumpbacks\t%10d jmin\t%10d jmax\t%10.0f jmean\t%s total\t%s duration\n",
		// 	rsa.totalReads, rsa.smallestRead, rsa.largestRead, rmean,
		// 	rsa.jumpBacks, rsa.smallestJumpback, rsa.largestJumpback, jmean,
		// 	humanize.IBytes(uint64(rsa.size)),
		// 	time.Since(rsa.firstRead))
		fmt.Printf("%10d reads\t%10d chunk reads\t%10.2fx read calls\t%10s filesize\t%10s read\t%10.2fx data read\t%10.2f%% HIT ratio\t%s duration\n",
			rsa.totalReads,
			rsa.chunkReads,
			float64(rsa.chunkReads)/float64(rsa.totalReads),
			humanize.IBytes(uint64(rsa.size)),
			humanize.IBytes(uint64(rsa.chunkReads*rsaChunkSize)),
			float64(rsa.chunkReads*rsaChunkSize)/float64(rsa.size),
			float64(rsa.totalChunkReads-rsa.chunkReads)/float64(rsa.totalChunkReads)*100,
			time.Since(rsa.firstRead),
		)
	}

	if c, ok := rsa.r.(io.Closer); ok {
		return c.Close()
	}

	return nil
}
