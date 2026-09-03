package steamsync

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	return errors.New("downloading and pushing is not implemented yet, use --dry-run")
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
