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

	"github.com/itchio/butler/configurator"

	"github.com/itchio/butler/buse/messages"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/wipe"
	"github.com/itchio/butler/endpoints/launch"
	"github.com/itchio/butler/runner"
)

func Register() {
	launch.RegisterLauncher(launch.LaunchStrategyNative, &Launcher{})
}

type Launcher struct{}

var _ launch.Launcher = (*Launcher)(nil)

func (l *Launcher) Do(params *launch.LauncherParams) error {
	consumer := params.RequestContext.Consumer
	installFolder := params.InstallFolder

	cwd := installFolder
	_, err := filepath.Rel(installFolder, params.FullTargetPath)
	if err == nil {
		// if it's relative, set the cwd to the folder the
		// target is in
		cwd = filepath.Dir(params.FullTargetPath)
	}

	_, err = os.Stat(params.FullTargetPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = handlePrereqs(params)
	if err != nil {
		if errors.Is(err, &buse.ErrAborted{}) {
			return err
		}

		consumer.Warnf("While handling prereqs: %s", err.Error())

		var errorStack string
		if se, ok := err.(*errors.Error); ok {
			errorStack = se.ErrorStack()
		}

		r, err := messages.PrereqsFailed.Call(params.RequestContext, &buse.PrereqsFailedParams{
			Error:      err.Error(),
			ErrorStack: errorStack,
		})
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if r.Continue {
			// continue!
			consumer.Warnf("Continuing after prereqs failure because user told us to")
		} else {
			// abort
			consumer.Warnf("Giving up after prereqs failure because user asked us to")
			return &buse.ErrAborted{}
		}
	}

	envMap := make(map[string]string)
	for k, v := range params.Env {
		envMap[k] = v
	}

	// give the app its own temporary directory
	tempDir := filepath.Join(params.InstallFolder, ".itch", "temp")
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

	const maxLines = 40
	stdout := newOutputCollector(maxLines)
	stderr := newOutputCollector(maxLines)

	fullTargetPath := params.FullTargetPath
	name := params.FullTargetPath
	args := params.Args

	if params.Candidate != nil && params.Candidate.Flavor == configurator.FlavorLove {
		// TODO: add prereqs when that happens
		args = append([]string{name}, args...)
		name = "love"
		fullTargetPath = "love"
	}

	runParams := &runner.RunnerParams{
		RequestContext: params.RequestContext,
		Ctx:            params.Ctx,

		Sandbox: params.Sandbox,

		FullTargetPath: fullTargetPath,

		Name:   name,
		Dir:    cwd,
		Args:   args,
		Env:    envBlock,
		Stdout: stdout,
		Stderr: stderr,

		PrereqsDir:    params.PrereqsDir,
		Credentials:   params.Credentials,
		InstallFolder: params.InstallFolder,
		Runtime:       params.Runtime,
	}

	run, err := runner.GetRunner(runParams)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = run.Prepare()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = func() error {
		startTime := time.Now()

		messages.LaunchRunning.Notify(params.RequestContext, &buse.LaunchRunningNotification{})
		exitCode, err := interpretRunError(run.Run())
		messages.LaunchExited.Notify(params.RequestContext, &buse.LaunchExitedNotification{})
		if err != nil {
			return errors.Wrap(err, 0)
		}

		runDuration := time.Since(startTime)
		err = params.RecordPlayTime(runDuration)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if exitCode != 0 {
			var signedExitCode = int64(exitCode)
			if runtime.GOOS == "windows" {
				// Windows uses 32-bit unsigned integers as exit codes, although the
				// command interpreter treats them as signed. If a process fails
				// initialization, a Windows system error code may be returned.
				signedExitCode = int64(int32(signedExitCode))

				// The line above turns `4294967295` into -1
			}

			exeName := filepath.Base(params.FullTargetPath)
			msg := fmt.Sprintf("Exit code 0x%x (%d) for (%s)", uint32(exitCode), signedExitCode, exeName)
			consumer.Warnf(msg)

			if runDuration.Seconds() > 10 {
				consumer.Warnf("That's after running for %s, ignoring non-zero exit code", runDuration)
			} else {
				return errors.New(msg)
			}
		}

		return nil
	}()

	if err != nil {
		consumer.Errorf("Had error: %s", err.Error())
		if len(stderr.Lines()) == 0 {
			consumer.Errorf("No messages for standard error")
			consumer.Errorf("→ Standard error: empty")
		} else {
			consumer.Errorf("→ Standard error ================")
			for _, l := range stderr.Lines() {
				consumer.Errorf("  %s", l)
			}
			consumer.Errorf("=================================")
		}

		if len(stdout.Lines()) == 0 {
			consumer.Errorf("→ Standard output: empty")
		} else {
			consumer.Errorf("→ Standard output ===============")
			for _, l := range stdout.Lines() {
				consumer.Errorf("  %s", l)
			}
			consumer.Errorf("=================================")
		}
		consumer.Errorf("Relaying launch failure.")
		return errors.Wrap(err, 0)
	}

	return nil
}

func interpretRunError(err error) (int, error) {
	if err != nil {
		if exitError, ok := AsExitError(err); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus(), nil
			}
		}

		return 127, err
	}

	return 0, nil
}

func AsExitError(err error) (*exec.ExitError, bool) {
	if err == nil {
		return nil, false
	}

	if se, ok := err.(*errors.Error); ok {
		return AsExitError(se.Err)
	}

	if ee, ok := err.(*exec.ExitError); ok {
		return ee, true
	}

	return nil, false
}
