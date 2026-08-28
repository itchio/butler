// Package launchcmd implements `butler launch`, a headless game runner.
//
// It drives butlerd's Launch endpoint in-process against the itch app's
// database, so an installed game can be started (prereqs, wine, sandbox,
// env, playtime tracking included) without booting the Electron app.
// Strategies that need a UI (html, shell, url) fail by default; with
// --fallback-to-app they hand the launch to the itch app via its itch://
// URL handler instead.
//
// Generated shortcuts (for launchers like Steam) should use --game with
// --fallback-to-app rather than --cave: the itch:// fallback URL cannot
// carry a cave, so --cave forfeits the fallback.
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
	"path/filepath"
	"runtime"
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

var args = struct {
	game                       *int64
	cave                       *string
	target                     *string
	profileID                  *int64
	prereqsDir                 *string
	fallbackToApp              *bool
	acceptLicenses             *bool
	continueAfterPrereqFailure *bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("launch", "Launch an installed game without the itch app").Hidden()
	args.game = cmd.Flag("game", "itch.io game ID to launch (picks the most recently used install)").Int64()
	args.cave = cmd.Flag("cave", "Cave ID to launch (takes precedence over --game)").String()
	args.target = cmd.Flag("target", "Launch target, matched against manifest action names then paths").String()
	args.profileID = cmd.Flag("profile-id", "itch.io user ID of the profile to launch as (default: resolve one with access to the game)").Int64()
	args.prereqsDir = cmd.Flag("prereqs-dir", "Directory to store prerequisite installers in").String()
	args.fallbackToApp = cmd.Flag("fallback-to-app", "Hand the launch to the itch app when it cannot run headlessly (detached: external launchers can no longer track the game process)").Bool()
	args.acceptLicenses = cmd.Flag("accept-licenses", "Accept any license agreement the game requires").Bool()
	args.continueAfterPrereqFailure = cmd.Flag("continue-after-prereq-failure", "Try to launch even when prerequisite installation fails").Bool()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx))
}

func Do(ctx *mansion.Context) error {
	if *args.cave == "" && *args.game == 0 {
		return errors.New("one of --game or --cave must be given")
	}

	if ctx.DBPath == "" {
		if p := defaultDBPath(); p != "" {
			comm.Logf("Using itch app database (%s)", p)
			ctx.DBPath = p
		}
	}
	ctx.EnsureDBPath()
	if _, err := os.Stat(ctx.DBPath); err != nil {
		return errors.Errorf("no butler database at (%s): install the itch app and log in first", ctx.DBPath)
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
		reason := fmt.Sprintf("database schema version (%d) doesn't match this butler (%d)",
			dbVersion, migrations.LatestSchemaVersion())
		if *args.game != 0 {
			return fallback(*args.game, 0, reason)
		}
		return errors.Errorf("%s: launch through the itch app instead", reason)
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
		return fallback(cave.GameID, cave.UploadID, "no itch.io profile found in the database")
	}

	prereqsDir := *args.prereqsDir
	if prereqsDir == "" {
		// the app's layout is <userData>/db/butler.db and <userData>/prereqs
		prereqsDir = filepath.Join(filepath.Dir(filepath.Dir(ctx.DBPath)), "prereqs")
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
		CaveID:     cave.ID,
		PrereqsDir: prereqsDir,
		Target:     *args.target,
		ProfileID:  *args.profileID,
		// enforced server-side under the launch lock, before any launcher or
		// session side effects
		AllowedStrategies: []butlerd.LaunchStrategy{butlerd.LaunchStrategyNative},
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
			return fallback(cave.GameID, cave.UploadID, reason)
		}
		var rpcErr *jsonrpc2.Error
		if goerrors.As(err, &rpcErr) &&
			rpcErr.Code == butlerd.CodeLaunchStrategyNotAllowed.RpcErrorCode() {
			return fallback(cave.GameID, cave.UploadID, "the game's launch strategy needs the app")
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

// defaultDBPath returns the itch app's database location for this
// platform (under Electron's userData dir), or "" if undeterminable.
func defaultDBPath() string {
	var base string
	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("APPDATA")
	case "darwin":
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, "Library", "Application Support")
		}
	default:
		base = os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			if home, err := os.UserHomeDir(); err == nil {
				base = filepath.Join(home, ".config")
			}
		}
	}
	if base == "" {
		return ""
	}
	return filepath.Join(base, "itch", "db", "butler.db")
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

func fallback(gameID int64, uploadID int64, reason string) error {
	if !*args.fallbackToApp {
		return errors.Errorf("cannot launch headlessly (%s): pass --fallback-to-app to hand the launch to the itch app", reason)
	}
	// the itch:// URL can't carry a target, cave, or profile, so falling
	// back would silently ignore those explicit flags
	if *args.target != "" {
		return errors.Errorf("cannot launch headlessly (%s) and --target cannot be honored by the itch app", reason)
	}
	if *args.cave != "" {
		return errors.Errorf("cannot launch headlessly (%s) and --cave cannot be honored by the itch app", reason)
	}
	if *args.profileID != 0 {
		return errors.Errorf("cannot launch headlessly (%s) and --profile-id cannot be honored by the itch app", reason)
	}
	comm.Logf("Handing launch to the itch app: %s", reason)
	return openItchURL(gameID, uploadID)
}
