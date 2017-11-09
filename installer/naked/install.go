package naked

import (
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/ditto"
	"github.com/itchio/butler/installer"
)

func (m *Manager) Install(params *installer.InstallParams) (*installer.InstallResult, error) {
	destName := filepath.Base(params.SourcePath)
	destAbsolutePath := filepath.Join(params.InstallFolderPath, destName)

	err := ditto.Do(params.SourcePath, destAbsolutePath)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var res = installer.InstallResult{
		Files: []string{
			destName,
		},
	}
	return &res, nil
}
