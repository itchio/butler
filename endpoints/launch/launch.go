package launch

import (
	"context"
	"fmt"
	"sync"
	"time"

	goerrors "errors"

	"github.com/pkg/errors"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/horror"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/endpoints/launch/manifest"

	"github.com/itchio/httpkit/neterr"
	"github.com/itchio/ox"

	itchio "github.com/itchio/go-itchio"
)

var ErrCandidateDisappeared = goerrors.New("candidate disappeared from disk!")

func Register(router *butlerd.Router) {
	messages.Launch.Register(router, Launch)
	messages.LaunchGetTargets.Register(router, LaunchGetTargets)
}

func LaunchGetTargets(rc *butlerd.RequestContext, params butlerd.LaunchGetTargetsParams) (*butlerd.LaunchGetTargetsResult, error) {
	var res *butlerd.LaunchGetTargetsResult

	err := withInstallFolderLock(withInstallFolderLockParams{
		rc:     rc,
		caveID: params.CaveID,
		reason: "Launch.GetTargets",
	}, func(info withInstallFolderInfo) error {
		var platforms = []ox.Platform{ox.CurrentRuntime().Platform}
		platforms = append(platforms, params.NonNativePlatforms...)

		targetRes, err := getTargets(rc, getTargetsParams{
			info,
			platforms,
		})
		if err != nil {
			return err
		}

		res = &butlerd.LaunchGetTargetsResult{
			Targets: targetRes.targets,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return res, nil
}

func Launch(rc *butlerd.RequestContext, params butlerd.LaunchParams) (*butlerd.LaunchResult, error) {
	consumer := rc.Consumer
	var res *butlerd.LaunchResult

	err := withInstallFolderLock(withInstallFolderLockParams{
		rc:     rc,
		caveID: params.CaveID,
		reason: "Launch",
	}, func(info withInstallFolderInfo) error {
		cave := info.cave
		installFolder := info.installFolder
		access := info.access
		runtime := info.runtime

		game := cave.Game

		consumer.Infof("→ Launching %s", operate.GameToString(game))
		consumer.Infof("   on runtime %s", runtime)
		consumer.Infof("   (%s) is our install folder", installFolder)

		err := ensureLicenseAcceptance(rc, installFolder)
		if err != nil {
			return errors.WithStack(err)
		}

		targetRes, err := getTargets(rc, getTargetsParams{
			info:      info,
			platforms: []ox.Platform{runtime.Platform},
		})
		if err != nil {
			return err
		}
		targets := targetRes.targets

		var target *butlerd.LaunchTarget
		if len(targets) == 0 {
			return errors.WithStack(butlerd.CodeNoLaunchCandidates)
		} else if len(targets) == 1 {
			consumer.Infof("Single target, picking it:")
			target = targets[0]
			consumer.Logf("%s", target.Strategy.String())
		} else {
			consumer.Infof("Found (%d) targets, asking client to pick via PickManifestAction", len(targets))
			var actions []*manifest.Action
			for _, t := range targets {
				actions = append(actions, t.Action)
			}

			r, err := messages.PickManifestAction.Call(rc, butlerd.PickManifestActionParams{
				Actions: actions,
			})
			if err != nil {
				consumer.Warnf("PickManifestAction call failed")
				return errors.WithStack(err)
			}

			if r.Index < 0 {
				consumer.Warnf("PickManifestAction call aborted (Index < 0)")
				return errors.WithStack(butlerd.CodeOperationAborted)
			}

			target = targets[r.Index]
			consumer.Infof("Target picked:")
			consumer.Logf("%s", target.Strategy.String())
		}

		consumer.Infof("→ Using strategy (%s)", target.Strategy.Strategy)
		consumer.Infof("  (%s) is our target", target.Strategy.FullTargetPath)

		launcher := launchers[target.Strategy.Strategy]
		if launcher == nil {
			err := fmt.Errorf("no launcher for strategy (%s)", target.Strategy.Strategy)
			return errors.WithStack(err)
		}

		var args = []string{}
		var env = make(map[string]string)

		args = append(args, target.Action.Args...)

		err = requestAPIKeyIfNecessary(rc, target.Action, game, access, env)
		if err != nil {
			return errors.WithMessage(err, "While requesting API key")
		}

		sandbox := params.Sandbox
		if target.Action.Sandbox {
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

				res, err := client.CreateUserGameSession(rc.Ctx, itchio.CreateUserGameSessionParams{
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
				res, err := client.UpdateUserGameSession(rc.Ctx, itchio.UpdateUserGameSessionParams{
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

			FullTargetPath: target.Strategy.FullTargetPath,
			Candidate:      target.Strategy.Candidate,
			AppManifest:    targetRes.appManifest,
			Action:         target.Action,
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
			return errors.WithStack(err)
		}

		consumer.Debugf("Waiting on session watcher...")
		sessionCancel()
		select {
		case <-sessionWatcherDone:
			consumer.Debugf("Session watcher completed")
		case <-time.After(5 * time.Second):
			consumer.Warnf("Timed out waiting on session watcher")
		}

		res = &butlerd.LaunchResult{}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func requestAPIKeyIfNecessary(rc *butlerd.RequestContext, manifestAction *manifest.Action, game *itchio.Game, access *operate.GameAccess, env map[string]string) error {
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

	res, err := client.Subkey(rc.Ctx, itchio.SubkeyParams{
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

func interactionPlatform(runtime ox.Runtime) itchio.SessionPlatform {
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

func interactionArchitecture(runtime ox.Runtime) itchio.SessionArchitecture {
	if runtime.Is64 {
		return itchio.SessionArchitectureAmd64
	}
	return itchio.SessionArchitecture386
}
