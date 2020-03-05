package launch

import (
	"context"
	"os"
	"path/filepath"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/butler/manager"
	"github.com/itchio/hush/manifest"
	"github.com/itchio/lake/tlc"
	"github.com/itchio/pelican"
	"github.com/pkg/errors"

	"github.com/itchio/dash"
)

type LauncherParams struct {
	RequestContext *butlerd.RequestContext
	Ctx            context.Context

	WorkingDirectory string

	// If relative, it's relative to the WorkingDirectory
	FullTargetPath string

	// May be nil
	PeInfo *pelican.PeInfo

	// May be nil
	Candidate *dash.Candidate

	// Lazily computed
	installContainer *tlc.Container

	// May be nil
	AppManifest *manifest.Manifest

	// May be nil
	Action *manifest.Action

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
	Host          manager.Host

	SessionStarted func()
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

type Launcher interface {
	Do(params LauncherParams) error
}

var launchers = make(map[butlerd.LaunchStrategy]Launcher)

func RegisterLauncher(strategy butlerd.LaunchStrategy, launcher Launcher) {
	launchers[strategy] = launcher
}
