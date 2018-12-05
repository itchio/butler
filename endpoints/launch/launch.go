package launch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/itchio/butler/butlerd/horror"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/hades"

	"github.com/itchio/httpkit/neterr"
	"github.com/itchio/httpkit/progress"
	"github.com/itchio/ox"

	"github.com/itchio/pelican"

	goerrors "errors"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/endpoints/launch/manifest"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/butler/manager"
	"github.com/itchio/dash"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

var ErrCandidateDisappeared = goerrors.New("candidate disappeared from disk!")

func Register(router *butlerd.Router) {
	messages.Launch.Register(router, Launch)
}

func Launch(rc *butlerd.RequestContext, params butlerd.LaunchParams) (*butlerd.LaunchResult, error) {
	consumer := rc.Consumer

	cave := operate.ValidateCave(rc, params.CaveID)
	var installFolder string
	rc.WithConn(func(conn *sqlite.Conn) {
		installFolder = cave.GetInstallFolder(conn)
	})

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
	// these credentials will be used for prereqs, etc., we don't want
	// a game-specific download key here
	var access *operate.GameAccess
	rc.WithConn(func(conn *sqlite.Conn) {
		access = operate.AccessForGameID(conn, game.ID).OnlyAPIKey()
	})

	runtime := ox.CurrentRuntime()

	consumer.Infof("→ Launching %s", operate.GameToString(game))
	consumer.Infof("   on runtime %s", runtime)
	consumer.Infof("   (%s) is our install folder", installFolder)

	// attempt to refresh upload
	{
		client := rc.Client(access.APIKey)
		uploadRes, err := client.GetUpload(itchio.GetUploadParams{
			Credentials: access.Credentials,
			UploadID:    upload.ID,
		})
		if err != nil {
			consumer.Warnf("Could not refresh upload: %v", err)
		} else {
			upload = uploadRes.Upload
			rc.WithConn(func(conn *sqlite.Conn) {
				models.MustSave(conn, upload, hades.Assoc("Build"))
			})
			consumer.Debugf("Refreshed upload (last updated %s)", upload.UpdatedAt)
		}
	}

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

	err = ensureLicenseAcceptance(rc, installFolder)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var fullTargetPath string
	var strategy = LaunchStrategyUnknown
	var candidate *dash.Candidate
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
			consumer.Infof("Manifest with single action: %#v", manifestAction)
		} else {
			consumer.Infof("Manifest with %d actions, picking...", len(actions))
			r, err := messages.PickManifestAction.Call(rc, butlerd.PickManifestActionParams{
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
		res, err := DetermineStrategy(consumer, runtime, installFolder, manifestAction)
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

	filterSetupExes := func(candidatesIn []*dash.Candidate) []*dash.Candidate {
		var candidatesOut []*dash.Candidate
		for _, c := range candidatesIn {
			exclude := false

			switch c.Flavor {
			case dash.FlavorNativeWindows:
				{
					err := func() error {
						f, err := os.Open(filepath.Join(installFolder, c.Path))
						if err != nil {
							return errors.WithStack(err)
						}
						defer f.Close()

						peInfo, err := pelican.Probe(f, &pelican.ProbeParams{
							Consumer: consumer,
						})
						if err != nil {
							return errors.WithStack(err)
						}

						if peInfo.RequiresElevation() {
							exclude = true
							return nil
						}

						if peInfo.AssemblyInfo == nil && installer.HasSuspiciouslySetupLikeName(filepath.Base(c.Path)) {
							exclude = false
							return nil
						}

						return nil
					}()
					if err != nil {
						consumer.Warnf("Could not filter elevated exes: %+v", err)
						return candidatesIn
					}
				}
			}

			if !exclude {
				candidatesOut = append(candidatesOut, c)
			}
		}
		return candidatesOut
	}

	filterCandidates := func(candidatesIn []*dash.Candidate) []*dash.Candidate {
		if len(candidatesIn) <= 1 {
			return candidatesIn
		}

		var nativeFlavor dash.Flavor
		var nativeArch dash.Arch
		switch runtime.Platform {
		case ox.PlatformWindows:
			nativeFlavor = dash.FlavorNativeWindows
		case ox.PlatformLinux:
			nativeFlavor = dash.FlavorNativeLinux
		}
		if runtime.Is64 {
			nativeArch = dash.ArchAmd64
		} else {
			nativeArch = dash.Arch386
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

		var candidatesOut []*dash.Candidate
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

		for _, c := range verdict.Candidates {
			candidatePath := filepath.Join(installFolder, c.Path)
			_, err := os.Stat(candidatePath)
			if err != nil && os.IsNotExist(err) {
				consumer.Warnf("%s disappeared, forcing reconfigure", candidatePath)
				return errors.WithMessage(ErrCandidateDisappeared, "while picking from verdict")
			}
		}
		consumer.Statf("All launch targets still exist on disk")

		candidates := filterCandidates(verdict.Candidates)
		numCandidatesEliminated := len(verdict.Candidates) - len(candidates)
		if numCandidatesEliminated > 0 {
			consumer.Infof("Eliminated %d candidates via filtering", numCandidatesEliminated)
		}

		if len(candidates) >= 0 {
			nonSetupCandidates := filterSetupExes(verdict.Candidates)
			numCandidatesEliminated := len(candidates) - len(nonSetupCandidates)
			if numCandidatesEliminated > 0 {
				consumer.Infof("Eliminated %d candidates via setup filtering", numCandidatesEliminated)
				candidates = nonSetupCandidates
			}
		}

		switch len(candidates) {
		case 0:
			return errors.WithStack(butlerd.CodeNoLaunchCandidates)
		case 1:
			candidate = candidates[0]
		default:
			fakeActions := []*butlerd.Action{}
			for _, c := range candidates {
				name := fmt.Sprintf("%s (%s)", c.Path, progress.FormatBytes(c.Size))
				fakeActions = append(fakeActions, &butlerd.Action{
					Name: name,
					Path: c.Path,
				})
			}

			r, err := messages.PickManifestAction.Call(rc, butlerd.PickManifestActionParams{
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

	if upload != nil {
		switch upload.Type {
		case "soundtrack", "book", "video", "documentation", "mod", "audio_assets", "graphical_assets", "sourcecode":
			consumer.Infof("Forcing shell strategy because upload is of type (%s)", upload.Type)
			fullTargetPath = installFolder
			strategy = LaunchStrategyShell
		}
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
			rc.WithConn(cave.Save)

			err = pickFromVerdict()
			if err != nil {
				return nil, errors.WithStack(err)
			}
		} else {
			// pick from cached verdict
			err = pickFromVerdict()
			if err != nil {
				redoReason := ""
				if errors.Cause(err) == ErrCandidateDisappeared {
					redoReason = "Candidate disappeared!"
				} else if errors.Cause(err) == butlerd.CodeNoLaunchCandidates {
					redoReason = "No candidates!"
				}

				if redoReason != "" {
					consumer.Warnf("%s Re-configuring...", redoReason)

					newVerdict, err := manager.Configure(consumer, installFolder, runtime)
					if err != nil {
						return nil, errors.WithStack(err)
					}
					verdict = newVerdict
					consumer.Infof("→ New verdict: %s", verdict)

					cave.SetVerdict(verdict)
					rc.WithConn(cave.Save)

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

		err = requestAPIKeyIfNecessary(rc, manifestAction, game, access, env)
		if err != nil {
			return nil, errors.WithMessage(err, "While requesting API key")
		}
	}

	sandbox := params.Sandbox
	if manifestAction != nil && manifestAction.Sandbox {
		consumer.Infof("Enabling sandbox because of manifest opt-in")
		sandbox = true
	}

	crashed := false
	sessionWatcherDone := make(chan struct{})
	sessionStartedChan := make(chan struct{})
	var startSessionOnce sync.Once
	sessionEndedChan := make(chan struct{})

	sessionCtx, sessionCancel := context.WithCancel(rc.Ctx)
	defer sessionCancel()

	sessionWatcher := func() {
		defer close(sessionWatcherDone)
		defer horror.RecoverAndLog(consumer)

		lastRunAt := time.Now().UTC()
		sessionStartedAt := time.Now().UTC()
		var secondsRun int64 = 0

		conn := rc.GetConn()
		defer rc.PutConn(conn)
		access := operate.AccessForGameID(conn, cave.GameID)
		client := rc.Client(access.APIKey)

		var session *itchio.UserGameSession

		createSession := func() (retErr error) {
			defer horror.RecoverInto(&retErr)

			res, err := client.CreateUserGameSession(itchio.CreateUserGameSessionParams{
				GameID:       cave.GameID,
				UploadID:     cave.UploadID,
				BuildID:      cave.BuildID,
				Credentials:  access.Credentials,
				Platform:     interactionPlatform(runtime),
				Architecture: interactionArchitecture(runtime),

				SecondsRun: 0,
				LastRunAt:  &lastRunAt,
			})
			if err != nil {
				return errors.WithStack(err)
			}
			session = res.UserGameSession

			cave.UpdateInteractions(res.Summary)
			rc.WithConn(cave.Save)

			return
		}

		updateSession := func() (retErr error) {
			defer horror.RecoverInto(&retErr)

			lastRunAt = time.Now().UTC()
			secondsRun = int64(lastRunAt.Sub(sessionStartedAt).Seconds())
			res, err := client.UpdateUserGameSession(itchio.UpdateUserGameSessionParams{
				SessionID: session.ID,

				SecondsRun: secondsRun,
				LastRunAt:  &lastRunAt,
				Crashed:    crashed,
			})
			if err != nil {
				return errors.WithStack(err)
			}
			session = res.UserGameSession

			cave.UpdateInteractions(res.Summary)
			rc.WithConn(cave.Save)

			return
		}

		// At game launch, create a session
		err := createSession()
		if err != nil {
			consumer.Warnf("Initial session creation: %+v", err)
			return
		}

		// Then wait for session to actually start
		select {
		case <-sessionCtx.Done():
			consumer.Debugf("Launch cancelled while waiting for session to start, bailing out")
			return
		case <-sessionStartedChan:
			sessionStartedAt = time.Now().UTC()
			lastRunAt = time.Now().UTC()
		}

	regularUpdates:
		for {
			select {
			case <-sessionCtx.Done():
				consumer.Debugf("Launch cancelled while updating session regularly, bailing out")
				return
			case <-time.After(1 * time.Minute):
				err := updateSession()
				if err != nil {
					consumer.Warnf("Regular session update: %+v", err)
				}
			case <-sessionEndedChan:
				consumer.Debugf("Session ended normally!")
				break regularUpdates
			}
		}

		// Then, do a final session update for accurate stats
		err = updateSession()
		if err != nil {
			consumer.Warnf("Final session update: %+v", err)
			return
		}

		consumer.Debugf("Entire session committed successfully!")
	}

	go sessionWatcher()

	launcherParams := LauncherParams{
		RequestContext: rc,
		Ctx:            rc.Ctx,

		FullTargetPath: fullTargetPath,
		Candidate:      candidate,
		AppManifest:    appManifest,
		Action:         manifestAction,
		Sandbox:        sandbox,
		Args:           args,
		Env:            env,

		PrereqsDir:    params.PrereqsDir,
		ForcePrereqs:  params.ForcePrereqs,
		Access:        access,
		InstallFolder: installFolder,
		Runtime:       runtime,

		SessionStarted: func() {
			startSessionOnce.Do(func() {
				close(sessionStartedChan)
			})
		},
	}

	err = launcher.Do(launcherParams)
	close(sessionEndedChan)
	if err != nil {
		crashed = true
		return nil, errors.WithStack(err)
	}

	consumer.Debugf("Waiting on session watcher...")
	sessionCancel()
	select {
	case <-sessionWatcherDone:
		consumer.Debugf("Session watcher completed")
	case <-time.After(5 * time.Second):
		consumer.Warnf("Timed out waiting on session watcher")
	}

	return &butlerd.LaunchResult{}, nil
}

func requestAPIKeyIfNecessary(rc *butlerd.RequestContext, manifestAction *butlerd.Action, game *itchio.Game, access *operate.GameAccess, env map[string]string) error {
	consumer := rc.Consumer

	if manifestAction.Scope == "" {
		// nothing to do
		return nil
	}

	const onlyPermittedScope = "profile:me"
	if manifestAction.Scope != onlyPermittedScope {
		err := fmt.Errorf("Game asked for scope (%s), asking for permission is unimplemented for now", manifestAction.Scope)
		return errors.WithStack(err)
	}

	client := rc.Client(access.APIKey)

	res, err := client.Subkey(itchio.SubkeyParams{
		GameID: game.ID,
		Scope:  manifestAction.Scope,
	})
	if err != nil {
		if neterr.IsNetworkError(err) {
			consumer.Infof("No Internet connection, integration API won't be available")
			env["ITCHIO_OFFLINE_MODE"] = "1"
			return nil
		}
		return errors.WithStack(err)
	}

	consumer.Infof("Got subkey (%d chars, expires %s)", len(res.Key), res.ExpiresAt)
	env["ITCHIO_API_KEY"] = res.Key
	env["ITCHIO_API_KEY_EXPIRES_AT"] = res.ExpiresAt
	return nil
}

func interactionPlatform(runtime *ox.Runtime) itchio.SessionPlatform {
	switch runtime.Platform {
	case ox.PlatformLinux:
		return itchio.SessionPlatformLinux
	case ox.PlatformWindows:
		return itchio.SessionPlatformWindows
	case ox.PlatformOSX:
		return itchio.SessionPlatformMacOS
	}
	return itchio.SessionPlatform("")
}

func interactionArchitecture(runtime *ox.Runtime) itchio.SessionArchitecture {
	if runtime.Is64 {
		return itchio.SessionArchitectureAmd64
	}
	return itchio.SessionArchitecture386
}
