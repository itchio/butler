package installer

import (
	"context"
	"errors"
	"os"

	"github.com/itchio/boar"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/headway/state"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/savior"
)

var ErrNeedLocal = errors.New("install source needs to be available locally")

type Manager interface {
	Install(params InstallParams) (*InstallResult, error)
	Uninstall(params UninstallParams) error
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

	EventSink *InstallEventSink
}

type UninstallParams struct {
	// The folder we're uninstalling from
	InstallFolderPath string

	// Listener for progress events, logging etc.
	Consumer *state.Consumer

	// Receipt at the time we asked for an uninstall
	Receipt *bfs.Receipt
}

type InstallResult struct {
	// Files is a list of paths, relative to the install folder
	Files []string

	// optional, installer-specific fields:
	MSIProductCode string
}

type InstallerInfo struct {
	Type        InstallerType
	ArchiveInfo *boar.Info
	Entries     []*savior.Entry
}

type InstallerType string

const (
	InstallerTypeNaked   InstallerType = "naked"
	InstallerTypeArchive InstallerType = "archive"
	InstallerTypeUnknown InstallerType = "unknown"
)

// AsLocalFile takes an eos.File and tries to cast it to an *os.File.
// If that fails, it returns `ErrNeedLocal`. Consumers of functions that
// call + relay AsLocalFile's errors are expected to know how to
// download a file to disk and call again with an *os.File instance instead
// of, say, an *htfs.File
func AsLocalFile(f eos.File) (*os.File, error) {
	if lf, ok := f.(*os.File); ok {
		return lf, nil
	}

	return nil, ErrNeedLocal
}
