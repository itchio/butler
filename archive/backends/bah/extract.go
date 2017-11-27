package bah

import (
	"github.com/go-errors/errors"
	"github.com/itchio/wharf/archiver"

	"github.com/itchio/butler/archive"
)

func (h *Handler) Extract(params *archive.ExtractParams) (*archive.Contents, error) {
	fi, err := params.File.Stat()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var entries []*archive.Entry
	_, err = archiver.ExtractZip(params.File, fi.Size(), params.OutputPath, archiver.ExtractSettings{
		Consumer:    params.Consumer,
		Concurrency: 1, // force 1 worker because we're probably extracting a remote file
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &archive.Contents{
		Entries: entries,
	}
	return res, nil
}
