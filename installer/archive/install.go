package archive

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/itchio/boar"
	"github.com/itchio/savior"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/archive/intervalsaveconsumer"
	"github.com/itchio/butler/installer/bfs"
	"github.com/pkg/errors"
)

func (m *Manager) Install(params installer.InstallParams) (*installer.InstallResult, error) {
	consumer := params.Consumer

	var res = installer.InstallResult{
		Files: []string{},
	}

	f := params.File

	archiveInfo := params.InstallerInfo.ArchiveInfo

	consumer.Infof("Archive installer ready for action")
	stat, _ := f.Stat()
	consumer.Infof("Archive name is (%s)", stat.Name())

	if archiveInfo == nil {
		consumer.Infof("Missing archive info, probing...")
		var err error
		archiveInfo, err = boar.Probe(&boar.ProbeParams{
			File:     f,
			Consumer: consumer,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	if archiveInfo.Features.ResumeSupport == savior.ResumeSupportNone {
		consumer.Infof("Forcing local for %s", archiveInfo.Features)
		localFile, err := installer.AsLocalFile(f)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		f = localFile
	}

	ex, err := archiveInfo.GetExtractor(f, consumer)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ex.SetConsumer(consumer)

	statePath := filepath.Join(params.StageFolderPath, "install-state.dat")
	sc := intervalsaveconsumer.New(statePath, intervalsaveconsumer.DefaultInterval, consumer, params.Context)
	ex.SetSaveConsumer(sc)

	cancelled := false
	defer func() {
		if !cancelled {
			consumer.Infof("Clearing archive install state")
			os.Remove(statePath)
		}
	}()

	checkpoint, err := sc.Load()
	if err != nil {
		consumer.Warnf("Could not load checkpoint: %s", err.Error())
	}

	sink := &savior.FolderSink{
		Directory: params.InstallFolderPath,
		Consumer:  consumer,
	}
	var closeSinkOnce sync.Once
	defer closeSinkOnce.Do(func() {
		sink.Close()
	})

	aRes, err := ex.Resume(checkpoint, sink)
	if err != nil {
		if errors.Cause(err) == savior.ErrStop {
			cancelled = true
			return nil, errors.WithStack(butlerd.CodeOperationCancelled)
		}
		return nil, errors.WithStack(err)
	}

	closeSinkOnce.Do(func() {
		err = sink.Close()
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	consumer.Opf("Applying stage-two...")
	aRes, err = archiveInfo.ApplyStageTwo(consumer, aRes, params.InstallFolderPath)
	if err != nil {
		if errors.Cause(err) == savior.ErrStop {
			cancelled = true
			return nil, errors.WithStack(butlerd.CodeOperationCancelled)
		}
		return nil, errors.WithStack(err)
	}

	for _, entry := range aRes.Entries {
		res.Files = append(res.Files, entry.CanonicalPath)
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
	err = params.EventSink.PostGhostBusting("install::archive", bustGhostStats)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
