package bah

import (
	"github.com/go-errors/errors"
	"github.com/itchio/arkive/zip"
	"github.com/itchio/butler/archive"
)

func (h *Handler) TryOpen(params *archive.TryOpenParams) error {
	stats, err := params.File.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	_, err = zip.NewReader(params.File, stats.Size())
	if err != nil {
		params.Consumer.Infof("bah can't open %s: %s", stats.Name(), err.Error())
		return archive.ErrUnrecognizedArchiveType
	}
	return nil
}
