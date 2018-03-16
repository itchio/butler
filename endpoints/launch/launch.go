package launch

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	goerrors "errors"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/configurator"
	"github.com/itchio/butler/endpoints/launch/manifest"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/butler/manager"
	itchio "github.com/itchio/go-itchio"
)

var ErrNoCandidates = goerrors.New("no candidates")
var ErrCandidateDisappeared = goerrors.New("candidate disappeared from disk!")

func Register(router *buse.Router) {
	messages.Launch.Register(router, Launch)
}

func Launch(rc *buse.RequestContext, params *buse.LaunchParams) (*buse.LaunchResult, error) {
	consumer := rc.Consumer

	cave := operate.ValidateCave(rc, params.CaveID)
	installFolder := cave.GetInstallFolder(rc.DB())

	_, err := os.Stat(installFolder)
	if err != nil && os.IsNotExist(err) {
		return nil, &buse.RpcError{
			Code:    int64(buse.CodeInstallFolderDisappeared),
			Message: fmt.Sprintf("Could not find install folder (%s)", installFolder),
		}
	}

	game := cave.Game
	upload := cave.Upload
	build := cave.Build
	verdict := cave.GetVerdict()
	credentials := operate.CredentialsForGame(rc.DB(), consumer, game)

	runtime := manager.CurrentRuntime()

	consumer.Infof("→ Launching %s", operate.GameToString(game))
	consumer.Infof("   on runtime %s", runtime)
	consumer.Infof("   (%s) is our install folder", installFolder)
	consumer.Infof("Passed:")
	operate.LogUpload(consumer, upload, build)

	receiptIn, err := bfs.ReadReceipt(installFolder)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	receiptSaidOtherwise := false

	if receiptIn != nil {
		if receiptIn.Upload != nil {
			if upload == nil || upload.ID != receiptIn.Upload.ID {
				receiptSaidOtherwise = true
				upload = receiptIn.Upload
			}

			if receiptIn.Build != nil {
				if build == nil || build.ID != receiptIn.Build.ID {
					receiptSaidOtherwise = true
					build = receiptIn.Build
				}
			}
		}
	}

	if receiptSaidOtherwise {
		consumer.Warnf("Receipt had different data, switching to:")
		operate.LogUpload(consumer, upload, build)
	}

	var fullTargetPath string
	var strategy = LaunchStrategyUnknown
	var candidate *configurator.Candidate
	var manifestAction *buse.Action

	appManifest, err := manifest.Read(installFolder)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	pickManifestAction := func() error {
		var err error

		if appManifest == nil {
			consumer.Infof("No manifest found at (%s)", manifest.Path(installFolder))
			return nil
		}

		actions := manifest.ListActions(appManifest, runtime)

		if len(actions) == 0 {
			consumer.Warnf("Had manifest, but no actions available (for this platform at least)")
			return nil
		}

		if len(actions) == 1 {
			manifestAction = actions[0]
		} else {
			r, err := messages.PickManifestAction.Call(rc, &buse.PickManifestActionParams{
				Actions: actions,
			})
			if err != nil {
				return errors.Wrap(err, 0)
			}

			if r.Index < 0 {
				return &buse.ErrAborted{}
			}

			manifestAction = actions[r.Index]
		}

		if manifestAction == nil {
			consumer.Warnf("No manifest action picked")
			return nil
		}

		// is it a path?
		res, err := DetermineStrategy(runtime, installFolder, manifestAction)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		strategy = res.Strategy
		fullTargetPath = res.FullTargetPath
		candidate = res.Candidate
		return nil
	}
	err = pickManifestAction()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	pickFromVerdict := func() error {
		consumer.Infof("→ Using verdict: %s", verdict)

		switch len(verdict.Candidates) {
		case 0:
			return ErrNoCandidates
		case 1:
			candidate = verdict.Candidates[0]
		default:
			fakeActions := []*buse.Action{}
			for _, c := range verdict.Candidates {
				name := fmt.Sprintf("%s (%s)", c.Path, humanize.IBytes(uint64(c.Size)))
				fakeActions = append(fakeActions, &buse.Action{
					Name: name,
					Path: c.Path,
				})
			}

			r, err := messages.PickManifestAction.Call(rc, &buse.PickManifestActionParams{
				Actions: fakeActions,
			})
			if err != nil {
				return errors.Wrap(err, 0)
			}

			if r.Index < 0 {
				return &buse.ErrAborted{}
			}
			candidate = verdict.Candidates[r.Index]
		}

		fullPath := filepath.Join(installFolder, candidate.Path)
		_, err := os.Stat(fullPath)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		fullTargetPath = fullPath
		return nil
	}

	if fullTargetPath == "" {
		consumer.Infof("Switching to verdict!")

		if verdict == nil {
			consumer.Infof("No verdict, configuring now")

			newVerdict, err := manager.Configure(consumer, installFolder, runtime)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}
			verdict = newVerdict

			cave.SetVerdict(verdict)
			cave.Save(rc.DB())

			err = pickFromVerdict()
			if err != nil {
				if !errors.Is(err, ErrNoCandidates) {
					return nil, errors.Wrap(err, 0)
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

					newVerdict, err := manager.Configure(consumer, installFolder, runtime)
					if err != nil {
						return nil, errors.Wrap(err, 0)
					}
					verdict = newVerdict

					cave.SetVerdict(verdict)
					cave.Save(rc.DB())

					err = pickFromVerdict()
					if err != nil {
						return nil, errors.Wrap(err, 0)
					}
				} else {
					return nil, errors.Wrap(err, 0)
				}
			}
		}
	}
	if upload != nil {
		switch upload.Type {
		case "soundtrack", "book", "video", "documentation", "mod", "audio_assets", "graphical_assets", "sourcecode":
			consumer.Infof("Forcing shell strategy because upload is of type (%s)", upload.Type)
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
			return nil, errors.Wrap(err, 0)
		}

		strategy = flavorToStrategy(candidate.Flavor)
	}

	consumer.Infof("→ Using strategy (%s)", strategy)
	consumer.Infof("  (%s) is our target", fullTargetPath)

	launcher := launchers[strategy]
	if launcher == nil {
		err := fmt.Errorf("no launcher for strategy (%s)", strategy)
		return nil, errors.Wrap(err, 0)
	}

	var args = []string{}
	var env = make(map[string]string)

	if manifestAction != nil {
		args = append(args, manifestAction.Args...)

		if manifestAction.Scope != "" {
			const onlyPermittedScope = "profile:me"
			if manifestAction.Scope != onlyPermittedScope {
				err := fmt.Errorf("Game asked for scope (%s), asking for permission is unimplemented for now", manifestAction.Scope)
				return nil, errors.Wrap(err, 0)
			}

			client, err := operate.ClientFromCredentials(credentials)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}

			res, err := client.Subkey(&itchio.SubkeyParams{
				GameID: game.ID,
				Scope:  manifestAction.Scope,
			})
			if err != nil {
				return nil, errors.Wrap(err, 0)
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
		RequestContext: rc,

		FullTargetPath: fullTargetPath,
		Candidate:      candidate,
		AppManifest:    appManifest,
		Action:         manifestAction,
		Sandbox:        sandbox,
		Args:           args,
		Env:            env,

		PrereqsDir:    params.PrereqsDir,
		Credentials:   credentials,
		InstallFolder: installFolder,
		Runtime:       runtime,

		RecordPlayTime: func(playTime time.Duration) error {
			defer func() {
				if e := recover(); e != nil {
					consumer.Warnf("Could not record play time: %s", e)
				}
			}()

			cave.RecordPlayTime(playTime)
			cave.Save(rc.DB())
			return nil
		},
	}

	cave.Touch()
	cave.Save(rc.DB())

	err = launcher.Do(launcherParams)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &buse.LaunchResult{}, nil
}
