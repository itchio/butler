package bsdiff

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"

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
func Patch(old io.Reader, new io.Writer, newSize int64, readMessage ReadMessageFunc) error {
	// TODO: still debating whether we should take an io.ReadSeeker instead... probably?
	// the consumer can still do `ReadAll` themselves and pass a bytes.NewBuffer()
	obuf, err := ioutil.ReadAll(old)
	if err != nil {
		return err
	}

	// TODO: write directly to new instead of using a buffer
	nbuf := make([]byte, newSize)

	var oldpos, newpos int64

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
		for i := int64(0); i < int64(len(ctrl.Add)); i++ {
			nbuf[newpos+i] = ctrl.Add[i] + obuf[oldpos+i]
		}

		// Adjust pointers
		newpos += int64(len(ctrl.Add))
		oldpos += int64(len(ctrl.Add))

		// Sanity-check
		if newpos+int64(len(ctrl.Copy)) > newSize {
			return ErrCorrupt
		}

		// Read extra string
		copy(nbuf[newpos:], ctrl.Copy)

		// Adjust pointers
		newpos += int64(len(ctrl.Copy))
		oldpos += ctrl.Seek
	}

	if newpos != newSize {
		return fmt.Errorf("bsdiff: expected new file to be %d, was %d (%s difference)", newSize, newpos, humanize.IBytes(uint64(newSize-newpos)))
	}

	// Write the new file
	_, err = new.Write(nbuf)
	if err != nil {
		return err
	}

	return nil
}
