package steam

import (
	"context"
	"strconv"
	"time"

	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

// Logf receives chatter that is only interesting when debugging.
type Logf func(format string, args ...interface{})

type DepotStats struct {
	Fetched uint64
	Reused  uint64
	Skipped uint64
}

// SyncEvents is how a sync reports progress. Every method may be called
// from the goroutine running Sync; none may block for long.
type SyncEvents interface {
	// Planned is called once, before anything is downloaded.
	Planned(plan *SyncPlan)
	// ChannelUpToDate means the channel already carries this build id.
	ChannelUpToDate(channel string)
	DepotStart(dp *DepotPlan, files int, totalBytes uint64)
	DepotProgress(dp *DepotPlan, doneBytes, totalBytes uint64)
	DepotDone(dp *DepotPlan, stats DepotStats)
	// ChannelAssembled is called with the directory about to be pushed
	// and any Steamworks SDK files found in it.
	ChannelAssembled(channel, dir string, steamworks []string)
}

// PushFunc uploads dir as a build of target ("user/game:channel").
// metadata records which Steam build the files came from.
type PushFunc func(ctx context.Context, dir, target, userVersion string, hidden bool, metadata itchio.BuildMetadata) error

type SyncOptions struct {
	PlanOptions
	// Client is used to skip channels that already carry the build. nil
	// skips the check.
	Client *itchio.Client
	// Push does the upload. nil means download and assemble only, and
	// then CacheDir must be set or nothing would be left behind.
	Push PushFunc
	// CacheDir keeps depots between syncs. Empty means a temporary
	// directory that is removed when Sync returns.
	CacheDir string
	// Force pushes even when the channel already has this build.
	Force bool
	// Hidden marks new channels hidden on itch.io.
	Hidden bool
	Events SyncEvents
	Logf   Logf
}

type ChannelResult struct {
	Name string
	// Dir is where the channel was assembled. Gone after Sync returns
	// unless CacheDir was set.
	Dir string
	// UpToDate means nothing was pushed because the build was already there.
	UpToDate bool
}

type SyncResult struct {
	Plan     *SyncPlan
	Channels []ChannelResult
}

// Sync downloads the app's depots for the branch, assembles one directory
// per channel and pushes each one, with the Steam build id as the user
// version.
func Sync(ctx context.Context, s Store, opts SyncOptions) (*SyncResult, error) {
	if opts.Events == nil {
		opts.Events = NopEvents{}
	}
	if opts.Logf == nil {
		opts.Logf = func(string, ...interface{}) {}
	}
	if opts.Push == nil && opts.CacheDir == "" {
		return nil, errors.New("a sync without a push needs a cache directory, since the temporary directory is removed on return")
	}
	if err := s.CheckAppAccess(ctx, opts.AppID); err != nil {
		return nil, err
	}
	sess, err := s.OpenSession(ctx, opts.Logf)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	plan, err := planWith(ctx, sess, opts.PlanOptions)
	if err != nil {
		return nil, err
	}
	opts.Events.Planned(plan)
	result := &SyncResult{Plan: plan}

	var todo []*ChannelPlan
	for _, c := range plan.Channels {
		if opts.Client != nil && !opts.Force {
			synced, err := alreadySynced(ctx, opts.Client, plan, c, opts.Logf)
			if err != nil {
				return nil, err
			}
			if synced {
				opts.Events.ChannelUpToDate(c.Name)
				result.Channels = append(result.Channels, ChannelResult{Name: c.Name, UpToDate: true})
				continue
			}
		}
		todo = append(todo, c)
	}
	if len(todo) == 0 {
		return result, nil
	}

	st, cleanup, err := openStage(s, opts.CacheDir, opts.Logf)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	downloaded := map[uint32]bool{}
	for _, c := range todo {
		for _, dp := range c.Depots {
			if downloaded[dp.ID] {
				continue
			}
			if err := st.downloadDepot(ctx, sess, plan, dp, opts.Password, opts.Events, opts.Logf); err != nil {
				return nil, err
			}
			downloaded[dp.ID] = true
		}
	}

	for _, c := range todo {
		dir, err := st.assembleChannel(c)
		if err != nil {
			return nil, err
		}
		opts.Events.ChannelAssembled(c.Name, dir, SteamworksFiles(dir))
		result.Channels = append(result.Channels, ChannelResult{Name: c.Name, Dir: dir})
		if opts.Push == nil {
			continue
		}
		target := plan.Target + ":" + c.Name
		if err := opts.Push(ctx, dir, target, strconv.FormatUint(uint64(plan.BuildID), 10), opts.Hidden, buildMetadata(plan, c)); err != nil {
			return nil, errors.Wrapf(err, "pushing %s", target)
		}
	}
	return result, nil
}

// buildMetadata is stored with the itch.io build so it can be traced
// back to the Steam build it was copied from. The server validates the
// shape; gids are strings because they are uint64.
func buildMetadata(plan *SyncPlan, c *ChannelPlan) itchio.BuildMetadata {
	depots := make([]map[string]interface{}, 0, len(c.Depots))
	for _, dp := range c.Depots {
		depots = append(depots, map[string]interface{}{
			"id":  dp.ID,
			"gid": strconv.FormatUint(dp.GID, 10),
		})
	}
	return itchio.BuildMetadata{
		"steam": map[string]interface{}{
			"app_id":   plan.AppID,
			"build_id": plan.BuildID,
			"branch":   plan.Branch,
			"depots":   depots,
		},
	}
}

// alreadySynced reports whether the channel's newest build, processed
// or still pending, was pushed with this Steam build id as its user
// version.
func alreadySynced(ctx context.Context, client *itchio.Client, plan *SyncPlan, c *ChannelPlan, logf Logf) (bool, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	info, err := client.GetChannel(reqCtx, plan.Target, c.Name)
	if err != nil {
		// A channel that does not exist yet is the common first-sync case.
		logf("channel %s lookup: %v", c.Name, err)
		return false, nil
	}
	if info == nil || info.Channel == nil {
		return false, nil
	}
	want := strconv.FormatUint(uint64(plan.BuildID), 10)
	for _, b := range []*itchio.Build{info.Channel.Pending, info.Channel.Head} {
		if b != nil && b.UserVersion == want {
			return true, nil
		}
	}
	return false, nil
}

// NopEvents discards everything.
type NopEvents struct{}

func (NopEvents) Planned(*SyncPlan)                         {}
func (NopEvents) ChannelUpToDate(string)                    {}
func (NopEvents) DepotStart(*DepotPlan, int, uint64)        {}
func (NopEvents) DepotProgress(*DepotPlan, uint64, uint64)  {}
func (NopEvents) DepotDone(*DepotPlan, DepotStats)          {}
func (NopEvents) ChannelAssembled(string, string, []string) {}
