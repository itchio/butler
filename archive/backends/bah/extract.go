package bah

import (
	"os"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/archiver"

	"github.com/itchio/butler/archive"
)

func (h *Handler) Extract(params *archive.ExtractParams) error {
	f, err := os.Open(params.Path)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	_, err = archiver.ExtractZip(f, fi.Size(), params.OutputPath, archiver.ExtractSettings{
		Consumer:    params.Consumer,
		Concurrency: -1,
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
