package bsdiff

import (
	"fmt"
	"io"

	"os"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/golang/protobuf/proto"
	"github.com/itchio/wharf/bsdiff/lrufile"
	"github.com/itchio/wharf/counter"
)

// ErrCorrupt indicates that a patch is corrupted, most often that it would produce a longer file
// than specified
var ErrCorrupt = errors.New("corrupt patch")

// ReadMessageFunc should read the passed protobuf and relay any errors.
// See the `wire` package for an example implementation.
type ReadMessageFunc func(msg proto.Message) error

type PatchContext struct {
	buffer []byte
	lf     lrufile.File
}

func NewPatchContext() *PatchContext {
	return &PatchContext{}
}

var useLru = os.Getenv("BUTLER_LRU") == "1"
var showLruStats = os.Getenv("BUTLER_LRU_STATS") == "1"

// Patch applies patch to old, according to the bspatch algorithm,
// and writes the result to new.
func (ctx *PatchContext) Patch(oldorig io.ReadSeeker, neworig io.Writer, newSize int64, readMessage ReadMessageFunc) error {
	if ctx.lf == nil {
		// let's commandeer 32MiB of memory to avoid too many syscalls.
		// these values found empirically: https://twitter.com/fasterthanlime/status/950823147472850950
		// but also, 32K is golang's default copy size.
		const lruChunkSize int64 = 32 * 1024
		const lruNumEntries = 1024

		var err error
		ctx.lf, err = lrufile.New(lruChunkSize, lruNumEntries)

		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	const minBufferSize = 32 * 1024 // golang's io.Copy default szie
	if len(ctx.buffer) < minBufferSize {
		ctx.buffer = make([]byte, minBufferSize)
	}
	buffer := ctx.buffer

	var old io.ReadSeeker
	if useLru {
		err := ctx.lf.Reset(oldorig)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		old = ctx.lf
	} else {
		old = oldorig
	}

	var err error

	ctrl := &Control{}

	new := counter.NewWriter(neworig)

	for {
		ctrl.Reset()

		err = readMessage(ctrl)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if ctrl.Eof {
			break
		}

		// Add old data to diff string
		addlen := len(ctrl.Add)
		if addlen > 0 {
			ar := &AdderReader{
				Buffer: ctrl.Add,
				Reader: old,
			}

			copied, err := io.CopyBuffer(new, io.LimitReader(ar, int64(addlen)), buffer)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			if copied != int64(addlen) {
				return errors.Wrap(fmt.Errorf("bsdiff-add: expected to copy %d bytes but copied %d", addlen, copied), 0)
			}
		}

		// Read extra string
		copylen := len(ctrl.Copy)
		if copylen > 0 {
			copied, err := new.Write(ctrl.Copy)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			if copied != copylen {
				return errors.Wrap(fmt.Errorf("bsdiff-copy: expected to copy %d bytes but copied %d", addlen, copied), 0)
			}
		}

		_, err = old.Seek(ctrl.Seek, os.SEEK_CUR)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	if new.Count() != newSize {
		return fmt.Errorf("bsdiff: expected new file to be %d, was %d (%s difference)", newSize, new.Count(), humanize.IBytes(uint64(newSize-new.Count())))
	}

	if useLru && showLruStats {
		s := ctx.lf.Stats()
		hitRate := float64(s.Hits) / float64(s.Hits+s.Misses)
		fmt.Printf("%.2f%% hit rate\n", hitRate*100)
	}

	return nil
}
