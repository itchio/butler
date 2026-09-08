package steamsync

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/itchio/butler/cmd/push"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/butler/steam"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/headway/united"
	"github.com/pkg/errors"
)

var syncArgs = struct {
	appID    uint32
	target   string
	branch   string
	password string
	mappings []string
	skips    []uint32
	dryRun   bool
	cacheDir string
	force    bool
	noPush   bool
	hidden   bool
	config   string
}{}

func RegisterSync(ctx *mansion.Context) {
	cmd := ctx.App.Command("steam-sync", "Copy a Steam app's builds to itch.io, one channel per platform. Give an app id and target, or --from-config to run every entry in a file.").Hidden()
	cmd.Arg("appid", "Steam app id").Uint32Var(&syncArgs.appID)
	cmd.Arg("target", "itch.io project, for example 'leafo/x-moon'. Channel names are chosen per platform, use --map to override.").StringVar(&syncArgs.target)
	cmd.Flag("branch", "Steam branch to sync (default public)").StringVar(&syncArgs.branch)
	cmd.Flag("password", "Password for a private Steam branch. Also read from "+envBranchPassword+".").StringVar(&syncArgs.password)
	cmd.Flag("map", "Send a depot to a specific channel, as DEPOTID=CHANNEL. Repeatable.").StringsVar(&syncArgs.mappings)
	cmd.Flag("skip", "Leave a depot out. Repeatable.").Uint32ListVar(&syncArgs.skips)
	cmd.Flag("dry-run", "Show the plan without downloading or pushing anything").BoolVar(&syncArgs.dryRun)
	cmd.Flag("cache-dir", "Keep downloaded depots here between syncs so the next one only fetches what changed on Steam. Without it everything is downloaded into a temporary directory and removed once the push is done.").StringVar(&syncArgs.cacheDir)
	cmd.Flag("force", "Push even when the channel's latest build already has this Steam build id").BoolVar(&syncArgs.force)
	cmd.Flag("no-push", "Download and assemble the channel directories, then stop. Requires a cache directory, otherwise there would be nothing left to look at.").BoolVar(&syncArgs.noPush)
	cmd.Flag("hidden", "When pushing to a new channel, mark it as hidden so it's not immediately downloadable").BoolVar(&syncArgs.hidden)
	cmd.Flag("from-config", "Run every entry of this sync config file instead of taking an app id and target").StringVar(&syncArgs.config)
	registerCredFlags(cmd)
	ctx.Register(cmd, doSync)
}

const envBranchPassword = "BUTLER_STEAM_BRANCH_PASSWORD"

func doSync(ctx *mansion.Context) {
	ctx.Must(Sync(ctx))
}

func Sync(ctx *mansion.Context) error {
	goCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	entries, err := resolveEntries(syncRequest{
		AppID:    syncArgs.appID,
		Target:   syncArgs.target,
		Branch:   syncArgs.branch,
		Mappings: syncArgs.mappings,
		Skips:    syncArgs.skips,
		CacheDir: syncArgs.cacheDir,
		Hidden:   syncArgs.hidden,
		Config:   syncArgs.config,
		NoPush:   syncArgs.noPush && !syncArgs.dryRun,
	})
	if err != nil {
		return err
	}

	warnUngated()
	var failed []string
	for _, e := range entries {
		if len(entries) > 1 {
			comm.Opf("Syncing app %d to %s", e.App, e.Target)
		}
		if err := runEntry(ctx, goCtx, e); err != nil {
			if len(entries) == 1 {
				return hint(err)
			}
			comm.Warnf("app %d to %s: %v", e.App, e.Target, hint(err))
			failed = append(failed, fmt.Sprintf("%d", e.App))
		}
	}
	if len(failed) > 0 {
		return errors.Errorf("%d of %d entries failed: app %s", len(failed), len(entries), strings.Join(failed, ", "))
	}
	return nil
}

type syncRequest struct {
	AppID    uint32
	Target   string
	Branch   string
	Mappings []string
	Skips    []uint32
	CacheDir string
	Hidden   bool
	Config   string
	// NoPush requires every entry to have a cache directory.
	NoPush bool
}

// The command line and the config file are never combined, so per-app
// flags alongside --from-config are an error rather than ignored.
func resolveEntries(req syncRequest) ([]steam.SyncEntry, error) {
	var raw []steam.SyncEntry
	var cfg *steam.SyncConfig
	if req.AppID == 0 && req.Config == "" {
		return nil, errors.New("give an app id and target, or --from-config FILE")
	}
	if req.AppID != 0 && req.Config != "" {
		return nil, errors.New("--from-config cannot be combined with an app id")
	}
	if req.AppID == 0 && (req.Target != "" || req.Branch != "" || len(req.Mappings) > 0 || len(req.Skips) > 0 || req.CacheDir != "" || req.Hidden) {
		return nil, errors.New("per-app flags need an app id; with --from-config they come from the file")
	}
	if req.AppID != 0 {
		entry := steam.SyncEntry{
			App:      req.AppID,
			Target:   req.Target,
			Branch:   req.Branch, // empty means public, see BranchOrDefault
			Hidden:   req.Hidden,
			Skip:     req.Skips,
			CacheDir: req.CacheDir,
		}
		if len(req.Mappings) > 0 {
			entry.Map = map[string]string{}
			for _, m := range req.Mappings {
				id, channel, ok := strings.Cut(m, "=")
				if _, err := strconv.ParseUint(id, 10, 32); !ok || err != nil || channel == "" {
					return nil, errors.Errorf("bad --map value '%s', want DEPOTID=CHANNEL", m)
				}
				entry.Map[id] = channel
			}
		}
		if entry.Target == "" {
			return nil, errors.New("an itch.io target is required, for example 'leafo/x-moon'")
		}
		cfg = &steam.SyncConfig{}
		raw = []steam.SyncEntry{entry}
	} else {
		var err error
		cfg, err = steam.LoadSyncConfig(req.Config)
		if err != nil {
			return nil, err
		}
		if len(cfg.Sync) == 0 {
			return nil, errors.Errorf("%s has no sync entries", req.Config)
		}
		raw = cfg.Sync
	}

	entries := make([]steam.SyncEntry, 0, len(raw))
	for _, e := range raw {
		e, err := validateEntry(e)
		if err != nil {
			return nil, errors.Wrapf(err, "app %d", e.App)
		}
		e = cfg.Resolve(e)
		if req.NoPush && e.CacheDir == "" {
			return nil, errors.Errorf("--no-push needs a cache directory for app %d, since the temporary directory is removed when the command exits", e.App)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func validateEntry(e steam.SyncEntry) (steam.SyncEntry, error) {
	if err := e.Validate(); err != nil {
		return e, err
	}
	spec, err := itchio.ParseSpec(e.Target)
	if err != nil {
		return e, errors.Wrapf(err, "parsing target '%s'", e.Target)
	}
	if spec.Channel != "" {
		return e, errors.Errorf("target '%s' names a channel, but channels are chosen per platform. Use --map DEPOTID=%s to send a depot there.", e.Target, spec.Channel)
	}
	e.Target = spec.Target
	return e, nil
}

func runEntry(ctx *mansion.Context, goCtx context.Context, entry steam.SyncEntry) error {
	password := syncArgs.password
	if password == "" {
		password = os.Getenv(envBranchPassword)
	}
	planOpts, err := entry.PlanOptions(password)
	if err != nil {
		return err
	}

	if syncArgs.dryRun {
		comm.Opf("Fetching Steam app info for %d", entry.App)
		plan, err := steam.Plan(goCtx, store(ctx), planOpts)
		if err != nil {
			return err
		}
		comm.ResultOrPrint(plan, func() { printPlan(plan) })
		return nil
	}

	opts := steam.SyncOptions{
		PlanOptions: planOpts,
		CacheDir:    entry.CacheDir,
		Force:       syncArgs.force,
		Hidden:      entry.Hidden,
		Events:      &cliEvents{},
		Logf:        comm.Debugf,
	}
	if !syncArgs.noPush {
		// Authenticate with itch.io before downloading anything so a bad
		// target fails in seconds rather than after gigabytes.
		client, err := ctx.AuthenticateViaOauth()
		if err != nil {
			return errors.Wrap(err, "authenticating with itch.io")
		}
		opts.Client = client
		opts.Push = func(goCtx context.Context, dir, target, userVersion string, hidden bool) error {
			if comm.JsonEnabled() {
				comm.Object("steamSyncPushStart", comm.JsonMessage{"target": target, "channel": channelOf(target)})
			} else {
				comm.Opf("Pushing %s", target)
			}
			return push.Do(ctx, dir, target, userVersion, true, false, false, true, false, hidden)
		}
	}

	comm.Opf("Fetching Steam app info for %d", entry.App)
	result, err := steam.Sync(goCtx, store(ctx), opts)
	if err != nil {
		return err
	}
	pushed := 0
	channels := make([]comm.JsonMessage, 0, len(result.Channels))
	for _, c := range result.Channels {
		channels = append(channels, comm.JsonMessage{"channel": c.Name, "upToDate": c.UpToDate, "dir": c.Dir})
		if c.UpToDate {
			continue
		}
		pushed++
		if syncArgs.noPush {
			comm.Statf("%s:%s ready at %s", result.Plan.Target, c.Name, c.Dir)
		}
	}
	if pushed == 0 {
		comm.Statf("Everything is up to date.")
	}
	// The per-push "result" events above belong to push.Do, so the sync's
	// own summary goes out under a name of its own.
	comm.Object("steamSyncDone", comm.JsonMessage{
		"appId":    result.Plan.AppID,
		"buildId":  result.Plan.BuildID,
		"target":   result.Plan.Target,
		"channels": channels,
	})
	return nil
}

func channelOf(target string) string {
	if i := strings.LastIndex(target, ":"); i >= 0 {
		return target[i+1:]
	}
	return ""
}

// cliEvents renders sync progress the way the rest of butler does. With
// --json it emits one typed event per step instead, which is what the
// butlerd worker path reads.
type cliEvents struct {
	lastProgress time.Time
}

func (cliEvents) Planned(plan *steam.SyncPlan) {
	if comm.JsonEnabled() {
		comm.Object("steamSyncPlan", comm.JsonMessage{"plan": plan})
		return
	}
	printPlan(plan)
}

func (cliEvents) ChannelUpToDate(channel string) {
	if comm.JsonEnabled() {
		comm.Object("steamSyncChannelUpToDate", comm.JsonMessage{"channel": channel})
		return
	}
	comm.Statf("%s already has this Steam build, skipping (use --force to push anyway)", channel)
}

func (cliEvents) DepotStart(dp *steam.DepotPlan, files int, totalBytes uint64) {
	if comm.JsonEnabled() {
		comm.Object("steamSyncDepotStart", comm.JsonMessage{"depotId": dp.ID, "files": files, "totalBytes": totalBytes})
		return
	}
	comm.Opf("Downloading depot %d (%d files)", dp.ID, files)
	comm.StartProgressWithTotalBytes(int64(totalBytes))
}

func (e *cliEvents) DepotProgress(dp *steam.DepotPlan, done, total uint64) {
	if comm.JsonEnabled() {
		// chunks land many times a second; the reader only needs a
		// few updates per second
		if time.Since(e.lastProgress) < 250*time.Millisecond && done < total {
			return
		}
		e.lastProgress = time.Now()
		comm.Object("steamSyncDepotProgress", comm.JsonMessage{"depotId": dp.ID, "doneBytes": done, "totalBytes": total})
		return
	}
	if total > 0 {
		comm.Progress(float64(done) / float64(total))
	}
}

func (cliEvents) DepotDone(dp *steam.DepotPlan, st steam.DepotStats) {
	if comm.JsonEnabled() {
		comm.Object("steamSyncDepotDone", comm.JsonMessage{"depotId": dp.ID, "fetchedBytes": st.Fetched, "reusedBytes": st.Reused, "skippedBytes": st.Skipped})
		return
	}
	comm.EndProgress()
	comm.Statf("Depot %d: %s fetched, %s reused from previous files, %s unchanged",
		dp.ID, united.FormatBytes(int64(st.Fetched)), united.FormatBytes(int64(st.Reused)), united.FormatBytes(int64(st.Skipped)))
}

func (cliEvents) ChannelAssembled(channel, dir string, steamworks []string) {
	if comm.JsonEnabled() {
		comm.Object("steamSyncChannelAssembled", comm.JsonMessage{"channel": channel, "steamworksFiles": steamworks})
		return
	}
	comm.Opf("Assembled %s", channel)
	warnSteamworks(channel, steamworks)
}

func warnSteamworks(channel string, files []string) {
	if len(files) == 0 {
		return
	}
	lines := []string{
		fmt.Sprintf("The %s build ships the Steamworks SDK:", channel),
		"",
	}
	for _, f := range files {
		lines = append(lines, "  "+f)
	}
	lines = append(lines, "",
		"If the game initializes Steam at startup it may not run for itch.io players.",
		"Consider a build with Steam integration disabled for this channel.")
	comm.Notice("Steamworks SDK detected", lines)
}

func printPlan(p *steam.SyncPlan) {
	comm.Logf("")
	comm.Statf("%s (app %d), branch %s, build %d", p.AppName, p.AppID, p.Branch, p.BuildID)
	for _, c := range p.Channels {
		size, download := c.Size()
		comm.Logf("")
		comm.Logf("  %s:%s  (%s on disk, %s to download)", p.Target, c.Name, united.FormatBytes(int64(size)), united.FormatBytes(int64(download)))
		for _, d := range c.Depots {
			shared := ""
			if d.Shared {
				shared = "  [all platforms]"
			}
			comm.Logf("    depot %-10d %-30s %10s  manifest %d%s", d.ID, d.Name, united.FormatBytes(int64(d.Size)), d.GID, shared)
		}
	}
	if len(p.Skipped) > 0 {
		comm.Logf("")
		comm.Logf("  skipped:")
		for _, d := range p.Skipped {
			comm.Logf("    depot %-10d %-30s %s", d.ID, d.Name, d.Reason)
		}
	}
	for _, w := range p.Warnings {
		comm.Warnf("%s", w)
	}
	comm.Logf("")
	comm.Logf("Each channel is pushed with --userversion %s.", fmt.Sprint(p.BuildID))
}
