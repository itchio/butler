package steamsync

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/itchio/fresh-steamer/appinfo"
	"github.com/itchio/fresh-steamer/session"
	"github.com/pkg/errors"
)

// Plan is everything a sync needs to know, computed before any bytes move.
// It is the shape that will eventually be stored on the itch.io channel so
// later syncs do not have to be told the app id and mapping again.
type Plan struct {
	AppID    uint32         `json:"app_id"`
	AppName  string         `json:"app_name"`
	Branch   string         `json:"branch"`
	BuildID  uint32         `json:"build_id"`
	Target   string         `json:"target"`
	Channels []*ChannelPlan `json:"channels"`
	Skipped  []SkippedDepot `json:"skipped,omitempty"`
	Warnings []string       `json:"warnings,omitempty"`
}

type ChannelPlan struct {
	Name   string       `json:"name"`
	OS     string       `json:"os,omitempty"`
	Arch   string       `json:"arch,omitempty"`
	Depots []*DepotPlan `json:"depots"`
}

type DepotPlan struct {
	ID       uint32 `json:"id"`
	Name     string `json:"name"`
	GID      uint64 `json:"manifest"`
	Size     uint64 `json:"size"`
	Download uint64 `json:"download"`
	// Shared depots have no platform of their own and are copied into
	// every platform channel.
	Shared bool `json:"shared,omitempty"`
}

type SkippedDepot struct {
	ID     uint32 `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type PlanOptions struct {
	App      *appinfo.App
	Branch   string
	Password string
	Target   string
	// Map assigns a depot to a channel by hand, overriding platform detection.
	Map map[uint32]string
	// Skip leaves depots out entirely.
	Skip map[uint32]bool
}

func (c *ChannelPlan) Size() (size, download uint64) {
	for _, d := range c.Depots {
		size += d.Size
		download += d.Download
	}
	return
}

// itch.io infers a channel's platform from its name, so these are the
// names it recognizes. Steam calls macOS "macos"; itch.io wants "mac".
var channelOS = map[string]string{
	"windows": "windows",
	"linux":   "linux",
	"macos":   "mac",
}

// BuildPlan decides which depots go to which itch.io channel.
//
// Platform-specific depots create one channel per platform. When a
// platform has depots that declare an architecture, it gets one channel
// per architecture and its unarchitected depots are copied into each.
// Depots with no platform at all are copied into every channel. DLC,
// depots borrowed from another app, and non-English language packs are
// left out.
func BuildPlan(goCtx context.Context, s *session.Session, opts PlanOptions) (*Plan, error) {
	app := opts.App
	branch := app.Branch(opts.Branch)
	if branch == nil {
		var names []string
		for _, b := range app.Branches {
			names = append(names, b.Name)
		}
		return nil, errors.Errorf("app %d has no branch %q, available: %s", app.ID, opts.Branch, strings.Join(names, ", "))
	}

	plan := &Plan{
		AppID:   app.ID,
		AppName: app.Name,
		Branch:  branch.Name,
		BuildID: branch.BuildID,
		Target:  opts.Target,
	}

	type placed struct {
		depot *DepotPlan
		os    []string
	}
	var auto []placed
	channels := map[string]*ChannelPlan{}
	channel := func(name, os, arch string) *ChannelPlan {
		if c, ok := channels[name]; ok {
			return c
		}
		c := &ChannelPlan{Name: name, OS: os, Arch: arch}
		channels[name] = c
		return c
	}

	for _, d := range app.Depots {
		skip := func(reason string) {
			plan.Skipped = append(plan.Skipped, SkippedDepot{ID: d.ID, Name: d.Name, Reason: reason})
		}
		switch {
		case opts.Skip[d.ID]:
			skip("skipped by request")
			continue
		case d.DLCAppID != 0:
			skip(fmt.Sprintf("DLC for app %d", d.DLCAppID))
			continue
		case d.SharedFromApp != 0:
			skip(fmt.Sprintf("shared from app %d", d.SharedFromApp))
			continue
		case d.Language != "" && !strings.EqualFold(d.Language, "english"):
			skip(fmt.Sprintf("%s language pack", d.Language))
			continue
		}

		gid, err := s.ResolveManifest(goCtx, app, d, branch.Name, opts.Password)
		if err != nil {
			if _, encrypted := d.EncryptedManifests[branch.Name]; !encrypted && !hasManifest(d, branch.Name) {
				skip(fmt.Sprintf("not on branch %s", branch.Name))
				continue
			}
			return nil, errors.Wrapf(err, "depot %d", d.ID)
		}
		dp := &DepotPlan{ID: d.ID, Name: d.Name, GID: gid}
		if m := findManifest(d, branch.Name); m != nil {
			dp.Size = m.Size
			dp.Download = m.Download
		}

		if name, ok := opts.Map[d.ID]; ok {
			channel(name, "", "").Depots = append(channel(name, "", "").Depots, dp)
			continue
		}

		oslist := app.EffectiveOSList(d)
		if len(oslist) == 0 {
			dp.Shared = true
		}
		auto = append(auto, placed{depot: dp, os: oslist})
	}

	// Which architectures each platform declares. An empty string means a
	// depot without an arch, which is copied into every arch of that OS.
	arches := map[string]map[string]bool{}
	for _, p := range auto {
		for _, os := range p.os {
			if arches[os] == nil {
				arches[os] = map[string]bool{}
			}
			arches[os][archOf(app, p.depot.ID)] = true
		}
	}
	targetsFor := func(os string) []string {
		var out []string
		for arch := range arches[os] {
			if arch != "" {
				out = append(out, arch)
			}
		}
		if len(out) == 0 {
			out = []string{""}
		}
		sort.Strings(out)
		return out
	}

	var shared []*DepotPlan
	for _, p := range auto {
		if p.depot.Shared {
			shared = append(shared, p.depot)
			continue
		}
		for _, os := range p.os {
			base, ok := channelOS[strings.ToLower(os)]
			if !ok {
				plan.Warnings = append(plan.Warnings, fmt.Sprintf("depot %d targets unknown platform %q, channel name will need fixing", p.depot.ID, os))
				base = strings.ToLower(os)
			}
			depotArch := archOf(app, p.depot.ID)
			for _, arch := range targetsFor(os) {
				if depotArch != "" && depotArch != arch {
					continue
				}
				name := base
				if arch != "" {
					name += "-" + arch
				}
				c := channel(name, base, arch)
				c.Depots = append(c.Depots, p.depot)
			}
		}
	}

	if len(channels) == 0 && len(shared) > 0 {
		plan.Warnings = append(plan.Warnings, "no depot declares a platform, everything goes to a channel named \"all\". Use --map to pick real channel names.")
		channel("all", "", "")
	}
	for _, c := range channels {
		c.Depots = append(c.Depots, shared...)
	}

	for _, c := range channels {
		sort.Slice(c.Depots, func(i, j int) bool { return c.Depots[i].ID < c.Depots[j].ID })
		plan.Channels = append(plan.Channels, c)
	}
	sort.Slice(plan.Channels, func(i, j int) bool { return plan.Channels[i].Name < plan.Channels[j].Name })
	sort.Slice(plan.Skipped, func(i, j int) bool { return plan.Skipped[i].ID < plan.Skipped[j].ID })

	if len(plan.Channels) == 0 {
		return nil, errors.Errorf("nothing to sync: no depot of app %d is on branch %s", app.ID, branch.Name)
	}
	return plan, nil
}

func archOf(app *appinfo.App, depotID uint32) string {
	d := app.Depot(depotID)
	if d.OSArch != "" {
		return d.OSArch
	}
	return app.OSArch
}

func findManifest(d *appinfo.Depot, branch string) *appinfo.Manifest {
	for name, m := range d.Manifests {
		if strings.EqualFold(name, branch) {
			return m
		}
	}
	return nil
}

func hasManifest(d *appinfo.Depot, branch string) bool {
	return findManifest(d, branch) != nil
}
