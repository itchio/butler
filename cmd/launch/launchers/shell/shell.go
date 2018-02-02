package shell

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/launch"
)

func Register() {
	launch.Register(launch.LaunchStrategyShell, &Launcher{})
}

type Launcher struct{}

var _ launch.Launcher = (*Launcher)(nil)

func (l *Launcher) Do(params *launch.LauncherParams) error {
	ctx := params.Ctx
	conn := params.Conn

	var r buse.ShellLaunchResult
	err := conn.Call(ctx, "ShellLaunch", &buse.ShellLaunchParams{
		ItemPath: params.FullTargetPath,
	}, &r)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
