package imag

import (
	"github.com/itchio/butler/archive"
	"github.com/itchio/wharf/state"
)

// imag = Install Manager

type Manager interface {
	Install(params *InstallParams) (*InstallResult, error)
	Uninstall(params *UninstallParams) error
	Name() string
}

type InstallParams struct {
	// An archive file, .exe setup file, .dmg file etc.
	SourcePath string

	// Where the item should be installed
	InstallFolderPath string

	// A temporary folder we can do whatever with
	StagePath string

	Consumer *state.Consumer

	InstallerType     string
	ArchiveListResult archive.ListResult
}

type UninstallParams struct {
	InstallPath string
}

type InstallResult struct {
	// Files is a list of paths, relative to the install folder
	Files []string
}
