package archive

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
)

func (m *Manager) Install(params *installer.InstallParams) (*installer.InstallResult, error) {
	var res = installer.InstallResult{
		Files: []string{},
	}

	aRes, err := archive.Extract(&archive.ExtractParams{
		File:       params.File,
		OutputPath: params.InstallFolderPath,
		Consumer:   params.Consumer,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	for _, entry := range aRes.Entries {
		res.Files = append(res.Files, entry.Name)
	}

	err = bfs.BustGhosts(&bfs.BustGhostsParams{
		Folder:   params.InstallFolderPath,
		NewFiles: res.Files,
		Receipt:  params.ReceiptIn,

		Consumer: params.Consumer,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &res, nil
}
