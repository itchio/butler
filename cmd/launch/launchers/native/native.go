package native

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/launch"
	"github.com/itchio/butler/cmd/wipe"
)

func Register() {
	launch.Register(launch.LaunchStrategyNative, &Launcher{})
}

type Launcher struct{}

var _ launch.Launcher = (*Launcher)(nil)

func (l *Launcher) Do(params *launch.LauncherParams) error {
	ctx := params.Ctx
	conn := params.Conn
	consumer := params.Consumer
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

	envMap := make(map[string]string)
	for k, v := range params.Env {
		envMap[k] = v
	}

	// give the app its own temporary directory
	tempDir := filepath.Join(params.ParentParams.InstallFolder, ".itch", "temp")
	err = os.MkdirAll(tempDir, 0755)
	if err != nil {
		consumer.Warnf("Could not make temporary directory: %s", err.Error())
	} else {
		defer wipe.Do(consumer, tempDir)
		envMap["TMP"] = tempDir
		envMap["TEMP"] = tempDir
		consumer.Infof("Giving app temp dir (%s)", tempDir)
	}

	var envKeys []string
	for k := range envMap {
		envKeys = append(envKeys, k)
	}
	consumer.Infof("Environment variables passed: %s", strings.Join(envKeys, ", "))

	// TODO: sanitize environment somewhat?
	envBlock := os.Environ()
	for k, v := range envMap {
		envBlock = append(envBlock, fmt.Sprintf("%s=%s", k, v))
	}

	cmd.Env = envBlock

	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	startTime := time.Now()

	conn.Notify(ctx, "LaunchRunning", &buse.LaunchRunningNotification{})
	exitCode, err := waitCommand(cmd)
	conn.Notify(ctx, "LaunchExited", &buse.LaunchExitedNotification{})

	if err != nil {
		return errors.Wrap(err, 0)
	}

	runDuration := time.Since(startTime)

	if exitCode != 0 {
		var signedExitCode = int64(exitCode)
		if runtime.GOOS == "windows" {
			// Windows uses 32-bit unsigned integers as exit codes,[11] although the
			// command interpreter treats them as signed.[12] If a process fails
			// initialization, a Windows system error code may be returned.[13][14]
			signedExitCode = int64(int32(signedExitCode))

			// The line above turns `4294967295` into -1
		}

		exeName := filepath.Base(params.FullTargetPath)
		msg := fmt.Sprintf("Exit code 0x%x (%d) for (%s)", uint32(exitCode), signedExitCode, exeName)
		consumer.Warnf(msg)

		if runDuration.Seconds() > 2 {
			consumer.Warnf("That's after running for %s, ignoring non-zero exit code", runDuration)
		} else {
			return errors.New(msg)
		}
	}

	return nil
}

func waitCommand(cmd *exec.Cmd) (int, error) {
	err := cmd.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus(), nil
			}
		}

		return 127, err
	}

	return 0, nil
}
