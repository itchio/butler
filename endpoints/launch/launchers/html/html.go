package html

import (
	"path/filepath"
	"time"

	"github.com/itchio/butler/butlerd/messages"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/butlerd"
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

	startTime := time.Now()

	messages.LaunchRunning.Notify(params.RequestContext, &butlerd.LaunchRunningNotification{})
	_, err = messages.HTMLLaunch.Call(params.RequestContext, &butlerd.HTMLLaunchParams{
		RootFolder: rootFolder,
		IndexPath:  indexPath,
		Args:       params.Args,
		Env:        params.Env,
	})
	messages.LaunchExited.Notify(params.RequestContext, &butlerd.LaunchExitedNotification{})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	runDuration := time.Since(startTime)
	err = params.RecordPlayTime(runDuration)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
