package auditzip

import (
	"fmt"
	"io"
	"io/ioutil"

	upstreamzip "archive/zip"

	itchiozip "github.com/itchio/arkive/zip"

	humanize "github.com/dustin/go-humanize"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

var args = struct {
	file     *string
	upstream *bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("auditzip", "Audit a zip file for common errors")
	args.file = cmd.Arg("file", ".zip file to audit").Required().String()
	args.upstream = cmd.Flag("upstream", "Use upstream zip implementation (archive/zip)").Bool()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	consumer := comm.NewStateConsumer()
	ctx.Must(Do(consumer, *args.file))
}

func Do(consumer *state.Consumer, file string) error {
	f, err := eos.Open(file)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer f.Close()

	stats, err := f.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Opf("Auditing (%s)...", stats.Name())

	var impl ZipImpl
	if *args.upstream {
		consumer.Opf("Using upstream zip implementation")
		impl = &upstreamImpl{}
	} else {
		consumer.Opf("Using itchio/arkive zip implementation")
		impl = &itchioImpl{}
	}

	paths := make(map[string]int)
	comm.StartProgress()
	err = impl.EachEntry(f, stats.Size(), func(index int, name string, uncompressedSize int64, rc io.ReadCloser, numEntries int) error {
		path := archive.CleanFileName(name)

		comm.Progress(float64(index) / float64(numEntries))
		comm.ProgressLabel(path)

		if previousIndex, ok := paths[path]; ok {
			consumer.Warnf("Duplicate path (%s) at indices (%d) and (%d)", path, index, previousIndex)
		}
		paths[path] = index

		actualSize, err := io.Copy(ioutil.Discard, rc)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if actualSize != uncompressedSize {
			err := fmt.Errorf("Dictionary says (%s) is %s (%d bytes), but it's actually %s (%d bytes)",
				path,
				humanize.IBytes(uint64(uncompressedSize)),
				uncompressedSize,
				humanize.IBytes(uint64(actualSize)),
				actualSize,
			)
			return errors.Wrap(err, 0)
		}
		return nil
	})
	comm.EndProgress()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Statf("Everything checks out!")

	return nil
}

// zip implementation types

type EachEntryFunc func(index int, name string, uncompressedSize int64, rc io.ReadCloser, numEntries int) error

type ZipImpl interface {
	EachEntry(r io.ReaderAt, size int64, cb EachEntryFunc) error
}

// itchio zip impl

type itchioImpl struct{}

var _ ZipImpl = (*itchioImpl)(nil)

func (a *itchioImpl) EachEntry(r io.ReaderAt, size int64, cb EachEntryFunc) error {
	zr, err := itchiozip.NewReader(r, size)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	numEntries := len(zr.File)
	for index, entry := range zr.File {
		rc, err := entry.Open()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		err = cb(index, entry.Name, int64(entry.UncompressedSize64), rc, numEntries)
		rc.Close()
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return nil
}

// upstream zip impl

type upstreamImpl struct{}

var _ ZipImpl = (*upstreamImpl)(nil)

func (a *upstreamImpl) EachEntry(r io.ReaderAt, size int64, cb EachEntryFunc) error {
	zr, err := upstreamzip.NewReader(r, size)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	numEntries := len(zr.File)
	for index, entry := range zr.File {
		rc, err := entry.Open()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		err = cb(index, entry.Name, int64(entry.UncompressedSize64), rc, numEntries)
		rc.Close()
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return nil
}
