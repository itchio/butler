package archive

import (
	"path/filepath"

	"github.com/itchio/savior"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/archive/intervalsaveconsumer"
	"github.com/itchio/butler/installer/bfs"
)

func (m *Manager) Install(params *installer.InstallParams) (*installer.InstallResult, error) {
	consumer := params.Consumer

	var res = installer.InstallResult{
		Files: []string{},
	}

	archiveInfo := params.InstallerInfo.ArchiveInfo

	ex, err := archiveInfo.GetExtractor(params.File, consumer)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	ex.SetConsumer(consumer)

	statePath := filepath.Join(params.StageFolderPath, "install-state.dat")
	sc := intervalsaveconsumer.New(statePath, intervalsaveconsumer.DefaultInterval, consumer, params.Context)
	ex.SetSaveConsumer(sc)

	checkpoint := &savior.ExtractorCheckpoint{}
	err = sc.Load(checkpoint)
	if err != nil {
		consumer.Warnf("could not load checkpoint, ignoring: %s", err.Error())
		checkpoint = nil
	}

	sink := &savior.FolderSink{
		Directory: params.InstallFolderPath,
		Consumer:  consumer,
	}

	aRes, err := ex.Resume(checkpoint, sink)
	if err != nil {
		if errors.Is(err, savior.ErrStop) {
			return nil, operate.ErrCancelled
		}
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
