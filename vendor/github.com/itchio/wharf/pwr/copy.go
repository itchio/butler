package pwr

import (
	"io"

	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

// CopyContainer copies from one container to the other. Combined with fspool
// and blockpool, it can be used to split a container into blocks or join it back
// into regular files.
func CopyContainer(container *tlc.Container, outPool wsync.WritablePool, inPool wsync.Pool, consumer *state.Consumer) error {
	copyFile := func(byteOffset int64, fileIndex int64) error {
		r, err := inPool.GetReader(fileIndex)
		if err != nil {
			return err
		}

		w, err := outPool.GetWriter(fileIndex)
		if err != nil {
			return err
		}

		cw := counter.NewWriterCallback(func(count int64) {
			alpha := float64(byteOffset+count) / float64(container.Size)
			consumer.Progress(alpha)
		}, w)

		_, err = io.Copy(cw, r)
		if err != nil {
			return err
		}

		err = w.Close()
		if err != nil {
			return err
		}

		return nil
	}

	byteOffset := int64(0)

	for fileIndex, f := range container.Files {
		consumer.ProgressLabel(f.Path)

		err := copyFile(byteOffset, int64(fileIndex))
		if err != nil {
			return err
		}

		byteOffset += f.Size
	}

	return nil
}
