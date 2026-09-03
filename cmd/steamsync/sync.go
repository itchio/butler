package steamsync

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/itchio/butler/cmd/push"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
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
	noGate   bool
	dir      string
	force    bool
	noPush   bool
	hidden   bool
}{}

func RegisterSync(ctx *mansion.Context) {
	cmd := ctx.App.Command("steam-sync", "Copy a Steam app's builds to itch.io, one channel per platform.").Hidden()
	cmd.Arg("appid", "Steam app id").Required().Uint32Var(&syncArgs.appID)
	cmd.Arg("target", "itch.io project, for example 'leafo/x-moon'. Channel names are chosen per platform, use --map to override.").Required().StringVar(&syncArgs.target)
	cmd.Flag("branch", "Steam branch to sync").Default("public").StringVar(&syncArgs.branch)
	cmd.Flag("password", "Password for a private Steam branch").StringVar(&syncArgs.password)
	cmd.Flag("map", "Send a depot to a specific channel, as DEPOTID=CHANNEL. Repeatable.").StringsVar(&syncArgs.mappings)
	cmd.Flag("skip", "Leave a depot out. Repeatable.").Uint32ListVar(&syncArgs.skips)
	cmd.Flag("dry-run", "Show the plan without downloading or pushing anything").BoolVar(&syncArgs.dryRun)
	cmd.Flag("dir", "Where to keep downloaded depots between syncs. Defaults to steam-sync/APPID next to butler's credentials.").StringVar(&syncArgs.dir)
	cmd.Flag("force", "Push even when the channel's latest build already has this Steam build id").BoolVar(&syncArgs.force)
	cmd.Flag("no-push", "Download and assemble the channel directories, then stop").BoolVar(&syncArgs.noPush)
	cmd.Flag("hidden", "When pushing to a new channel, mark it as hidden so it's not immediately downloadable").BoolVar(&syncArgs.hidden)
	// Development only. Lets a dry run plan an app the publisher key does
	// not control. Never honored when bytes would actually move.
	cmd.Flag("no-gate", "").Hidden().BoolVar(&syncArgs.noGate)
	ctx.Register(cmd, doSync)
}

func doSync(ctx *mansion.Context) {
	ctx.Must(Sync(ctx))
}

func Sync(ctx *mansion.Context) error {
	goCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	spec, err := itchio.ParseSpec(syncArgs.target)
	if err != nil {
		return errors.Wrapf(err, "parsing target '%s'", syncArgs.target)
	}
	if spec.Channel != "" {
		return errors.Errorf("target '%s' names a channel, but channels are chosen per platform. Use --map DEPOTID=%s to send a depot there.", syncArgs.target, spec.Channel)
	}

	mapping := map[uint32]string{}
	for _, m := range syncArgs.mappings {
		id, channel, ok := strings.Cut(m, "=")
		depotID, err := strconv.ParseUint(id, 10, 32)
		if !ok || err != nil || channel == "" {
			return errors.Errorf("bad --map value '%s', want DEPOTID=CHANNEL", m)
		}
		mapping[uint32(depotID)] = channel
	}
	skip := map[uint32]bool{}
	for _, id := range syncArgs.skips {
		skip[id] = true
	}

	if !(syncArgs.dryRun && syncArgs.noGate) {
		if err := checkAppAccess(ctx, goCtx, syncArgs.appID); err != nil {
			return err
		}
	}

	s, err := openSession(ctx, goCtx)
	if err != nil {
		return err
	}
	defer s.Close()

	comm.Opf("Fetching Steam app info for %d", syncArgs.appID)
	app, err := s.AppInfo(goCtx, syncArgs.appID)
	if err != nil {
		return errors.Wrapf(err, "fetching app info for %d", syncArgs.appID)
	}

	plan, err := BuildPlan(goCtx, s, PlanOptions{
		App:      app,
		Branch:   syncArgs.branch,
		Password: syncArgs.password,
		Target:   spec.Target,
		Map:      mapping,
		Skip:     skip,
	})
	if err != nil {
		return err
	}

	comm.ResultOrPrint(plan, func() { printPlan(plan) })

	if syncArgs.dryRun {
		return nil
	}

	dir := syncArgs.dir
	if dir == "" {
		dir = defaultStageDir(ctx, plan.AppID)
	}
	st := &stage{Dir: dir}

	// Authenticate with itch.io before downloading anything so a bad
	// target fails in seconds rather than after gigabytes.
	var client *itchio.Client
	if !syncArgs.noPush {
		client, err = ctx.AuthenticateViaOauth()
		if err != nil {
			return errors.Wrap(err, "authenticating with itch.io")
		}
	}

	var todo []*ChannelPlan
	for _, c := range plan.Channels {
		if client != nil && !syncArgs.force {
			synced, err := alreadySynced(ctx, client, plan, c)
			if err != nil {
				return err
			}
			if synced {
				comm.Statf("%s:%s already has Steam build %d, skipping (use --force to push anyway)", plan.Target, c.Name, plan.BuildID)
				continue
			}
		}
		todo = append(todo, c)
	}
	if len(todo) == 0 {
		comm.Statf("Everything is up to date.")
		return nil
	}

	downloaded := map[uint32]bool{}
	for _, c := range todo {
		for _, dp := range c.Depots {
			if downloaded[dp.ID] {
				continue
			}
			if err := st.downloadDepot(goCtx, s, plan, dp, syncArgs.password); err != nil {
				return err
			}
			downloaded[dp.ID] = true
		}
	}

	for _, c := range todo {
		comm.Opf("Assembling %s from %d depot(s)", c.Name, len(c.Depots))
		chanDir, err := st.assembleChannel(c)
		if err != nil {
			return err
		}
		warnSteamworks(c.Name, steamworksFiles(chanDir))
		if syncArgs.noPush {
			comm.Statf("%s:%s ready at %s", plan.Target, c.Name, chanDir)
			continue
		}
		target := plan.Target + ":" + c.Name
		comm.Opf("Pushing %s", target)
		err = push.Do(ctx, chanDir, target, strconv.FormatUint(uint64(plan.BuildID), 10), true, false, false, true, false, syncArgs.hidden)
		if err != nil {
			return errors.Wrapf(err, "pushing %s", target)
		}
	}
	return nil
}

// alreadySynced reports whether the channel's newest build was pushed
// with this Steam build id as its user version.
func alreadySynced(ctx *mansion.Context, client *itchio.Client, plan *Plan, c *ChannelPlan) (bool, error) {
	reqCtx, cancel := ctx.DefaultCtx()
	defer cancel()
	info, err := client.GetChannel(reqCtx, plan.Target, c.Name)
	if err != nil {
		// A channel that does not exist yet is the common first-sync case.
		comm.Debugf("channel %s lookup: %v", c.Name, err)
		return false, nil
	}
	if info == nil || info.Channel == nil || info.Channel.Head == nil {
		return false, nil
	}
	return info.Channel.Head.UserVersion == strconv.FormatUint(uint64(plan.BuildID), 10), nil
}

func printPlan(p *Plan) {
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
