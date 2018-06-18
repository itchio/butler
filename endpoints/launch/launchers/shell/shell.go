package shell

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/endpoints/launch"
	"github.com/pkg/errors"
)

func Register() {
	launch.RegisterLauncher(launch.LaunchStrategyShell, &Launcher{})
}

type Launcher struct{}

var _ launch.Launcher = (*Launcher)(nil)

func (l *Launcher) Do(params launch.LauncherParams) error {
	_, err := messages.ShellLaunch.Call(params.RequestContext, butlerd.ShellLaunchParams{
		ItemPath: params.FullTargetPath,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
