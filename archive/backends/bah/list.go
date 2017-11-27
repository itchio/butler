package bah

import (
	"github.com/go-errors/errors"
	"github.com/itchio/arkive/zip"
	"github.com/itchio/butler/archive"
)

func (h *Handler) List(params *archive.ListParams) (*archive.Contents, error) {
	fi, err := params.File.Stat()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	zr, err := zip.NewReader(params.File, fi.Size())
	if err != nil {
		return nil, archive.ErrUnrecognizedArchiveType
	}

	var entries []*archive.Entry
	for _, f := range zr.File {
		entries = append(entries, &archive.Entry{
			Name:             archive.CleanFileName(f.Name),
			UncompressedSize: int64(f.UncompressedSize64),
		})
	}

	res := &archive.Contents{
		Entries: entries,
	}
	return res, nil
}
