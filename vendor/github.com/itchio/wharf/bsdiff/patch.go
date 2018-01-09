package bsdiff

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"os"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/golang/protobuf/proto"
	"github.com/itchio/wharf/bsdiff/lrufile"
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

// Patch applies patch to old, according to the bspatch algorithm,
// and writes the result to new.
func (ctx *PatchContext) Patch(oldorig io.ReadSeeker, new io.Writer, newSize int64, readMessage ReadMessageFunc) error {
	if ctx.lruFile == nil {
		// let's commandeer 32MiB of memory to avoid too many syscalls.
		// these values found empirically: https://twitter.com/fasterthanlime/status/950823147472850950
		// but also, 32K is golang's default copy size.
		const lruChunkSize int64 = 32 * 1024
		const lruNumEntries = 1024

		var err error
		ctx.lruFile, err = lrufile.New(lruChunkSize, lruNumEntries)

		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	const minBufferSize = 32 * 1024 // golang's io.Copy default szie
	if len(ctx.buffer) < minBufferSize {
		ctx.buffer = make([]byte, minBufferSize)
	}
	buffer := ctx.buffer

	var old io.ReadSeeker
	if os.Getenv("BUTLER_IN_MEMORY") == "1" {
		buf, err := ioutil.ReadAll(oldorig)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		old = bytes.NewReader(buf)
	} else {
		old = oldorig
	}

	var oldpos, newpos int64
	var err error

	ctrl := &Control{}

	for {
		ctrl.Reset()

		err = readMessage(ctrl)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if ctrl.Eof {
			break
		}

		// Sanity-check
		if newpos+int64(len(ctrl.Add)) > newSize {
			return errors.Wrap(ErrCorrupt, 0)
		}

		// Add old data to diff string
		ar := &AdderReader{
			Buffer: ctrl.Add,
			Reader: old,
		}

		_, err := io.CopyBuffer(new, io.LimitReader(ar, int64(len(ctrl.Add))), buffer)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		// Adjust pointers
		newpos += int64(len(ctrl.Add))
		oldpos += int64(len(ctrl.Add))

		// Sanity-check
		if newpos+int64(len(ctrl.Copy)) > newSize {
			return errors.Wrap(ErrCorrupt, 0)
		}

		// Read extra string
		_, err = new.Write(ctrl.Copy)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		// Adjust pointers
		newpos += int64(len(ctrl.Copy))

		oldpos, err = old.Seek(ctrl.Seek, os.SEEK_CUR)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	if newpos != newSize {
		return fmt.Errorf("bsdiff: expected new file to be %d, was %d (%s difference)", newSize, newpos, humanize.IBytes(uint64(newSize-newpos)))
	}

	return nil
}
