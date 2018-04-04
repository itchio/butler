package launch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	goerrors "errors"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/configurator"
	"github.com/itchio/butler/endpoints/launch/manifest"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/butler/manager"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

var ErrNoCandidates = goerrors.New("no candidates")
var ErrCandidateDisappeared = goerrors.New("candidate disappeared from disk!")

func Register(router *butlerd.Router) {
	messages.Launch.Register(router, Launch)
	messages.LaunchCancel.Register(router, LaunchCancel)
}

var launchCancelID = "Launch"

func LaunchCancel(rc *butlerd.RequestContext, params *butlerd.LaunchCancelParams) (*butlerd.LaunchCancelResult, error) {
	didCancel := rc.CancelFuncs.Call(launchCancelID)
	return &butlerd.LaunchCancelResult{
		DidCancel: didCancel,
	}, nil
}

func Launch(rc *butlerd.RequestContext, params *butlerd.LaunchParams) (*butlerd.LaunchResult, error) {
	consumer := rc.Consumer

	ctx, cancelFunc := context.WithCancel(rc.Ctx)

	rc.CancelFuncs.Add(launchCancelID, cancelFunc)
	defer rc.CancelFuncs.Remove(launchCancelID)

	cave := operate.ValidateCave(rc, params.CaveID)
	installFolder := cave.GetInstallFolder(rc.DB())

	_, err := os.Stat(installFolder)
	if err != nil && os.IsNotExist(err) {
		return nil, &butlerd.RpcError{
			Code:    int64(butlerd.CodeInstallFolderDisappeared),
			Message: fmt.Sprintf("Could not find install folder (%s)", installFolder),
		}
	}

	game := cave.Game
	upload := cave.Upload
	build := cave.Build
	verdict := cave.GetVerdict()
	credentials := operate.CredentialsForGameID(rc.DB(), game.ID)

	runtime := manager.CurrentRuntime()

	consumer.Infof("→ Launching %s", operate.GameToString(game))
	consumer.Infof("   on runtime %s", runtime)
	consumer.Infof("   (%s) is our install folder", installFolder)
	consumer.Infof("Passed:")
	operate.LogUpload(consumer, upload, build)

	receiptIn, err := bfs.ReadReceipt(installFolder)
	if err != nil {
		return nil, errors.WithStack(err)
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
	var manifestAction *butlerd.Action

	appManifest, err := manifest.Read(installFolder)
	if err != nil {
		return nil, errors.WithStack(err)
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
			r, err := messages.PickManifestAction.Call(rc, &butlerd.PickManifestActionParams{
				Actions: actions,
			})
			if err != nil {
				return errors.WithStack(err)
			}

			if r.Index < 0 {
				return errors.WithStack(butlerd.CodeOperationAborted)
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
			return errors.WithStack(err)
		}

		strategy = res.Strategy
		fullTargetPath = res.FullTargetPath
		candidate = res.Candidate
		return nil
	}
	err = pickManifestAction()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	filterCandidates := func(candidatesIn []*configurator.Candidate) []*configurator.Candidate {
		if len(candidatesIn) <= 1 {
			return candidatesIn
		}

		var nativeFlavor configurator.Flavor
		var nativeArch configurator.Arch
		switch runtime.Platform {
		case butlerd.ItchPlatformWindows:
			nativeFlavor = configurator.FlavorNativeWindows
		case butlerd.ItchPlatformLinux:
			nativeFlavor = configurator.FlavorNativeLinux
		}
		if runtime.Is64 {
			nativeArch = configurator.ArchAmd64
		} else {
			nativeArch = configurator.Arch386
		}

		for _, c := range candidatesIn {
			if c.Flavor != nativeFlavor {
				consumer.Infof("Not filtering candidates, we found non-native (%s) flavor", c.Flavor)
				return candidatesIn
			}
		}

		hasNativeArch := false
		for _, c := range candidatesIn {
			if c.Arch == nativeArch {
				hasNativeArch = true
				break
			}
		}

		if !hasNativeArch {
			consumer.Infof("Not filtering candidates, none of them are native arch (%s)", nativeArch)
			return candidatesIn
		}

		var candidatesOut []*configurator.Candidate
		consumer.Infof("Filtering %d candidates by preferring native arch (%s)", len(candidatesIn), nativeArch)
		for _, c := range candidatesIn {
			if c.Arch == nativeArch {
				candidatesOut = append(candidatesOut, c)
			}
		}

		return candidatesOut
	}

	pickFromVerdict := func() error {
		consumer.Infof("→ Using verdict: %s", verdict)

		candidates := filterCandidates(verdict.Candidates)
		numCandidatesElimineated := len(verdict.Candidates) - len(candidates)
		if numCandidatesElimineated > 0 {
			consumer.Infof("Eliminated %d candidates via filtering", numCandidatesElimineated)
		}

		switch len(candidates) {
		case 0:
			return ErrNoCandidates
		case 1:
			candidate = candidates[0]
		default:
			fakeActions := []*butlerd.Action{}
			for _, c := range candidates {
				name := fmt.Sprintf("%s (%s)", c.Path, humanize.IBytes(uint64(c.Size)))
				fakeActions = append(fakeActions, &butlerd.Action{
					Name: name,
					Path: c.Path,
				})
			}

			r, err := messages.PickManifestAction.Call(rc, &butlerd.PickManifestActionParams{
				Actions: fakeActions,
			})
			if err != nil {
				return errors.WithStack(err)
			}

			if r.Index < 0 {
				return errors.WithStack(butlerd.CodeOperationAborted)
			}
			candidate = candidates[r.Index]
		}

		fullPath := filepath.Join(installFolder, candidate.Path)
		_, err := os.Stat(fullPath)
		if err != nil {
			return errors.WithStack(err)
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
				return nil, errors.WithStack(err)
			}
			verdict = newVerdict

			cave.SetVerdict(verdict)
			cave.Save(rc.DB())

			err = pickFromVerdict()
			if err != nil {
				if errors.Cause(err) == ErrNoCandidates {
					return nil, errors.WithStack(err)
				}
			}
		} else {
			// pick from cached verdict
			err = pickFromVerdict()
			if err != nil {
				redoReason := ""
				if errors.Cause(err) == ErrCandidateDisappeared {
					redoReason = "Candidate disappeared!"
				} else if errors.Cause(err) == ErrNoCandidates {
					redoReason = "No candidates!"
				}

				if redoReason != "" {
					consumer.Warnf("%s Re-configuring...", redoReason)

					newVerdict, err := manager.Configure(consumer, installFolder, runtime)
					if err != nil {
						return nil, errors.WithStack(err)
					}
					verdict = newVerdict

					cave.SetVerdict(verdict)
					cave.Save(rc.DB())

					err = pickFromVerdict()
					if err != nil {
						return nil, errors.WithStack(err)
					}
				} else {
					return nil, errors.WithStack(err)
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
			return nil, errors.WithStack(err)
		}

		strategy = flavorToStrategy(candidate.Flavor)
	}

	consumer.Infof("→ Using strategy (%s)", strategy)
	consumer.Infof("  (%s) is our target", fullTargetPath)

	launcher := launchers[strategy]
	if launcher == nil {
		err := fmt.Errorf("no launcher for strategy (%s)", strategy)
		return nil, errors.WithStack(err)
	}

	var args = []string{}
	var env = make(map[string]string)

	if manifestAction != nil {
		args = append(args, manifestAction.Args...)

		if manifestAction.Scope != "" {
			const onlyPermittedScope = "profile:me"
			if manifestAction.Scope != onlyPermittedScope {
				err := fmt.Errorf("Game asked for scope (%s), asking for permission is unimplemented for now", manifestAction.Scope)
				return nil, errors.WithStack(err)
			}

			client, err := operate.ClientFromCredentials(credentials)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			res, err := client.Subkey(&itchio.SubkeyParams{
				GameID: game.ID,
				Scope:  manifestAction.Scope,
			})
			if err != nil {
				return nil, errors.WithStack(err)
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
		Ctx:            ctx,

		FullTargetPath: fullTargetPath,
		Candidate:      candidate,
		AppManifest:    appManifest,
		Action:         manifestAction,
		Sandbox:        sandbox,
		Args:           args,
		Env:            env,

		PrereqsDir:    params.PrereqsDir,
		ForcePrereqs:  params.ForcePrereqs,
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
		return nil, errors.WithStack(err)
	}

	return &butlerd.LaunchResult{}, nil
}
