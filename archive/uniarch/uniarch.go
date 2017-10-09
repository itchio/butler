package uniarch

// UNIversal ARCHive handler

import (
	"os"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
	"github.com/itchio/butler/archive/backends/bah"
	"github.com/itchio/butler/archive/backends/xad"
)

var handlers = []archive.Handler{
	bah.NewHandler(),
	xad.NewHandler(),
}

// List returns information on a given archive
// it cannot be an `eos` path because unarchiver doesn't
// support those.
func List(params *archive.ListParams) (archive.ListResult, error) {
	// this gets size & ensures the file exists locally
	_, err := os.Lstat(params.Path)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	for _, handler := range handlers {
		res, err := handler.List(params)
		if err == nil {
			return res, nil
		}
	}

	return nil, archive.ErrUnrecognizedArchiveType
}
