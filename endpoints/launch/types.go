package launch

import (
	"context"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/manager"
	"github.com/itchio/pelican"
	"github.com/pkg/errors"

	"github.com/itchio/butler/configurator"
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
	Candidate *configurator.Candidate

	// Lazily computed
	unfilteredVerdict *configurator.Verdict

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
	Credentials   *butlerd.GameCredentials
	InstallFolder string
	Runtime       *manager.Runtime

	RecordPlayTime RecordPlayTimeFunc
}

func (lp *LauncherParams) GetUnfilteredVerdict() (*configurator.Verdict, error) {
	if lp.unfilteredVerdict == nil {
		var err error
		lp.unfilteredVerdict, err = configurator.Configure(lp.InstallFolder, false)
		if err != nil {
			return nil, errors.WithMessage(err, "while getting unfiltered verdict")
		}
	}
	return lp.unfilteredVerdict, nil
}

type RecordPlayTimeFunc func(playTime time.Duration) error

type Launcher interface {
	Do(params *LauncherParams) error
}

var launchers = make(map[LaunchStrategy]Launcher)

func RegisterLauncher(strategy LaunchStrategy, launcher Launcher) {
	launchers[strategy] = launcher
}
