package launch

import (
	"context"

	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/manager"

	"github.com/itchio/butler/cmd/launch/manifest"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/configurator"
	"github.com/itchio/wharf/state"
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
	Ctx      context.Context
	Conn     operate.Conn
	Consumer *state.Consumer

	// If relative, it's relative to the WorkingDirectory
	FullTargetPath string

	// May be nil
	Candidate *configurator.Candidate

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
	Credentials   *buse.GameCredentials
	InstallFolder string
	Runtime       *manager.Runtime
}

type Launcher interface {
	Do(params *LauncherParams) error
}

var launchers = make(map[LaunchStrategy]Launcher)

func Register(strategy LaunchStrategy, launcher Launcher) {
	launchers[strategy] = launcher
}
