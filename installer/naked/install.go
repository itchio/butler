package naked

import (
	"path/filepath"

	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
	"github.com/pkg/errors"
)

func (m *Manager) Install(params installer.InstallParams) (*installer.InstallResult, error) {
	consumer := params.Consumer

	stats, err := params.File.Stat()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	destName := filepath.Base(stats.Name())
	destAbsolutePath := filepath.Join(params.InstallFolderPath, destName)

	err = operate.DownloadInstallSource(operate.DownloadInstallSourceParams{
		Context:       params.Context,
		Consumer:      params.Consumer,
		StageFolder:   params.StageFolderPath,
		OperationName: "naked-installer",
		File:          params.File,
		DestPath:      destAbsolutePath,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var res = installer.InstallResult{
		Files: []string{
			destName,
		},
	}

	consumer.Opf("Busting ghosts...")
	var bustGhostStats bfs.BustGhostStats
	err = bfs.BustGhosts(bfs.BustGhostsParams{
		Folder:   params.InstallFolderPath,
		NewFiles: res.Files,
		Receipt:  params.ReceiptIn,
		Consumer: params.Consumer,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = params.EventSink.PostGhostBusting("install::naked", bustGhostStats)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
