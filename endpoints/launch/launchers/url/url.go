package url

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/endpoints/launch"
	"github.com/pkg/errors"
)

func Register() {
	launch.RegisterLauncher(butlerd.LaunchStrategyURL, &Launcher{})
}

type Launcher struct{}

var _ launch.Launcher = (*Launcher)(nil)

func (l *Launcher) Do(params launch.LauncherParams) error {
	_, err := messages.URLLaunch.Call(params.RequestContext, butlerd.URLLaunchParams{
		URL: params.FullTargetPath,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
