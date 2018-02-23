package url

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/endpoints/launch"
)

func Register() {
	launch.RegisterLauncher(launch.LaunchStrategyURL, &Launcher{})
}

type Launcher struct{}

var _ launch.Launcher = (*Launcher)(nil)

func (l *Launcher) Do(params *launch.LauncherParams) error {
	_, err := messages.URLLaunch.Call(params.RequestContext, &buse.URLLaunchParams{
		URL: params.FullTargetPath,
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
