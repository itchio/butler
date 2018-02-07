package native

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/itchio/butler/redist"
	"github.com/itchio/wharf/eos"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/launch"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/cmd/prereqs"
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

	err = handlePrereqs(params)
	if err != nil {
		if errors.Is(err, operate.ErrAborted) {
			return err
		}

		consumer.Warnf("While handling prereqs: %s", err.Error())

		var r buse.PrereqsFailedResult
		var errorStack string
		if se, ok := err.(*errors.Error); ok {
			errorStack = se.ErrorStack()
		}

		err = conn.Call(ctx, "PrereqsFailed", &buse.PrereqsFailedParams{
			Error:      err.Error(),
			ErrorStack: errorStack,
		}, &r)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if r.Continue {
			// continue!
			consumer.Warnf("Continuing after prereqs failure because user told us to")
		} else {
			// abort
			consumer.Warnf("Giving up after prereqs failure because user asked us to")
			return operate.ErrAborted
		}
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

	const maxLines = 40
	stdout := newOutputCollector(maxLines)
	cmd.Stdout = stdout

	stderr := newOutputCollector(maxLines)
	cmd.Stderr = stderr

	err = func() error {
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

func handlePrereqs(params *launch.LauncherParams) error {
	consumer := params.Consumer
	ctx := params.Ctx
	conn := params.Conn

	if runtime.GOOS != "windows" {
		consumer.Infof("Not on windows, ignoring prereqs")
		return nil
	}

	if params.AppManifest == nil {
		consumer.Infof("No manifest, no prereqs")
		return nil
	}

	if len(params.AppManifest.Prereqs) == 0 {
		consumer.Infof("Got manifest but no prereqs requested")
		return nil
	}

	// TODO: store done somewhere
	prereqsDir := params.ParentParams.PrereqsDir

	// TODO: cache maybe
	consumer.Infof("Fetching prereqs registry...")

	registry := &redist.RedistRegistry{}

	err := func() error {
		registryURL := fmt.Sprintf("%s/info.json", prereqs.RedistsBaseURL)
		f, err := eos.Open(registryURL)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		dec := json.NewDecoder(f)
		err = dec.Decode(registry)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	var initialNames []string
	for _, p := range params.AppManifest.Prereqs {
		initialNames = append(initialNames, p.Name)
	}

	pa, err := prereqs.AssessPrereqs(consumer, registry, initialNames)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Infof("%d done: %s", len(pa.Done), strings.Join(pa.Done, ", "))
	consumer.Infof("%d todo: %s", len(pa.Todo), strings.Join(pa.Todo, ", "))

	if len(pa.Todo) == 0 {
		consumer.Infof("Everything done!")
		return nil
	}

	consumer.Infof("%d prereqs to install: %s", len(pa.Todo), strings.Join(pa.Todo, ", "))

	{
		psn := &buse.PrereqsStartedNotification{
			Tasks: make(map[string]*buse.PrereqTask),
		}
		for i, name := range pa.Todo {
			psn.Tasks[name] = &buse.PrereqTask{
				FullName: registry.Entries[name].FullName,
				Order:    i,
			}
		}

		err = conn.Notify(ctx, "PrereqsStarted", psn)
		if err != nil {
			consumer.Warnf(err.Error())
		}
	}

	tsc := &prereqs.TaskStateConsumer{
		OnState: func(state *buse.PrereqsTaskStateNotification) {
			err = conn.Notify(ctx, "PrereqsTaskState", state)
			if err != nil {
				consumer.Warnf(err.Error())
			}
		},
	}

	err = prereqs.FetchPrereqs(consumer, tsc, prereqsDir, registry, pa.Todo)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	plan := &prereqs.PrereqPlan{}

	for _, name := range pa.Todo {
		plan.Tasks = append(plan.Tasks, &prereqs.PrereqTask{
			Name:    name,
			WorkDir: filepath.Join(prereqsDir, name),
			Info:    *registry.Entries[name],
		})
	}

	err = prereqs.ElevatedInstall(consumer, plan, tsc)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = conn.Notify(ctx, "PrereqsEnded", &buse.PrereqsEndedNotification{})
	if err != nil {
		consumer.Warnf(err.Error())
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

//

type outputCollector struct {
	lines  []string
	writer io.Writer
}

var _ io.Writer = (*outputCollector)(nil)

func newOutputCollector(maxLines int) *outputCollector {
	pipeR, pipeW := io.Pipe()

	oc := &outputCollector{
		writer: pipeW,
	}

	go func() {
		s := bufio.NewScanner(pipeR)
		for s.Scan() {
			line := s.Text()
			oc.lines = append(oc.lines, line)

			if len(oc.lines) > maxLines {
				oc.lines = oc.lines[1:]
			}
		}
	}()

	return oc
}

func (oc *outputCollector) Lines() []string {
	return oc.lines
}

func (oc *outputCollector) Write(p []byte) (int, error) {
	return oc.writer.Write(p)
}
