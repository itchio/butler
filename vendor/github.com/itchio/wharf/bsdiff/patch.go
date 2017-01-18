package bsdiff

import (
	"errors"
	"fmt"
	"io"

	"os"

	humanize "github.com/dustin/go-humanize"
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

	for {
		ctrl.Reset()

		err = readMessage(ctrl)
		if err != nil {
			return err
		}

		if ctrl.Eof {
			break
		}

		// Sanity-check
		if newpos+int64(len(ctrl.Add)) > newSize {
			return ErrCorrupt
		}

		// Add old data to diff string
		ar := &AdderReader{
			Buffer: ctrl.Add,
			Reader: old,
		}

		_, err := io.CopyN(new, ar, int64(len(ctrl.Add)))
		if err != nil {
			return err
		}

		// Adjust pointers
		newpos += int64(len(ctrl.Add))
		oldpos += int64(len(ctrl.Add))

		// Sanity-check
		if newpos+int64(len(ctrl.Copy)) > newSize {
			return ErrCorrupt
		}

		// Read extra string
		_, err = new.Write(ctrl.Copy)
		if err != nil {
			return err
		}

		// Adjust pointers
		newpos += int64(len(ctrl.Copy))

		oldpos, err = old.Seek(ctrl.Seek, os.SEEK_CUR)
		if err != nil {
			return err
		}
	}

	if newpos != newSize {
		return fmt.Errorf("bsdiff: expected new file to be %d, was %d (%s difference)", newSize, newpos, humanize.IBytes(uint64(newSize-newpos)))
	}

	return nil
}
