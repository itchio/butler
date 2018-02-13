package launch

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/butler/configurator"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/butler/manager"
	itchio "github.com/itchio/go-itchio"

	goerrors "errors"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/launch/manifest"
	"github.com/itchio/butler/cmd/operate"

	"github.com/itchio/butler/buse"
)

var ErrNoCandidates = goerrors.New("no candidates")
var ErrCandidateDisappeared = goerrors.New("candidate disappeared from disk!")

func Do(ctx context.Context, conn operate.Conn, params *buse.LaunchParams) (err error) {
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

	if receiptIn != nil {
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

		// is it a path?

		fullPath := manifestAction.ExpandPath(runtime, params.InstallFolder)
		stats, err := os.Stat(fullPath)
		if err != nil {
			// is it an URL?
			{
				_, err := url.Parse(manifestAction.Path)
				if err == nil {
					strategy = LaunchStrategyURL
					fullTargetPath = manifestAction.Path
					return nil
				}
			}

			if os.IsNotExist(err) {
				err = fmt.Errorf("Manifest action '%s' refers to non-existent path (%s)", manifestAction.Name, fullPath)
				return errors.Wrap(err, 0)
			}
			return errors.Wrap(err, 0)
		}

		if stats.IsDir() {
			// is it an app bundle?
			if runtime.Platform == manager.ItchPlatformOSX && strings.HasSuffix(strings.ToLower(fullPath), ".app") {
				strategy = LaunchStrategyNative
				fullTargetPath = fullPath
				return nil
			}

			// if it's a folder, just browse it!
			strategy = LaunchStrategyShell
			fullTargetPath = fullPath
			return nil
		}

		verdict, err := configurator.Configure(fullPath, false)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if len(verdict.Candidates) > 0 {
			strategy = flavorToStrategy(verdict.Candidates[0].Flavor)
			candidate = verdict.Candidates[0]
		} else {
			// must not be an executable, that's ok, just open it
			strategy = LaunchStrategyShell
		}

		fullTargetPath = fullPath
		return nil
	}
	err = pickManifestAction()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	pickFromVerdict := func() error {
		consumer.Infof("→ Using verdict: %s", params.Verdict)

		switch len(params.Verdict.Candidates) {
		case 0:
			return ErrNoCandidates
		case 1:
			candidate = params.Verdict.Candidates[0]
		default:
			nameMap := make(map[string]*configurator.Candidate)

			fakeActions := []*manifest.Action{}
			for _, c := range params.Verdict.Candidates {
				name := fmt.Sprintf("%s (%s)", c.Path, humanize.IBytes(uint64(c.Size)))
				nameMap[name] = c
				fakeActions = append(fakeActions, &manifest.Action{
					Name: name,
					Path: c.Path,
				})
			}

			var r buse.PickManifestActionResult
			err := conn.Call(ctx, "PickManifestAction", &buse.PickManifestActionParams{
				Actions: fakeActions,
			}, &r)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			if r.Name == "" {
				return operate.ErrAborted
			}

			candidate = nameMap[r.Name]
		}

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
			consumer.Infof("No verdict, configuring now")

			verdict, err := configurator.Configure(params.InstallFolder, false)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			params.Verdict = verdict

			var r buse.SaveVerdictResult
			err = conn.Call(ctx, "SaveVerdict", &buse.SaveVerdictParams{
				Verdict: verdict,
			}, &r)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			err = pickFromVerdict()
			if err != nil {
				if !errors.Is(err, ErrNoCandidates) {
					return errors.Wrap(err, 0)
				}
			}
		} else {
			// pick from cached verdict
			err = pickFromVerdict()
			if err != nil {
				redoReason := ""
				if errors.Is(err, ErrCandidateDisappeared) {
					redoReason = "Candidate disappeared!"
				} else if errors.Is(err, ErrNoCandidates) {
					redoReason = "No candidates!"
				}

				if redoReason != "" {
					consumer.Warnf("%s Re-configuring...", redoReason)

					verdict, err := configurator.Configure(params.InstallFolder, false)
					if err != nil {
						return errors.Wrap(err, 0)
					}
					params.Verdict = verdict

					var r buse.SaveVerdictResult
					err = conn.Call(ctx, "SaveVerdict", &buse.SaveVerdictParams{
						Verdict: verdict,
					}, &r)
					if err != nil {
						return errors.Wrap(err, 0)
					}

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
	if params.Upload != nil {
		switch params.Upload.Type {
		case "soundtrack", "book", "video", "documentation", "mod", "audio_assets", "graphical_assets", "sourcecode":
			consumer.Infof("Forcing shell strategy because upload is of type (%s)", params.Upload.Type)
			fullTargetPath = "."
			strategy = LaunchStrategyShell
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

		strategy = flavorToStrategy(candidate.Flavor)
	}

	consumer.Infof("→ Using strategy (%s)", strategy)
	consumer.Infof("  (%s) is our target", fullTargetPath)

	launcher := launchers[strategy]
	if launcher == nil {
		err := fmt.Errorf("no launcher for strategy (%s)", strategy)
		return errors.Wrap(err, 0)
	}

	var args []string = []string{}
	var env = make(map[string]string)

	if manifestAction != nil {
		args = append(args, manifestAction.Args...)

		if manifestAction.Scope != "" {
			const onlyPermittedScope = "profile:me"
			if manifestAction.Scope != onlyPermittedScope {
				err := fmt.Errorf("Game asked for scope (%s), asking for permission is unimplemented for now", manifestAction.Scope)
				return errors.Wrap(err, 0)
			}

			client, err := operate.ClientFromCredentials(params.Credentials)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			res, err := client.Subkey(&itchio.SubkeyParams{
				GameID: params.Game.ID,
				Scope:  manifestAction.Scope,
			})
			if err != nil {
				return errors.Wrap(err, 0)
			}

			consumer.Infof("Got subkey (%d chars, expires %s)", len(res.Key), res.ExpiresAt)

			env["ITCHIO_API_KEY"] = res.Key
			env["ITCHIO_API_KEY_EXPIRES_AT"] = res.ExpiresAt
		}
	}

	sandbox := params.Sandbox
	if manifestAction != nil && manifestAction.Sandbox {
		consumer.Infof("Enabling sandbox because of manifest opt-in")
		sandbox = true
	}

	launcherParams := &LauncherParams{
		Conn:     conn,
		Ctx:      ctx,
		Consumer: consumer,

		FullTargetPath: fullTargetPath,
		Candidate:      candidate,
		AppManifest:    appManifest,
		Action:         manifestAction,
		Sandbox:        sandbox,
		Args:           args,
		Env:            env,

		PrereqsDir:    params.PrereqsDir,
		Credentials:   params.Credentials,
		InstallFolder: params.InstallFolder,
		Runtime:       runtime,
	}

	err = launcher.Do(launcherParams)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func flavorToStrategy(flavor configurator.Flavor) LaunchStrategy {
	switch flavor {
	// HTML
	case configurator.FlavorHTML:
		return LaunchStrategyHTML
	// Native
	case configurator.FlavorNativeLinux:
		return LaunchStrategyNative
	case configurator.FlavorNativeMacos:
		return LaunchStrategyNative
	case configurator.FlavorNativeWindows:
		return LaunchStrategyNative
	case configurator.FlavorAppMacos:
		return LaunchStrategyNative
	case configurator.FlavorScript:
		return LaunchStrategyNative
	case configurator.FlavorScriptWindows:
		return LaunchStrategyNative
	case configurator.FlavorJar:
		return LaunchStrategyNative
	case configurator.FlavorLove:
		return LaunchStrategyNative
	default:
		return LaunchStrategyUnknown
	}
}
