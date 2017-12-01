package szah

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
	"github.com/itchio/sevenzip-go/sz"
)

func (h *Handler) TryOpen(params *archive.TryOpenParams) error {
	stats, err := params.File.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = withArchive(params.Consumer, params.File, func(a *sz.Archive) error {
		// do nothing - we're happy if we managed to open it
		return nil
	})

	if err != nil {
		params.Consumer.Infof("szah can't open %s: %s", stats.Name(), err.Error())
		return archive.ErrUnrecognizedArchiveType
	}
	return nil
}
