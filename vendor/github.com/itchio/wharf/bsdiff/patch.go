package bsdiff

import (
	"fmt"
	"io"

	"os"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/golang/protobuf/proto"
)

// ErrCorrupt indicates that a patch is corrupted, most often that it would produce a longer file
// than specified
var ErrCorrupt = errors.New("corrupt patch")

// ReadMessageFunc should read the passed protobuf and relay any errors.
// See the `wire` package for an example implementation.
type ReadMessageFunc func(msg proto.Message) error

// Patch applies patch to old, according to the bspatch algorithm,
// and writes the result to new.
func Patch(old io.ReadSeeker, new io.Writer, newSize int64, readMessage ReadMessageFunc) error {
	var oldpos, newpos int64
	var err error

	ctrl := &Control{}

	messageNum := 0
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

		_, err := io.CopyN(new, ar, int64(len(ctrl.Add)))
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

		// fmt.Fprintf(os.Stderr, "Seeking %d, oldpos = %d (message %d)\n", ctrl.Seek, oldpos, messageNum)
		oldpos, err = old.Seek(ctrl.Seek, os.SEEK_CUR)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		messageNum++
	}

	if newpos != newSize {
		return fmt.Errorf("bsdiff: expected new file to be %d, was %d (%s difference)", newSize, newpos, humanize.IBytes(uint64(newSize-newpos)))
	}

	return nil
}
