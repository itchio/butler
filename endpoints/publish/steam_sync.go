package publish

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/shell/loggerwriter"
	"github.com/itchio/butler/steam"
	"github.com/pkg/errors"
)

func SteamSync(rc *butlerd.RequestContext, params butlerd.PublishSteamSyncSyncParams) (*butlerd.PublishSteamSyncSyncResult, error) {
	// Validated here so a bad map or target fails before the worker
	// starts, with the same message Plan gives.
	if _, err := steamPlanOptions(params.AppID, params.Target, params.Branch, params.Password, params.Map, params.Skip); err != nil {
		return nil, err
	}
	consumer := rc.Consumer

	profile, _ := rc.ProfileClient(params.ProfileID)

	ctx, cleanup := rc.MakeCancelable(params.ID)
	defer cleanup()

	selfPath, err := os.Executable()
	if err != nil {
		return nil, errors.Wrap(err, "resolving butler executable path")
	}

	store := steamStore(rc)
	cacheDir := filepath.Join(store.Dir, "steam-sync", "cache", strconv.FormatInt(params.AppID, 10))
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, errors.Wrapf(err, "creating %s", cacheDir)
	}

	args := []string{"steam-sync", strconv.FormatInt(params.AppID, 10), params.Target, "--json", "--cache-dir", cacheDir}
	if rc.Identity != "" {
		args = append(args, "--identity", rc.Identity)
	}
	if params.Branch != "" {
		args = append(args, "--branch", params.Branch)
	}
	for id, channel := range params.Map {
		args = append(args, "--map", id+"="+channel)
	}
	for _, id := range params.Skip {
		args = append(args, "--skip", strconv.FormatInt(id, 10))
	}
	if params.Force {
		args = append(args, "--force")
	}
	if params.Hidden {
		args = append(args, "--hidden")
	}

	consumer.Infof("Spawning butler steam-sync worker: %s %v", selfPath, args)

	cmd := exec.CommandContext(ctx, selfPath, args...)
	cmd.Env = append(os.Environ(), "BUTLER_API_KEY="+profile.APIKey)
	if params.Password != "" {
		cmd.Env = append(cmd.Env, "BUTLER_STEAM_BRANCH_PASSWORD="+params.Password)
	}
	cmd.Stderr = loggerwriter.New(consumer, "err")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "opening stdout pipe")
	}
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "starting butler steam-sync worker")
	}

	sc := &steamSyncScanner{rc: rc}
	sc.scan(stdout)
	waitErr := cmd.Wait()

	if waitErr != nil {
		if ctx.Err() != nil {
			return nil, butlerd.CodeOperationCancelled
		}
		if sc.lastErr != "" {
			return nil, mapSteamErr(steamErrFromMessage(sc.lastErr))
		}
		return nil, errors.Wrap(waitErr, "butler steam-sync worker failed")
	}
	if sc.done == nil {
		return nil, errors.New("butler steam-sync worker completed without a summary")
	}

	res := &butlerd.PublishSteamSyncSyncResult{
		BuildID:  sc.done.BuildID,
		Channels: []*butlerd.PublishSteamSyncSyncedChannel{},
	}
	for _, c := range sc.done.Channels {
		res.Channels = append(res.Channels, &butlerd.PublishSteamSyncSyncedChannel{
			Channel:  c.Channel,
			BuildID:  sc.builds[c.Channel],
			UpToDate: c.UpToDate,
		})
	}
	return res, nil
}

func SteamSyncCancel(rc *butlerd.RequestContext, params butlerd.PublishSteamSyncCancelParams) (*butlerd.PublishSteamSyncCancelResult, error) {
	return &butlerd.PublishSteamSyncCancelResult{DidCancel: rc.CancelFuncs.Call(params.ID)}, nil
}

// The worker reports errors as text, so the sentinel ones are matched
// back by their message to get their codes.
func steamErrFromMessage(msg string) error {
	for _, sentinel := range []error{steam.ErrNotLoggedIn, steam.ErrNoPublisherKey, steam.ErrPublisherKeyInvalid} {
		if strings.Contains(msg, sentinel.Error()) {
			return sentinel
		}
	}
	return errors.New(msg)
}

// steamSyncEvent is every JSON line the steam-sync worker writes to
// stdout, its own events and the ones of the pushes it runs. Fields not
// relevant to a Type stay zero.
type steamSyncEvent struct {
	Type    string          `json:"type"`
	Plan    *steam.SyncPlan `json:"plan"`
	Channel string          `json:"channel"`
	Target  string          `json:"target"`
	DepotID int64           `json:"depotId"`
	// Depot download counters
	DoneBytes int64 `json:"doneBytes"`
	// Push counters, see pushEvent
	BuildID       int64   `json:"buildId"`
	Progress      float64 `json:"progress"`
	ETA           float64 `json:"eta"`
	BPS           float64 `json:"bps"`
	ReadBytes     int64   `json:"readBytes"`
	TotalBytes    int64   `json:"totalBytes"`
	UploadedBytes int64   `json:"uploadedBytes"`
	PatchBytes    int64   `json:"patchBytes"`
	Level         string  `json:"level"`
	Message       string  `json:"message"`
	// steamSyncDone
	AppID    int64                  `json:"appId"`
	Channels []steamSyncDoneChannel `json:"channels"`
}

type steamSyncDoneChannel struct {
	Channel  string `json:"channel"`
	UpToDate bool   `json:"upToDate"`
}

type steamSyncScanner struct {
	rc *butlerd.RequestContext
	// channel whose push is running, so the generic progress events
	// that both the depot download and the push emit can be told apart
	pushing string
	builds  map[string]int64
	lastErr string
	done    *steamSyncEvent
}

func (s *steamSyncScanner) scan(stdout io.Reader) {
	consumer := s.rc.Consumer
	s.builds = map[string]int64{}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var ev steamSyncEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			consumer.Debugf("non-JSON steam-sync output: %s", scanner.Text())
			continue
		}
		switch ev.Type {
		case "steamSyncPlan":
			if ev.Plan != nil {
				_ = messages.PublishSteamSyncPlanned.Notify(s.rc, butlerd.PublishSteamSyncPlannedNotification{Plan: convertSteamPlan(ev.Plan)})
			}
		case "steamSyncDepotProgress":
			_ = messages.PublishSteamSyncDepotProgress.Notify(s.rc, butlerd.PublishSteamSyncDepotProgressNotification{
				DepotID:    ev.DepotID,
				DoneBytes:  ev.DoneBytes,
				TotalBytes: ev.TotalBytes,
			})
		case "steamSyncChannelUpToDate":
			_ = messages.PublishSteamSyncChannelUpToDate.Notify(s.rc, butlerd.PublishSteamSyncChannelUpToDateNotification{Channel: ev.Channel})
		case "steamSyncPushStart":
			s.pushing = ev.Channel
			_ = messages.PublishSteamSyncPushStarted.Notify(s.rc, butlerd.PublishSteamSyncPushStartedNotification{Channel: ev.Channel})
		case "buildCreated":
			s.builds[ev.Channel] = ev.BuildID
			_ = messages.PublishSteamSyncBuildAssigned.Notify(s.rc, butlerd.PublishSteamSyncBuildAssignedNotification{
				Channel: ev.Channel,
				BuildID: ev.BuildID,
			})
		case "buildFailed":
			_ = messages.PublishSteamSyncBuildFailed.Notify(s.rc, butlerd.PublishSteamSyncBuildFailedNotification{
				Channel: ev.Channel,
				BuildID: ev.BuildID,
				Message: ev.Message,
			})
		case "progress":
			if s.pushing == "" {
				continue
			}
			_ = messages.PublishSteamSyncPushProgress.Notify(s.rc, butlerd.PublishSteamSyncPushProgressNotification{
				Channel:       s.pushing,
				Progress:      ev.Progress,
				ETA:           ev.ETA,
				BPS:           ev.BPS,
				ReadBytes:     ev.ReadBytes,
				TotalBytes:    ev.TotalBytes,
				UploadedBytes: ev.UploadedBytes,
				PatchBytes:    ev.PatchBytes,
			})
		case "result":
			s.pushing = ""
		case "steamSyncDone":
			ev := ev
			s.done = &ev
		case "log":
			switch ev.Level {
			case "error":
				consumer.Errorf("%s", ev.Message)
			case "warn", "warning":
				consumer.Warnf("%s", ev.Message)
			case "debug":
				consumer.Debugf("%s", ev.Message)
			default:
				consumer.Infof("%s", ev.Message)
			}
		case "error":
			if ev.Message != "" {
				s.lastErr = ev.Message
				consumer.Errorf("%s", ev.Message)
			}
		}
	}
}
