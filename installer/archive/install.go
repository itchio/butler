package archive

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dchest/safefile"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
)

func (m *Manager) Install(params *installer.InstallParams) (*installer.InstallResult, error) {
	var res = installer.InstallResult{
		Files: []string{},
	}

	handler := archive.GetHandler(params.InstallerInfo.ArchiveHandlerName)
	if handler == nil {
		return nil, errors.Wrap(fmt.Errorf("could not find archive handler: %s", params.InstallerInfo.ArchiveHandlerName), 0)
	}

	statePath := filepath.Join(params.StageFolderPath, "install-state.dat")

	aRes, err := handler.Extract(&archive.ExtractParams{
		File:       params.File,
		OutputPath: params.InstallFolderPath,
		Consumer:   params.Consumer,
		Load: func(state interface{}) error {
			stateFile, err := os.Open(statePath)
			if err != nil {
				if os.IsNotExist(err) {
					// that's ok
					return nil
				}
				return errors.Wrap(err, 0)
			}
			defer stateFile.Close()

			dec := gob.NewDecoder(stateFile)
			return dec.Decode(state)
		},
		Save: func(state interface{}) error {
			stateFile, err := safefile.Create(statePath, 0644)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			defer stateFile.Close()

			enc := gob.NewEncoder(stateFile)
			err = enc.Encode(state)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			err = stateFile.Commit()
			if err != nil {
				return errors.Wrap(err, 0)
			}

			return nil
		},
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
