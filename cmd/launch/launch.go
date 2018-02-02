package launch

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/itchio/butler/configurator"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/butler/manager"

	goerrors "errors"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/launch/manifest"
	"github.com/itchio/butler/cmd/operate"

	"github.com/itchio/butler/buse"
	"github.com/sourcegraph/jsonrpc2"
)

var ErrCandidateDisappeared = goerrors.New("candidate disappeared from disk!")

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

	runtime := manager.CurrentRuntime()

	consumer.Infof("→ Launching %s", operate.GameToString(params.Game))
	consumer.Infof("   on runtime %s", runtime)
	consumer.Infof("   (%s) is our install folder", params.InstallFolder)
	consumer.Infof("Passed:")
	operate.LogUpload(consumer, params.Upload, params.Build)

	receiptIn, err := bfs.ReadReceipt(params.InstallFolder)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	receiptSaidOtherwise := false

	if receiptIn.Upload != nil {
		if params.Upload == nil || params.Upload.ID != receiptIn.Upload.ID {
			receiptSaidOtherwise = true
			params.Upload = receiptIn.Upload
		}

		if receiptIn.Build != nil {
			if params.Build == nil || params.Build.ID != receiptIn.Build.ID {
				receiptSaidOtherwise = true
				params.Build = receiptIn.Build
			}
		}
	}

	if receiptSaidOtherwise {
		consumer.Warnf("Receipt had different data, switching to:")
		operate.LogUpload(consumer, params.Upload, params.Build)
	}

	var fullTargetPath string
	var strategy = LaunchStrategyUnknown
	var candidate *configurator.Candidate
	var manifestAction *manifest.Action

	appManifest, err := manifest.Read(params.InstallFolder)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	pickManifestAction := func() error {
		var err error

		if appManifest == nil {
			consumer.Infof("No manifest found at (%s)", manifest.Path(params.InstallFolder))
			return nil
		}

		actions := appManifest.ListActions(runtime)

		if len(actions) == 0 {
			consumer.Warnf("Had manifest, but no actions available (for this platform at least)")
			return nil
		}

		if len(actions) > 1 {
			consumer.Warnf("Picking actions: stub! For now, picking the first one.")
		}

		manifestAction = actions[0]

		_, err = url.Parse(manifestAction.Path)
		if err != nil {
			strategy = LaunchStrategyURL
			fullTargetPath = manifestAction.Path
			return nil
		}

		// so it's not an URL, is it a path?

		fullPath := manifestAction.ExpandPath(runtime, params.InstallFolder)
		stats, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				err = fmt.Errorf("Manifest action '%s' refers to non-existent path (%s)", manifestAction.Name, fullPath)
				return errors.Wrap(err, 0)
			}
			return errors.Wrap(err, 0)
		}

		if stats.IsDir() {
			// if it's a folder, just browse it!
			strategy = LaunchStrategyShell
			fullTargetPath = fullPath
			return nil
		}

		verdict, err := configurator.Configure(fullPath, false)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if len(verdict.Candidates) == 0 {
			err = fmt.Errorf("Wasn't able to configure (%s) - no candidates", fullPath)
			return errors.Wrap(err, 0)
		}

		strategy = LaunchStrategyNative
		fullTargetPath = fullPath
		candidate = verdict.Candidates[0]
		return nil
	}
	err = pickManifestAction()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	pickFromVerdict := func() error {
		consumer.Infof("→ Using verdict: %s", params.Verdict)

		if len(params.Verdict.Candidates) == 0 {
			return errors.New("No candidates")
		}

		if len(params.Verdict.Candidates) > 1 {
			return errors.New("More than one candidate: stub")
		}

		candidate = params.Verdict.Candidates[0]
		fullPath := filepath.Join(params.InstallFolder, candidate.Path)
		_, err := os.Stat(fullPath)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		fullTargetPath = fullPath
		return nil
	}

	if fullTargetPath == "" {
		consumer.Infof("Switching to verdict!")

		if params.Verdict == nil {
			// TODO: send back to client
			consumer.Infof("No verdict, configuring now")

			verdict, err := configurator.Configure(params.InstallFolder, false)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			params.Verdict = verdict

			err = pickFromVerdict()
			if err != nil {
				return errors.Wrap(err, 0)
			}
		} else {
			// pick from cached verdict
			err = pickFromVerdict()
			if err != nil {
				if errors.Is(err, ErrCandidateDisappeared) {
					// TODO: send back to client
					consumer.Warnf("Candidate disappeared! Re-configuring...")

					verdict, err := configurator.Configure(params.InstallFolder, false)
					if err != nil {
						return errors.Wrap(err, 0)
					}
					params.Verdict = verdict

					err = pickFromVerdict()
					if err != nil {
						return errors.Wrap(err, 0)
					}
				} else {
					return errors.Wrap(err, 0)
				}
			}
		}
	}

	if fullTargetPath == "" {
		err = fmt.Errorf("internal error: Could not determine a target path through manifest or verdict")
		return errors.Wrap(err, 0)
	}

	// FIXME: cwd is only relevant for native

	cwd := params.InstallFolder
	_, err = filepath.Rel(params.InstallFolder, fullTargetPath)
	if err != nil {
		// if it's relative, set the cwd to the folder the
		// target is in
		cwd = filepath.Dir(fullTargetPath)
	}

	consumer.Infof("→ (%s) is our target", fullTargetPath)

	_, err = os.Stat(fullTargetPath)
	if err != nil {
		// TODO: reconfigure!
		return errors.Wrap(err, 0)
	}

	var args []string

	if manifestAction != nil {
		args = append(args, manifestAction.Args...)
	}

	// TODO: launch args & env

	cmd := exec.Command(fullTargetPath, args...)
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
