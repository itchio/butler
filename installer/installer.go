package installer

import (
	"context"
	"errors"
	"os"

	"github.com/itchio/butler/archive"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

var ErrNeedLocal = errors.New("install source needs to be available locally")

type Manager interface {
	Install(params *InstallParams) (*InstallResult, error)
	Uninstall(params *UninstallParams) error
	Name() string
}

type InstallParams struct {
	// An archive file, .exe setup file, .dmg file etc.
	File eos.File

	// The existing receipt, if any
	ReceiptIn *bfs.Receipt

	// A folder we can use to store temp files
	StageFolderPath string

	// The folder we're installing to
	InstallFolderPath string

	// Listener for progress events, logging etc.
	Consumer *state.Consumer

	InstallerInfo *InstallerInfo

	// For cancellation
	Context context.Context

	// true if first-time install
	Fresh bool
}

type UninstallParams struct {
	// The folder we're uninstalling from
	InstallFolderPath string

	// Listener for progress events, logging etc.
	Consumer *state.Consumer
}

type InstallResult struct {
	// Files is a list of paths, relative to the install folder
	Files []string

	// optional, installer-specific fields:
	MSIProductCode string
}

type InstallerInfo struct {
	Type        InstallerType
	ArchiveInfo *archive.ArchiveInfo
}

type InstallerType string

const (
	InstallerTypeNaked       InstallerType = "naked"
	InstallerTypeArchive     InstallerType = "archive"
	InstallerTypeDMG         InstallerType = "dmg"
	InstallerTypeInno        InstallerType = "inno"
	InstallerTypeNsis        InstallerType = "nsis"
	InstallerTypeMSI         InstallerType = "msi"
	InstallerTypeUnknown     InstallerType = "unknown"
	InstallerTypeUnsupported InstallerType = "unsupported"
)

// AsLocalFile takes an eos.File and tries to cast it to an *os.File.
// If that fails, it returns `ErrNeedLocal`. Consumers of functions that
// call + relay AsLocalFile's errors are expected to know how to
// download a file to disk and call again with an *os.File instance instead
// of, say, an httpfile.HTTPFile
func AsLocalFile(f eos.File) (*os.File, error) {
	if lf, ok := f.(*os.File); ok {
		return lf, nil
	}

	return nil, ErrNeedLocal
}
