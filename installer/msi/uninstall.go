package msi

import (
	"errors"

	"github.com/itchio/butler/installer"
)

func (m *Manager) Uninstall(params *installer.UninstallParams) error {
	return errors.New("msi/uninstall: stub")
}
