package archive

import (
	"os"

	"github.com/go-errors/errors"
)

// List returns information on a given archive
// it cannot be an `eos` path because unarchiver doesn't
// support those.
func List(params *ListParams) (ListResult, error) {
	// this gets size & ensures the file exists locally
	_, err := os.Lstat(params.Path)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	for _, handler := range handlers {
		res, err := handler.List(params)
		if err != nil {
			params.Consumer.Infof("Handler %s couldn't list: %s", handler.Name(), err.Error())
			continue
		}

		return res, nil
	}

	return nil, ErrUnrecognizedArchiveType
}
