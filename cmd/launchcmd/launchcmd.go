// Package launchcmd implements `butler launch`, a headless game runner.
//
// It drives butlerd's Launch endpoint in-process against a butler
// database, so an installed game can be started (prereqs, wine, sandbox,
// env, playtime tracking included) without booting the itch app. The
// invoking frontend (itch-setup) supplies all app conventions explicitly:
// --dbpath, --prereqs-dir, and --default-* flags carrying its global
// preferences. Butler resolves precedence itself: explicit params, then
// per-cave settings, then those defaults, then the manifest.
//
// When a launch needs the app (html/shell/url strategy, an unaccepted
// license, a prereqs failure, no profile, schema mismatch), the command
// exits with needsAppExitCode and emits a "launch/needs-app" JSON line
// carrying the reason and game/upload IDs. The frontend decides whether
// to hand the launch to the app; butler never starts the app itself.
//
// A running itch app is deliberately not forwarded to: external
// launchers track their shortcut's process tree (and preloads like
// overlays ride on env inheritance), so the game must be this process's
// child. Sharing the app's database is safe: WAL handles concurrent
// access, and the install-folder runlock coordinates launches and
// installs across processes. A locked install folder makes a launch
// wait, so launching an already-running game queues until it exits.
//
// Lifetime tracking follows smaug's semantics (same as the app): the
// direct child, plus anything holding its inherited stdio pipes. A game
// that fully daemonizes escapes tracking.
package launchcmd

import (
	"context"
	goerrors "errors"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync/atomic"
	"syscall"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/horror"
	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/itchio/butler/cmd/daemon"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/database/models/migrations"
	"github.com/itchio/butler/mansion"
	"github.com/pkg/errors"
)

// Exit code signalling that the launch needs the itch app. Accompanied
// by a "launch/needs-app" JSON line with the reason and game/upload IDs.
const needsAppExitCode = 3

var args = struct {
	game                       *int64
	cave                       *string
	target                     *string
	profileID                  *int64
	prereqsDir                 *string
	acceptLicenses             *bool
	continueAfterPrereqFailure *bool
	defaultSandbox             *bool
	defaultSandboxType         *string
	defaultSandboxNoNetwork    *bool
	defaultSandboxAllowEnv     *[]string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("launch", "Launch an installed game without the itch app").Hidden()
	args.game = cmd.Flag("game", "itch.io game ID to launch (picks the most recently used install)").Int64()
	args.cave = cmd.Flag("cave", "Cave ID to launch (takes precedence over --game)").String()
	args.target = cmd.Flag("target", "Launch target, matched against manifest action names then paths").String()
	args.profileID = cmd.Flag("profile-id", "itch.io user ID of the profile to launch as (default: resolve one with access to the game)").Int64()
	args.prereqsDir = cmd.Flag("prereqs-dir", "Directory to store prerequisite installers in (without it, titles that require prerequisites fail)").String()
	args.acceptLicenses = cmd.Flag("accept-licenses", "Accept any license agreement the game requires").Bool()
	args.continueAfterPrereqFailure = cmd.Flag("continue-after-prereq-failure", "Try to launch even when prerequisite installation fails").Bool()
	args.defaultSandbox = cmd.Flag("default-sandbox", "Sandbox games unless their per-game setting says otherwise").Bool()
	args.defaultSandboxType = cmd.Flag("default-sandbox-type", "Default sandbox runner").Enum("auto", "bubblewrap", "firejail", "fuji")
	args.defaultSandboxNoNetwork = cmd.Flag("default-sandbox-no-network", "Cut network access inside the sandbox unless the per-game setting says otherwise").Bool()
	args.defaultSandboxAllowEnv = cmd.Flag("default-sandbox-allow-env", "Environment variable to allow through the sandbox (repeatable)").Strings()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	err := Do(ctx)
	var na *needsAppError
	if goerrors.As(err, &na) {
		comm.Object("launch/needs-app", map[string]interface{}{
			"reason":   na.reason,
			"gameId":   na.gameID,
			"uploadId": na.uploadID,
		})
		comm.Logf("The itch app is needed to launch this: %s", na.reason)
		os.Exit(needsAppExitCode)
	}
	ctx.Must(err)
}

// needsAppError marks a launch butler cannot serve headlessly; do()
// translates it into needsAppExitCode plus the launch/needs-app line.
type needsAppError struct {
	reason   string
	gameID   int64
	uploadID int64
}

func (e *needsAppError) Error() string {
	return "the itch app is needed: " + e.reason
}

func Do(ctx *mansion.Context) error {
	if *args.cave == "" && *args.game == 0 {
		return errors.New("one of --game or --cave must be given")
	}

	ctx.EnsureDBPath()
	if _, err := os.Stat(ctx.DBPath); err != nil {
		return errors.Errorf("no butler database at (%s)", ctx.DBPath)
	}

	// no SQLITE_OPEN_CREATE: this command must never produce an empty DB
	openFlags := sqlite.SQLITE_OPEN_READWRITE | sqlite.SQLITE_OPEN_WAL |
		sqlite.SQLITE_OPEN_URI | sqlite.SQLITE_OPEN_NOMUTEX
	dbPool, err := sqlitex.Open(ctx.DBPath, openFlags, 4)
	if err != nil {
		return errors.WithMessage(err, "opening database")
	}
	defer dbPool.Close()

	// Never database.Prepare here: hades AutoMigrate rebuilds any table
	// whose columns don't match this binary's models and drops columns it
	// doesn't know, so a version-skewed runner would destroy data written
	// by a newer butler. Only the app's own butlerd migrates. The schema
	// version check is a coarse guard (it only tracks hand-written
	// migrations, not model changes); the real protection is invoking the
	// same broth-managed butler the app uses.
	dbVersion, err := getSchemaVersion(dbPool)
	if err != nil {
		return errors.WithMessagef(err, "reading schema version from (%s) (not a butler database?)", ctx.DBPath)
	}
	if dbVersion != migrations.LatestSchemaVersion() {
		return &needsAppError{
			reason: fmt.Sprintf("database schema version (%d) doesn't match this butler (%d)",
				dbVersion, migrations.LatestSchemaVersion()),
			gameID: *args.game,
		}
	}

	cave, err := resolveCave(dbPool)
	if err != nil {
		return err
	}

	if *args.profileID != 0 {
		// validated here so a typo'd ID fails before the launch machinery
		// runs; the server's strict path never falls back to another profile
		if !profileExists(dbPool, *args.profileID) {
			return errors.Errorf("profile (%d) not found in the database", *args.profileID)
		}
	} else if !hasProfile(dbPool) {
		// launch machinery panics without a profile row; the app owns login
		return &needsAppError{
			reason:   "no itch.io profile found in the database",
			gameID:   cave.GameID,
			uploadID: cave.UploadID,
		}
	}

	comm.Logf("Launching cave (%s) for game (%d)", cave.ID, cave.GameID)

	router := daemon.GetRouter(dbPool, ctx)
	client := &headlessClient{
		acceptLicenses:             *args.acceptLicenses,
		continueAfterPrereqFailure: *args.continueAfterPrereqFailure,
	}

	launchCtx, stopLaunch := context.WithCancel(context.Background())
	defer stopLaunch()

	tracker := newLaunchTracker(router)

	// cancelling launchCtx cancels the Launch request's context and stops
	// the game's process group; it also tears down the conn (so the reply
	// is lost), which is why tracker watches the endpoint itself
	serverTransport, clientTransport := loopbackPipe()
	serverConn := jsonrpc2.NewConn(launchCtx, serverTransport, tracker)
	defer serverConn.Close()
	clientConn := jsonrpc2.NewConn(context.Background(), clientTransport, client)
	defer clientConn.Close()

	// tie the game's lifetime to ours: an external launcher (or Ctrl+C)
	// stopping butler should stop the game rather than orphan it, with the
	// play session still recorded on the way out
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigs)
	go func() {
		sig := <-sigs
		comm.Warnf("Got %s, stopping the game...", sig)
		stopLaunch()
		<-sigs
		comm.Warnf("Got a second signal, exiting immediately")
		os.Exit(1)
	}()

	params := butlerd.LaunchParams{
		CaveID:            cave.ID,
		PrereqsDir:        *args.prereqsDir,
		Target:            *args.target,
		ProfileID:         *args.profileID,
		AllowedStrategies: []butlerd.LaunchStrategy{butlerd.LaunchStrategyNative},
		Defaults:          launchDefaults(),
	}

	var res butlerd.LaunchResult
	err = clientConn.Call("Launch", params, &res)

	if launchCtx.Err() != nil {
		tracker.waitForTeardown()
		comm.Statf("Launch stopped")
		return nil
	}

	if err != nil {
		if reason := client.needsApp(); reason != "" {
			return &needsAppError{reason: reason, gameID: cave.GameID, uploadID: cave.UploadID}
		}
		var rpcErr *jsonrpc2.Error
		if goerrors.As(err, &rpcErr) &&
			rpcErr.Code == butlerd.CodeLaunchStrategyNotAllowed.RpcErrorCode() {
			return &needsAppError{
				reason:   "the game's launch strategy needs the app",
				gameID:   cave.GameID,
				uploadID: cave.UploadID,
			}
		}
		return err
	}

	comm.Statf("Game exited")
	return nil
}

func resolveCave(dbPool *sqlitex.Pool) (*models.Cave, error) {
	conn := dbPool.Get(context.Background())
	defer dbPool.Put(conn)

	if *args.cave != "" {
		cave := models.CaveByID(conn, *args.cave)
		if cave == nil {
			return nil, errors.Errorf("cave (%s) not found", *args.cave)
		}
		return cave, nil
	}

	caves := models.CavesByGameID(conn, *args.game)
	if len(caves) == 0 {
		return nil, errors.Errorf("game (%d) is not installed", *args.game)
	}

	sort.SliceStable(caves, func(i, j int) bool {
		return caveFreshness(caves[i]).After(caveFreshness(caves[j]))
	})
	return caves[0], nil
}

// caveFreshness matches the ordering the itch app uses when it has to
// pick one cave for a game.
func caveFreshness(c *models.Cave) time.Time {
	for _, t := range []*time.Time{c.LocalLastRunAt, c.LastTouchedAt, c.InstalledAt} {
		if t != nil {
			return *t
		}
	}
	return time.Time{}
}

// models.GetSchemaVersion panics on SQL errors (e.g. a sqlite file with
// no schema_version table); recover so a foreign DB yields a clean error.
// This runs before any other model query, so once it passes the rest of
// the Must-style model helpers are safe against missing tables.
func getSchemaVersion(dbPool *sqlitex.Pool) (version int64, retErr error) {
	defer horror.RecoverInto(&retErr)
	conn := dbPool.Get(context.Background())
	defer dbPool.Put(conn)
	return models.GetSchemaVersion(conn), nil
}

func hasProfile(dbPool *sqlitex.Pool) bool {
	conn := dbPool.Get(context.Background())
	defer dbPool.Put(conn)
	return len(models.AllProfiles(conn)) > 0
}

func profileExists(dbPool *sqlitex.Pool, id int64) bool {
	conn := dbPool.Get(context.Background())
	defer dbPool.Put(conn)
	return models.ProfileByID(conn, id) != nil
}

// launchDefaults builds the defaults tier from the --default-* flags, or
// nil when none were passed. Flags are presence-only, so an unset knob is
// absent (never false), leaving lower tiers free to apply.
func launchDefaults() *butlerd.LaunchDefaults {
	d := &butlerd.LaunchDefaults{}
	set := false
	if *args.defaultSandbox {
		t := true
		d.Sandbox = &t
		set = true
	}
	if *args.defaultSandboxType != "" {
		st := butlerd.SandboxType(*args.defaultSandboxType)
		d.SandboxType = &st
		set = true
	}
	if *args.defaultSandboxNoNetwork {
		t := true
		d.SandboxNoNetwork = &t
		set = true
	}
	if len(*args.defaultSandboxAllowEnv) > 0 {
		d.SandboxAllowEnv = *args.defaultSandboxAllowEnv
		set = true
	}
	if !set {
		return nil
	}
	return d
}

// launchTracker wraps the router to expose when the Launch request has
// finished: an interrupt tears down the conn before the reply, but the
// endpoint keeps running in-process to kill the game, record the play
// session, and release the install folder lock.
type launchTracker struct {
	jsonrpc2.Handler
	launchStarted atomic.Bool
	launchDone    chan struct{}
}

func newLaunchTracker(router jsonrpc2.Handler) *launchTracker {
	return &launchTracker{Handler: router, launchDone: make(chan struct{})}
}

func (t *launchTracker) HandleRequest(conn jsonrpc2.Conn, req jsonrpc2.Request) (interface{}, error) {
	if req.Method != "Launch" {
		return t.Handler.HandleRequest(conn, req)
	}
	t.launchStarted.Store(true)
	defer close(t.launchDone)
	return t.Handler.HandleRequest(conn, req)
}

func (t *launchTracker) waitForTeardown() {
	// the final session update alone can take up to ~40s
	timeout := 60 * time.Second
	if !t.launchStarted.Load() {
		// the Launch request may still sneak in right as we cancel; with a
		// cancelled context it fails within milliseconds, so a short wait
		// covers it without stalling the common did-not-start case
		timeout = 2 * time.Second
	}
	select {
	case <-t.launchDone:
	case <-time.After(timeout):
		comm.Warnf("Timed out waiting for the launch to clean up")
	}
}
