package url

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/endpoints/launch"
)

func Register() {
	launch.RegisterLauncher(launch.LaunchStrategyURL, &Launcher{})
}

type Launcher struct{}

var _ launch.Launcher = (*Launcher)(nil)

func (l *Launcher) Do(params *launch.LauncherParams) error {
	_, err := messages.URLLaunch.Call(params.RequestContext, &butlerd.URLLaunchParams{
		URL: params.FullTargetPath,
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
