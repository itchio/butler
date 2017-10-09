package archive

import (
	"github.com/itchio/butler/cmd/cave/imag"
)

type Manager struct {
}

var _ imag.Manager = (*Manager)(nil)

func (m *Manager) Name() string {
	return "archive"
}
