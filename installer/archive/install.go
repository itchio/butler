package archive

import (
	"path/filepath"
	"time"

	"github.com/itchio/savior"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
)

var defaultSaveInterval = 1 * time.Second

func (m *Manager) Install(params *installer.InstallParams) (*installer.InstallResult, error) {
	var res = installer.InstallResult{
		Files: []string{},
	}

	archiveInfo := params.InstallerInfo.ArchiveInfo

	ex, err := archiveInfo.GetExtractor(params.File, params.Consumer)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	ex.SetConsumer(params.Consumer)

	statePath := filepath.Join(params.StageFolderPath, "install-state.dat")
	sc := newSaveConsumer(statePath, defaultSaveInterval, params.Consumer, params.Context)
	ex.SetSaveConsumer(sc)

	var checkpoint *savior.ExtractorCheckpoint
	err = sc.Load(checkpoint)
	if err != nil {
		params.Consumer.Warnf("could not load checkpoint, ignoring: %s", err.Error())
	}

	sink := &savior.FolderSink{
		Directory: params.InstallFolderPath,
	}

	aRes, err := ex.Resume(checkpoint, sink)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	for _, entry := range aRes.Entries {
		res.Files = append(res.Files, entry.CanonicalPath)
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
