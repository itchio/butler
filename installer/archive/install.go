package archive

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
)

func (m *Manager) Install(params *installer.InstallParams) (*installer.InstallResult, error) {
	listResult := params.ArchiveListResult
	if listResult == nil {
		var err error
		listResult, err = archive.List(&archive.ListParams{
			Path:     params.SourcePath,
			Consumer: params.Consumer,
		})

		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	handler := listResult.Handler()

	var res = installer.InstallResult{
		Files: []string{},
	}

	err := handler.Extract(&archive.ExtractParams{
		Consumer:   params.Consumer,
		Path:       params.SourcePath,
		OutputPath: params.InstallFolderPath,
		ListResult: listResult,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	for _, entry := range listResult.Entries() {
		res.Files = append(res.Files, entry.Name)
	}

	err = bfs.BustGhosts(&bfs.BustGhostsParams{
		Consumer: params.Consumer,
		Folder:   params.InstallFolderPath,
		NewFiles: res.Files,
		Receipt:  params.ReceiptIn,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &res, nil
}
