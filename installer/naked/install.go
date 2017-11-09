package naked

import (
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/ditto"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
)

func (m *Manager) Install(params *installer.InstallParams) (*installer.InstallResult, error) {
	destName := filepath.Base(params.SourcePath)
	destAbsolutePath := filepath.Join(params.InstallFolderPath, destName)

	params.Consumer.Infof("Creating %s", params.InstallFolderPath)
	err := os.MkdirAll(params.InstallFolderPath, 0755)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	params.Consumer.Infof("Writing %s", destAbsolutePath)

	err = ditto.Do(params.SourcePath, destAbsolutePath)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var res = installer.InstallResult{
		Files: []string{
			destName,
		},
	}

	err = bfs.BustGhosts(&bfs.BustGhostsParams{
		Consumer: params.Consumer,
		Folder:   params.InstallFolderPath,
		NewFiles: res.Files,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &res, nil
}
