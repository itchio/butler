//+build darwin

package runner

import (
	"os"
	"os/exec"
	"os/signal"

	"github.com/itchio/ox/macox"
	"github.com/pkg/errors"
)

type appRunner struct {
	params *RunnerParams
}

var _ Runner = (*appRunner)(nil)

func newAppRunner(params *RunnerParams) (Runner, error) {
	ar := &appRunner{
		params: params,
	}
	return ar, nil
}

func (ar *appRunner) Prepare() error {
	// nothing to prepare
	return nil
}

func (ar *appRunner) Run() error {
	params := ar.params

	return RunAppBundle(
		params,
		params.FullTargetPath,
	)
}

func RunAppBundle(params *RunnerParams, bundlePath string) error {
	consumer := params.Consumer

	var args = []string{
		"-W",
		bundlePath,
		"--args",
	}
	args = append(args, params.Args...)

	consumer.Infof("App bundle is (%s)", bundlePath)

	binaryPath, err := macox.GetExecutablePath(bundlePath)
	if err != nil {
		return errors.WithStack(err)
	}

	consumer.Infof("Actual binary is (%s)", binaryPath)

	cmd := exec.Command("open", args...)
	// I doubt this matters
	cmd.Dir = params.Dir
	cmd.Env = params.Env
	// 'open' does not relay stdout or stderr, so we don't
	// even bother setting them

	processDone := make(chan struct{})

	go func() {
		// catch SIGINT and kill the child if needed
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)

		consumer.Infof("Signal handler installed...")

		// Block until a signal is received.
		select {
		case <-params.Ctx.Done():
			consumer.Warnf("Context done!")
		case s := <-c:
			consumer.Warnf("Got signal: %v", s)
		case <-processDone:
			return
		}

		consumer.Warnf("Killing app...")
		// TODO: kill the actual binary, not the app
		cmd := exec.Command("pkill", "-f", binaryPath)
		err := cmd.Run()
		if err != nil {
			consumer.Errorf("While killing: %s", err.Error())
		}
	}()

	err = cmd.Run()
	close(processDone)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
