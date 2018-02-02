package launch

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/operate"

	"github.com/itchio/butler/buse"
	"github.com/sourcegraph/jsonrpc2"
)

func Do(ctx context.Context, conn *jsonrpc2.Conn, params *buse.LaunchParams) (err error) {
	consumer, err := operate.NewStateConsumer(&operate.NewStateConsumerParams{
		Ctx:     ctx,
		Conn:    conn,
		LogFile: nil,
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if params.InstallFolder == "" {
		return errors.New("InstallFolder must be specified")
	}

	consumer.Infof("Launching (%s)", params.InstallFolder)

	if params.Verdict == nil {
		return errors.New("Launching with nil verdict: stub!")
	}

	if len(params.Verdict.Candidates) == 0 {
		return errors.New("No candidates")
	}

	if len(params.Verdict.Candidates) > 1 {
		return errors.New("More than one candidate: stub!")
	}

	candidate := params.Verdict.Candidates[0]
	// TODO: flavors
	fullExePath := filepath.Join(params.InstallFolder, candidate.Path)
	cwd := filepath.Dir(fullExePath)

	_, err = os.Stat(fullExePath)
	if err != nil {
		// TODO: reconfigure!
		return errors.Wrap(err, 0)
	}

	consumer.Infof("Picked candidate: %s", candidate)

	cmd := exec.Command(fullExePath)
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
