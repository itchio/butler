package html

import (
	"path/filepath"

	"github.com/itchio/butler/buse/messages"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/endpoints/launch"
)

func Register() {
	launch.RegisterLauncher(launch.LaunchStrategyHTML, &Launcher{})
}

type Launcher struct{}

var _ launch.Launcher = (*Launcher)(nil)

func (l *Launcher) Do(params *launch.LauncherParams) error {
	rootFolder := params.InstallFolder
	indexPath, err := filepath.Rel(rootFolder, params.FullTargetPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	_, err = messages.HTMLLaunch.Call(params.RequestContext, &buse.HTMLLaunchParams{
		RootFolder: rootFolder,
		IndexPath:  indexPath,
		Args:       params.Args,
		Env:        params.Env,
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
