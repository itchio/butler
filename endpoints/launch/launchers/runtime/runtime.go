// Package runtime launches payloads the client runs with a runtime of its
// own. butler does the bookkeeping of a native launch, the session and the
// cave's play time, around a request the client answers when the game has
// exited. Like the html launcher, it never sees the process.
package runtime

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/endpoints/launch"
	"github.com/pkg/errors"
)

func Register() {
	launch.RegisterLauncher(butlerd.LaunchStrategyRuntime, &Launcher{})
}

type Launcher struct{}

var _ launch.Launcher = (*Launcher)(nil)

func (l *Launcher) Do(params launch.LauncherParams) error {
	messages.LaunchRunning.Notify(params.RequestContext, butlerd.LaunchRunningNotification{})
	params.SessionStarted()

	_, err := messages.RuntimeLaunch.Call(params.RequestContext, butlerd.RuntimeLaunchParams{
		FullTargetPath: params.FullTargetPath,
		Candidate:      params.Candidate,
		Args:           params.Args,
		Env:            params.Env,
	})
	messages.LaunchExited.Notify(params.RequestContext, butlerd.LaunchExitedNotification{})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
