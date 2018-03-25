package shell

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/endpoints/launch"
)

func Register() {
	launch.RegisterLauncher(launch.LaunchStrategyShell, &Launcher{})
}

type Launcher struct{}

var _ launch.Launcher = (*Launcher)(nil)

func (l *Launcher) Do(params *launch.LauncherParams) error {
	_, err := messages.ShellLaunch.Call(params.RequestContext, &butlerd.ShellLaunchParams{
		ItemPath: params.FullTargetPath,
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
