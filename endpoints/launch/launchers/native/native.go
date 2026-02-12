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

	"github.com/itchio/headway/state"
	"github.com/itchio/httpkit/neterr"

	"github.com/itchio/pelican"

	"github.com/itchio/dash"

	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/butler/shell"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/elevate"
	"github.com/itchio/butler/cmd/wipe"
	"github.com/itchio/butler/endpoints/launch"
	"github.com/itchio/smaug/runner"
	"github.com/pkg/errors"
)

func Register() {
	launch.RegisterLauncher(butlerd.LaunchStrategyNative, &Launcher{})
}

type Launcher struct{}

var _ launch.Launcher = (*Launcher)(nil)

func (l *Launcher) Do(params launch.LauncherParams) error {
	consumer := params.RequestContext.Consumer
	installFolder := params.InstallFolder

	cwd := params.WorkingDirectory
	if cwd == "" {
		_, err := filepath.Rel(installFolder, params.FullTargetPath)
		if err == nil {
			// if it's relative, set the cwd to the folder the
			// target is in
			cwd = filepath.Dir(params.FullTargetPath)
		} else {
			cwd = params.InstallFolder
		}
	}

	_, err := os.Stat(params.FullTargetPath)
	if err != nil {
		return errors.WithStack(err)
	}

	err = configureTargetIfNeeded(params)
	if err != nil {
		consumer.Warnf("Could not configure launch target: %s", err.Error())
	}

	err = fillPeInfoIfNeeded(params)
	if err != nil {
		consumer.Warnf("Could not determine PE info: %s", err.Error())
	}

	err = handlePrereqs(params)
	if err != nil {
		if be, ok := butlerd.AsButlerdError(err); ok {
			switch butlerd.Code(be.RpcErrorCode()) {
			case butlerd.CodeOperationAborted, butlerd.CodeOperationCancelled:
				return be
			}
		}

		consumer.Warnf("While handling prereqs: %+v", err)

		if neterr.IsNetworkError(err) {
			err = butlerd.CodeNetworkDisconnected
		}

		r, err := messages.PrereqsFailed.Call(params.RequestContext, butlerd.PrereqsFailedParams{
			Error:      err.Error(),
			ErrorStack: fmt.Sprintf("%+v", err),
		})
		if err != nil {
			return errors.WithStack(err)
		}

		if r.Continue {
			// continue!
			consumer.Warnf("Continuing after prereqs failure because user told us to")
		} else {
			// abort
			consumer.Warnf("Giving up after prereqs failure because user asked us to")
			return errors.WithStack(butlerd.CodeOperationAborted)
		}
	}

	envMap := make(map[string]string)
	for k, v := range params.Env {
		envMap[k] = v
	}

	// give the app its own temporary directory
	tempDir := filepath.Join(params.InstallFolder, ".itch", "temp")
	err = os.MkdirAll(tempDir, 0o755)
	if err != nil {
		consumer.Warnf("Could not make temporary directory: %s", err.Error())
	} else {
		defer wipe.Do(consumer, tempDir)
		envMap["TMP"] = tempDir
		envMap["TEMP"] = tempDir
		consumer.Infof("Giving app temp dir (%s)", tempDir)
	}

	if params.Sandbox {
		envMap["ITCHIO_SANDBOX"] = "1"
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
	args := params.Args

	if params.Host.Wrapper != nil {
		wr := params.Host.Wrapper

		// TODO: DRY (see runself)
		var wrapperArgs []string
		wrapperArgs = append(wrapperArgs, wr.BeforeTarget...)
		if wr.NeedRelativeTarget {
			cwd = filepath.Dir(fullTargetPath)
			relativeTarget := filepath.Base(fullTargetPath)
			wrapperArgs = append(wrapperArgs, relativeTarget)
		} else {
			wrapperArgs = append(wrapperArgs, fullTargetPath)
		}
		wrapperArgs = append(wrapperArgs, wr.BetweenTargetAndArgs...)
		wrapperArgs = append(wrapperArgs, args...)
		wrapperArgs = append(wrapperArgs, wr.AfterArgs...)
		args = wrapperArgs
		fullTargetPath = wr.WrapperBinary
	}
	name := params.FullTargetPath

	if params.Candidate != nil {
		switch params.Candidate.Flavor {
		case dash.FlavorLove:
			// TODO: add prereqs when that happens
			args = append([]string{name}, args...)
			name = "love"
			fullTargetPath = "love"
			consumer.Infof("We're launching a .love bundle, trying to execute with love runtime")
		case dash.FlavorJar:
			javaPath, err := exec.LookPath("java")
			if err != nil {
				return butlerd.CodeJavaRuntimeNeeded
			}

			args = append([]string{"-jar", name}, args...)
			name = "java"
			fullTargetPath = javaPath
			consumer.Infof("We're launching a .jar, trying to execute with Java runtime")
		}
	}

	console := false
	if params.Action != nil && params.Action.Console {
		console = true
		consumer.Infof("Console launch requested")
	}

	runParams := runner.RunnerParams{
		Consumer: consumer,
		Ctx:      params.Ctx,

		Sandbox: params.Sandbox,
		Console: console,

		FullTargetPath: fullTargetPath,

		Name:   name,
		Dir:    cwd,
		Args:   args,
		Env:    envBlock,
		Stdout: stdout,
		Stderr: stderr,

		TempDir:       tempDir,
		InstallFolder: params.InstallFolder,
		Runtime:       params.Host.Runtime,

		AttachParams:   l.AttachParams(params),
		FirejailParams: l.FirejailParams(params),
		FujiParams:     l.FujiParams(params),
	}

	run, err := runner.GetRunner(runParams)
	if err != nil {
		return errors.WithStack(err)
	}

	err = run.Prepare()
	if err != nil {
		return errors.WithStack(err)
	}

	err = func() error {
		startTime := time.Now().UTC()
		params.SessionStarted()

		messages.LaunchRunning.Notify(params.RequestContext, butlerd.LaunchRunningNotification{})
		exitCode, err := interpretRunError(run.Run())
		messages.LaunchExited.Notify(params.RequestContext, butlerd.LaunchExitedNotification{})
		if err != nil {
			return err
		}

		runDuration := time.Since(startTime)

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
			consumer.Warnf("%s", msg)

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
		return err
	}

	return nil
}

func (l *Launcher) FirejailParams(params launch.LauncherParams) runner.FirejailParams {
	// Prefer system-installed firejail
	if systemPath, err := exec.LookPath("firejail"); err == nil {
		return runner.FirejailParams{BinaryPath: systemPath}
	}

	// Fall back to prereqs-installed firejail
	name := fmt.Sprintf("firejail-%s", params.Host.Runtime.Arch())
	binaryPath := filepath.Join(params.PrereqsDir, name, "firejail")
	return runner.FirejailParams{BinaryPath: binaryPath}
}

func (l *Launcher) FujiParams(params launch.LauncherParams) runner.FujiParams {
	consumer := params.RequestContext.Consumer

	return runner.FujiParams{
		Settings: mansion.GetFujiSettings(),
		PerformElevatedSetup: func() error {
			r, err := messages.AllowSandboxSetup.Call(params.RequestContext, butlerd.AllowSandboxSetupParams{})
			if err != nil {
				return errors.WithStack(err)
			}

			if !r.Allow {
				return errors.WithStack(butlerd.CodeOperationAborted)
			}
			consumer.Infof("Proceeding with sandbox setup...")

			res, err := shell.RunSelf(shell.RunSelfParams{
				Host:       params.Host,
				Consumer:   consumer,
				PrereqsDir: params.PrereqsDir,
				Args: []string{
					"--elevate",
					"fuji",
					"setup",
				},
			})
			if err != nil {
				return errors.WithStack(err)
			}

			if res.ExitCode != 0 {
				if res.ExitCode == elevate.ExitCodeAccessDenied {
					return errors.WithStack(butlerd.CodeOperationAborted)
				}
			}

			err = shell.CheckExitCode(res.ExitCode, err)
			if err != nil {
				return errors.WithStack(err)
			}
			return nil
		},
	}
}

func (l *Launcher) AttachParams(params launch.LauncherParams) runner.AttachParams {
	return runner.AttachParams{
		BringWindowToForeground: func(hwnd int64) {
			setWindowForeground(hwnd)
		},
	}
}

func configureTargetIfNeeded(params launch.LauncherParams) error {
	if params.Candidate != nil {
		// already configured
		return nil
	}

	v, err := dash.Configure(params.FullTargetPath, dash.ConfigureParams{
		Consumer: params.RequestContext.Consumer,
		Filter:   filtering.FilterPaths,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	if len(v.Candidates) == 0 {
		return errors.Errorf("0 candidates after configure")
	}

	params.Candidate = v.Candidates[0]
	return nil
}

func fillPeInfoIfNeeded(params launch.LauncherParams) error {
	c := params.Candidate
	if c == nil {
		// no candidate for some reason?
		return nil
	}

	if c.Flavor != dash.FlavorNativeWindows {
		// not an .exe, ignore
		return nil
	}

	var err error
	f, err := os.Open(params.FullTargetPath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()

	consumer := params.RequestContext.Consumer

	var peLines []string
	memConsumer := &state.Consumer{
		OnMessage: func(lvl string, msg string) {
			peLines = append(peLines, fmt.Sprintf("[%s] %s", lvl, msg))
		},
	}

	params.PeInfo, err = pelican.Probe(f, pelican.ProbeParams{
		Consumer: memConsumer,
	})
	if err != nil {
		consumer.Warnf("pelican failed on (%s), full log:\n%s", params.FullTargetPath, strings.Join(peLines, "\n"))
		return errors.WithStack(err)
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

type causer interface {
	Cause() error
}

func AsExitError(err error) (*exec.ExitError, bool) {
	if err == nil {
		return nil, false
	}

	if se, ok := err.(causer); ok {
		return AsExitError(se.Cause())
	}

	if ee, ok := err.(*exec.ExitError); ok {
		return ee, true
	}

	return nil, false
}
