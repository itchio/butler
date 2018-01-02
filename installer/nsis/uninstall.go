package nsis

import (
	"errors"

	"github.com/itchio/butler/installer"
)

func (m *Manager) Uninstall(params *installer.UninstallParams) error {
	return errors.New("nsis/uninstall: stub")
}
