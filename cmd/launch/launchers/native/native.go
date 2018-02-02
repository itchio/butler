package native

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/launch"
)

func Register() {
	launch.Register(launch.LaunchStrategyNative, &Launcher{})
}

type Launcher struct{}

var _ launch.Launcher = (*Launcher)(nil)

func (l *Launcher) Do(params *launch.LauncherParams) error {
	ctx := params.Ctx
	conn := params.Conn
	installFolder := params.ParentParams.InstallFolder

	cwd := installFolder
	_, err := filepath.Rel(installFolder, params.FullTargetPath)
	if err != nil {
		// if it's relative, set the cwd to the folder the
		// target is in
		cwd = filepath.Dir(params.FullTargetPath)
	}

	_, err = os.Stat(params.FullTargetPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	cmd := exec.Command(params.FullTargetPath, params.Args...)
	cmd.Dir = cwd
	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	conn.Notify(ctx, "LaunchRunning", &buse.LaunchRunningNotification{})

	err = cmd.Wait()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	conn.Notify(ctx, "LaunchExited", &buse.LaunchExitedNotification{})

	return nil
}
