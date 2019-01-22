package iexpress

import (
	"github.com/pkg/errors"

	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/archive"
)

type Manager struct {
}

var _ installer.Manager = (*Manager)(nil)

func (m *Manager) Name() string {
	return "iexpress"
}

func Register() {
	installer.RegisterManager(&Manager{})
}

var archiveManager = &archive.Manager{}

func (m *Manager) Install(params installer.InstallParams) (*installer.InstallResult, error) {
	// force the file to be available locally.
	// opening an iexpress file in 7-zip will extract correctly eventually
	// but needs to read the file several times in its entirety for some reason.
	f := params.File
	localFile, err := installer.AsLocalFile(f)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	consumer := params.Consumer
	consumer.Infof("IExpress installer reporting for duty")

	params.File = localFile

	// defer to archive installer, since that's what we do
	return archiveManager.Install(params)
}

func (m *Manager) Uninstall(params installer.UninstallParams) error {
	return archiveManager.Uninstall(params)
}
