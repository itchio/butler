package html

import (
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/launch"
)

func Register() {
	launch.Register(launch.LaunchStrategyHTML, &Launcher{})
}

type Launcher struct{}

var _ launch.Launcher = (*Launcher)(nil)

func (l *Launcher) Do(params *launch.LauncherParams) error {
	ctx := params.Ctx
	conn := params.Conn

	rootFolder := filepath.Dir(params.FullTargetPath)
	indexPath := filepath.Base(params.FullTargetPath)

	var r buse.HTMLLaunchResult
	err := conn.Call(ctx, "HTMLLaunch", &buse.HTMLLaunchParams{
		RootFolder: rootFolder,
		IndexPath:  indexPath,
	}, &r)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
