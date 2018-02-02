package launch

import (
	"context"
	"fmt"
	"net/url"
	"os"
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

		if len(actions) == 1 {
			manifestAction = actions[0]
		} else {
			var r buse.PickManifestActionResult
			err := conn.Call(ctx, "PickManifestAction", &buse.PickManifestActionParams{
				Actions: actions,
			}, &r)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			if r.Name == "" {
				return operate.ErrAborted
			}

			for _, action := range actions {
				if action.Name == r.Name {
					manifestAction = action
					break
				}
			}
		}

		if manifestAction == nil {
			consumer.Warnf("No manifest action picked")
			return nil
		}

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
			return nil
		}

		if len(params.Verdict.Candidates) > 1 {
			// TODO: ask client to disambiguate
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
		consumer.Warnf("No target from manifest or verdict, falling back to shell strategy")
		fullTargetPath = "."
		strategy = LaunchStrategyShell
	}

	if strategy == LaunchStrategyUnknown {
		if candidate == nil {
			err := fmt.Errorf("could not determine launch strategy for %s", fullTargetPath)
			return errors.Wrap(err, 0)
		}

		switch candidate.Flavor {
		// HTML
		case configurator.FlavorHTML:
			strategy = LaunchStrategyHTML
		// Native
		case configurator.FlavorNativeLinux:
			strategy = LaunchStrategyNative
		case configurator.FlavorNativeMacos:
			strategy = LaunchStrategyNative
		case configurator.FlavorNativeWindows:
			strategy = LaunchStrategyNative
		case configurator.FlavorAppMacos:
			strategy = LaunchStrategyNative
		case configurator.FlavorScript:
			strategy = LaunchStrategyNative
		case configurator.FlavorScriptWindows:
			strategy = LaunchStrategyNative
		case configurator.FlavorJar:
			strategy = LaunchStrategyNative
		case configurator.FlavorLove:
			strategy = LaunchStrategyNative
		default:
			err := fmt.Errorf("unknown flavor (%s) for target (%s)", candidate.Flavor, fullTargetPath)
			return errors.Wrap(err, 0)
		}
	}

	if params.Upload != nil {
		switch params.Upload.Type {
		case "html":
			consumer.Infof("Forcing html because of upload")
		case "soundtrack", "book", "video", "documentation", "mod", "audio_assets", "graphical_assets", "sourcecode":
			consumer.Infof("Forcing html because of upload")
		}
	}

	consumer.Infof("→ Using strategy (%s)", strategy)
	consumer.Infof("  (%s) is our target", fullTargetPath)

	launcher := launchers[strategy]
	if launcher == nil {
		err := fmt.Errorf("no launcher for strategy (%s)", strategy)
		return errors.Wrap(err, 0)
	}

	launcherParams := &LauncherParams{
		Conn:         conn,
		Ctx:          ctx,
		Consumer:     consumer,
		ParentParams: params,

		FullTargetPath: fullTargetPath,
		Candidate:      candidate,
		Action:         manifestAction,
		Sandbox:        params.Sandbox,
		Args:           nil,
		Env:            nil,
	}

	err = launcher.Do(launcherParams)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
