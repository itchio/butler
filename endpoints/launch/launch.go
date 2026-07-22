package launch

import (
	"context"
	"fmt"
	"sync"
	"time"

	goerrors "errors"

	"github.com/pkg/errors"

	"crawshaw.io/sqlite"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/horror"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/hush/manifest"

	"github.com/itchio/httpkit/neterr"
	"github.com/itchio/ox"

	itchio "github.com/itchio/go-itchio"
)

var ErrCandidateDisappeared = goerrors.New("candidate disappeared from disk!")

func Register(router *butlerd.Router) {
	messages.Launch.Register(router, Launch)
	messages.LaunchGetTargets.Register(router, GetTargets)
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
		consumer.Infof("   (%s) is our install folder", installFolder)

		err := ensureLicenseAcceptance(rc, installFolder)
		if err != nil {
			return errors.WithStack(err)
		}

		hosts, err := rc.HostEnumerator().Enumerate(rc.Consumer)
		if err != nil {
			return err
		}

		targetRes, err := getTargets(rc, getTargetsParams{
			info:  info,
			hosts: hosts,
		})
		if err != nil {
			return err
		}
		targets := targetRes.targets

		var target *butlerd.LaunchTarget
		if len(targets) == 0 {
			return errors.WithStack(butlerd.CodeNoLaunchCandidates)
		}

		if params.Target != "" {
			// an explicit per-launch target is an API contract: fail rather
			// than surprising a non-interactive caller with a picker callback
			target = findTarget(targets, params.Target)
			if target == nil {
				consumer.Errorf("Requested target (%s) matched none of the (%d) targets", params.Target, len(targets))
				return errors.WithStack(butlerd.CodeLaunchTargetNotFound)
			}
			consumer.Infof("Using requested target (%s):", params.Target)
			consumer.Logf("%s", target.Strategy.String())
		} else if preferred := settingsLaunchTarget(rc, cave); preferred != "" {
			// a persisted preference is best-effort: it may go stale when the
			// game updates, so fall back to normal selection instead of failing
			target = findTarget(targets, preferred)
			if target != nil {
				consumer.Infof("Using preferred target (%s):", preferred)
				consumer.Logf("%s", target.Strategy.String())
			} else {
				consumer.Warnf("Preferred target (%s) matched none of the (%d) targets, using normal selection", preferred, len(targets))
			}
		}

		if target != nil {
			// preferred target found above
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
		consumer.Infof("  target (%s)", target.Strategy.FullTargetPath)
		consumer.Infof("  host (%s)", target.Host)

		launcher := launchers[target.Strategy.Strategy]
		if launcher == nil {
			err := fmt.Errorf("no launcher for strategy (%s)", target.Strategy.Strategy)
			return errors.WithStack(err)
		}

		var workingDirectory = ""
		var args = []string{}
		var env = make(map[string]string)
		env["ITCHIO_APP"] = "1"

		args = append(args, target.Action.Args...)
		fullTargetPath := target.Strategy.FullTargetPath
		if params.CommandTemplate != "" && target.Strategy.Strategy != butlerd.LaunchStrategyNative {
			consumer.Warnf("Custom command template does not apply to %s launches", target.Strategy.Strategy)
		}

		err = requestAPIKeyIfNecessary(rc, target.Action, game, access, env)
		if err != nil {
			return errors.WithMessage(err, "While requesting API key")
		}

		sandbox := resolveSandbox(params.Sandbox, target.Action.Sandbox)
		if target.Action.Sandbox {
			if sandbox {
				consumer.Infof("Enabling sandbox because of manifest opt-in")
			} else {
				consumer.Infof("Ignoring manifest sandbox opt-in: sandbox explicitly disabled for this game")
			}
		}

		tracksSession := launcherTracksSession(launcher)

		var (
			sessionWatcherDone chan struct{}
			sessionStartedChan chan time.Time
			sessionEndedChan   chan sessionEnd
			sessionCancel      context.CancelFunc
			startSessionOnce   sync.Once
		)
		sessionStarted := func() {}

		if tracksSession {
			sessionWatcherDone = make(chan struct{})
			sessionStartedChan = make(chan time.Time, 1)
			sessionEndedChan = make(chan sessionEnd, 1)

			var sessionCtx context.Context
			sessionCtx, sessionCancel = context.WithCancel(rc.Ctx)
			defer sessionCancel()

			// Resolve credentials once, up front, with a short-lived conn: the
			// watcher must not hold a pool connection for the game's lifetime.
			var sessionAccess *operate.GameAccess
			rc.WithConn(func(conn *sqlite.Conn) {
				sessionAccess = operate.AccessForGameID(conn, cave.GameID)
			})

			tracker := &sessionTracker{
				consumer:     consumer,
				client:       rc.Client(sessionAccess.APIKey),
				gameID:       cave.GameID,
				uploadID:     cave.UploadID,
				buildID:      cave.BuildID,
				credentials:  sessionAccess.Credentials,
				platform:     interactionPlatform(runtime),
				architecture: interactionArchitecture(runtime),
				persistSummary: func(summary *itchio.UserGameInteractionsSummary) {
					rc.WithConn(func(conn *sqlite.Conn) {
						if err := models.SaveUserGameInteractionSummary(conn, sessionAccess.ProfileID, cave.GameID, summary); err != nil {
							consumer.Warnf("Could not persist interaction summary: %+v", err)
						}
					})
				},
			}

			go func() {
				defer close(sessionWatcherDone)
				defer horror.RecoverAndLog(consumer)
				tracker.run(sessionCtx, sessionStartedChan, sessionEndedChan)
			}()

			sessionStarted = func() {
				startSessionOnce.Do(func() {
					sessionStartedChan <- time.Now()
				})
			}
		}

		launcherParams := LauncherParams{
			RequestContext: rc,
			Ctx:            rc.Ctx,

			FullTargetPath:   fullTargetPath,
			Candidate:        target.Strategy.Candidate,
			AppManifest:      targetRes.appManifest,
			Action:           target.Action,
			Sandbox:          sandbox,
			SandboxOptions:   params.SandboxOptions,
			WorkingDirectory: workingDirectory,
			Args:             args,
			Env:              env,
			CommandTemplate:  params.CommandTemplate,

			PrereqsDir:    params.PrereqsDir,
			ForcePrereqs:  params.ForcePrereqs,
			Access:        access,
			InstallFolder: installFolder,
			Host:          target.Host,

			SessionStarted: sessionStarted,
		}

		err = launcher.Do(launcherParams)

		if tracksSession {
			sessionEndedChan <- sessionEnd{at: time.Now(), crashed: err != nil}

			// go-itchio's retry backoffs are not context-aware, so this is a soft bound.
			consumer.Debugf("Waiting on session watcher...")
			select {
			case <-sessionWatcherDone:
				consumer.Debugf("Session watcher completed")
			case <-time.After(finalSessionUpdateTimeout + sessionWatcherJoinGrace):
				consumer.Warnf("Timed out waiting on session watcher")
			}
			sessionCancel()
		}

		if err != nil {
			return err
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
	switch runtime.Arch() {
	case "amd64":
		return itchio.SessionArchitectureAmd64
	case "arm64":
		return itchio.SessionArchitectureArm64
	case "386":
		return itchio.SessionArchitecture386
	case "arm":
		return itchio.SessionArchitectureArm
	}
	return itchio.SessionArchitecture("")
}
