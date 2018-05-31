package launch

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/ox"
	"github.com/itchio/pelican"
	"github.com/itchio/wharf/tlc"
	"github.com/pkg/errors"

	"github.com/itchio/dash"
)

type LaunchStrategy string

const (
	LaunchStrategyUnknown LaunchStrategy = ""
	LaunchStrategyNative  LaunchStrategy = "native"
	LaunchStrategyHTML    LaunchStrategy = "html"
	LaunchStrategyURL     LaunchStrategy = "url"
	LaunchStrategyShell   LaunchStrategy = "shell"
)

type LauncherParams struct {
	RequestContext *butlerd.RequestContext
	Ctx            context.Context

	// If relative, it's relative to the WorkingDirectory
	FullTargetPath string

	// May be nil
	PeInfo *pelican.PeInfo

	// May be nil
	Candidate *dash.Candidate

	// Lazily computed
	installContainer *tlc.Container

	// May be nil
	AppManifest *butlerd.Manifest

	// May be nil
	Action *butlerd.Action

	// If true, enable sandbox
	Sandbox bool

	// Additional command-line arguments
	Args []string

	// Additional environment variables
	Env map[string]string

	PrereqsDir    string
	ForcePrereqs  bool
	Access        *operate.GameAccess
	InstallFolder string
	Runtime       *ox.Runtime

	RecordPlayTime RecordPlayTimeFunc
}

// cf. https://github.com/itchio/itch/issues/1751
var ignoredInstallContainerPatterns = []string{
	"node_modules",
}

func (lp *LauncherParams) GetInstallContainer() (*tlc.Container, error) {
	if lp.installContainer == nil {
		var err error
		lp.installContainer, err = tlc.WalkDir(lp.InstallFolder, &tlc.WalkOpts{
			Filter: func(fileInfo os.FileInfo) bool {
				if !filtering.FilterPaths(fileInfo) {
					return false
				}

				for _, pattern := range ignoredInstallContainerPatterns {
					match, _ := filepath.Match(pattern, fileInfo.Name())
					if match {
						return false
					}
				}

				return true
			},
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}
	return lp.installContainer, nil
}

func (lp *LauncherParams) SniffFile(fileEntry *tlc.File) (*dash.Candidate, error) {
	f, err := os.Open(filepath.Join(lp.InstallFolder, fileEntry.Path))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer f.Close()

	stats, err := f.Stat()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	candidate, err := dash.Sniff(f, fileEntry.Path, stats.Size())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return candidate, nil
}

type RecordPlayTimeFunc func(playTime time.Duration) error

type Launcher interface {
	Do(params *LauncherParams) error
}

var launchers = make(map[LaunchStrategy]Launcher)

func RegisterLauncher(strategy LaunchStrategy, launcher Launcher) {
	launchers[strategy] = launcher
}
