package auditzip

import (
	"fmt"
	"io"
	"io/ioutil"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/arkive/zip"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

var args = struct {
	file *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("auditzip", "Audit a zip file for common errors")
	args.file = cmd.Arg("file", ".zip file to audit").Required().String()
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

	zr, err := zip.NewReader(f, stats.Size())
	if err != nil {
		return errors.Wrap(err, 0)
	}

	numEntries := len(zr.File)
	paths := make(map[string]int)
	checkEntry := func(index int, e *zip.File) error {
		path := archive.CleanFileName(e.Name)

		comm.Progress(float64(index) / float64(numEntries))
		comm.ProgressLabel(path)

		if previousIndex, ok := paths[path]; ok {
			consumer.Warnf("Duplicate path (%s) at indices (%d) and (%d)", path, index, previousIndex)
		}
		paths[path] = index

		rc, err := e.Open()
		if err != nil {
			return errors.Wrap(err, 0)
		}
		defer rc.Close()

		actualSize, err := io.Copy(ioutil.Discard, rc)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if actualSize != int64(e.UncompressedSize64) {
			err := fmt.Errorf("Dictionary says (%s) is %s (%d bytes), but it's actually %s (%d bytes)",
				path,
				humanize.IBytes(e.UncompressedSize64),
				e.UncompressedSize64,
				humanize.IBytes(uint64(actualSize)),
				actualSize,
			)
			return errors.Wrap(err, 0)
		}
		return nil
	}

	comm.StartProgress()
	for index, e := range zr.File {
		err = checkEntry(index, e)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}
	comm.EndProgress()

	consumer.Statf("Everything checks out!")

	return nil
}
