package bsdiff

import (
	"fmt"
	"io"

	"os"

	"github.com/golang/protobuf/proto"
	"github.com/itchio/httpkit/progress"
	"github.com/itchio/wharf/bsdiff/lrufile"
	"github.com/itchio/wharf/counter"
	"github.com/pkg/errors"
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

var showLruStats = os.Getenv("BUTLER_LRU_STATS") == "1"

type IndividualPatchContext struct {
	parent    *PatchContext
	OldOffset int64
	out       io.Writer
}

func (ctx *PatchContext) NewIndividualPatchContext(old io.ReadSeeker, oldOffset int64, out io.Writer) (*IndividualPatchContext, error) {
	// allocate buffer if needed
	const minBufferSize = 32 * 1024 // golang's io.Copy default szie
	if len(ctx.buffer) < minBufferSize {
		ctx.buffer = make([]byte, minBufferSize)
	}

	// allocate lruFile if needed
	if ctx.lf == nil {
		// let's commandeer 32MiB of memory to avoid too many syscalls.
		// these values found empirically: https://twitter.com/fasterthanlime/status/950823147472850950
		// but also, 32K is golang's default copy size.
		const lruChunkSize int64 = 32 * 1024
		const lruNumEntries = 1024

		var err error
		ctx.lf, err = lrufile.New(lruChunkSize, lruNumEntries)

		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	err := ctx.lf.Reset(old)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ipc := &IndividualPatchContext{
		parent:    ctx,
		OldOffset: oldOffset,
		out:       out,
	}
	return ipc, nil
}

func (ipc *IndividualPatchContext) Apply(ctrl *Control) error {
	buffer := ipc.parent.buffer

	old := ipc.parent.lf
	_, err := old.Seek(ipc.OldOffset, io.SeekStart)
	if err != nil {
		return errors.WithStack(err)
	}

	// Add old data to diff string
	addlen := len(ctrl.Add)
	if addlen > 0 {
		ar := &AdderReader{
			Buffer: ctrl.Add,
			Reader: old,
		}

		copied, err := io.CopyBuffer(ipc.out, io.LimitReader(ar, int64(addlen)), buffer)
		if err != nil {
			return errors.WithStack(err)
		}

		if copied != int64(addlen) {
			return errors.Errorf("bsdiff-add: expected to copy %d bytes but copied %d", addlen, copied)
		}

		ipc.OldOffset += int64(addlen)
	}

	// Read extra string
	copylen := len(ctrl.Copy)
	if copylen > 0 {
		copied, err := ipc.out.Write(ctrl.Copy)
		if err != nil {
			return errors.WithStack(err)
		}

		if copied != copylen {
			return errors.Errorf("bsdiff-copy: expected to copy %d bytes but copied %d", addlen, copied)
		}
	}

	ipc.OldOffset += ctrl.Seek

	return nil
}

// Patch applies patch to old, according to the bspatch algorithm,
// and writes the result to new.
func (ctx *PatchContext) Patch(old io.ReadSeeker, out io.Writer, newSize int64, readMessage ReadMessageFunc) error {
	countingOut := counter.NewWriter(out)

	ipc, err := ctx.NewIndividualPatchContext(old, 0, countingOut)
	if err != nil {
		return errors.WithStack(err)
	}

	ctrl := &Control{}

	for {
		err = readMessage(ctrl)
		if err != nil {
			return errors.WithStack(err)
		}

		if ctrl.Eof {
			break
		}

		err := ipc.Apply(ctrl)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if countingOut.Count() != newSize {
		return fmt.Errorf("bsdiff: expected new file to be %d, was %d (%s difference)", newSize, countingOut.Count(), progress.FormatBytes(newSize-countingOut.Count()))
	}

	if showLruStats {
		s := ctx.lf.Stats()
		hitRate := float64(s.Hits) / float64(s.Hits+s.Misses)
		fmt.Printf("%.2f%% hit rate\n", hitRate*100)
	}

	return nil
}
