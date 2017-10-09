package archive

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
	"github.com/itchio/butler/archive/uniarch"
	"github.com/itchio/butler/cmd/cave/imag"
)

func (m *Manager) Install(params *imag.InstallParams) (*imag.InstallResult, error) {
	listResult := params.ArchiveListResult
	if listResult == nil {
		var err error
		listResult, err = uniarch.List(&archive.ListParams{
			Path: params.SourcePath,
		})

		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	handler := listResult.Handler()

	var res = imag.InstallResult{}
	err := imag.SaveAngels(params, func() ([]string, error) {
		err := handler.Extract(&archive.ExtractParams{
			Path:       params.SourcePath,
			OutputPath: params.InstallFolderPath,
			ListResult: listResult,
		})
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		var paths []string
		for _, entry := range listResult.Entries() {
			paths = append(paths, entry)
		}
		return paths, nil
	})

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &imag, nil
}
