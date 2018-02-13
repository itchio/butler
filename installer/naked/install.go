package naked

import (
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
)

func (m *Manager) Install(params *installer.InstallParams) (*installer.InstallResult, error) {
	stats, err := params.File.Stat()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	destName := filepath.Base(stats.Name())
	destAbsolutePath := filepath.Join(params.InstallFolderPath, destName)

	err = operate.DownloadInstallSource(params.Consumer, params.StageFolderPath, params.Context, params.File, destAbsolutePath)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var res = installer.InstallResult{
		Files: []string{
			destName,
		},
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
