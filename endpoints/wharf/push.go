package wharf

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/shell/loggerwriter"
	"github.com/pkg/errors"
)

// pushResult mirrors the JSON `result` event emitted by `butler push --json`.
// Fields not relevant to a given run stay zero — the caller picks what it
// needs.
type pushResult struct {
	BuildID       int64                        `json:"buildId"`
	Channel       string                       `json:"channel"`
	Skipped       bool                         `json:"skipped"`
	HasParent     bool                         `json:"hasParent"`
	ParentBuildID int64                        `json:"parentBuildId"`
	Comparison    *butlerd.WharfPushComparison `json:"comparison,omitempty"`
}

// pushEvent is a discriminated union of every JSON message the butler push
// worker emits over stdout. Fields not relevant to a given Type stay zero.
type pushEvent struct {
	Type          string     `json:"type"`
	Progress      float64    `json:"progress"`
	ETA           float64    `json:"eta"`
	BPS           float64    `json:"bps"`
	ReadBytes     int64      `json:"readBytes"`
	TotalBytes    int64      `json:"totalBytes"`
	UploadedBytes int64      `json:"uploadedBytes"`
	PatchBytes    int64      `json:"patchBytes"`
	Level         string     `json:"level"`
	Message       string     `json:"message"`
	Value         pushResult `json:"value"`
}

// Push spawns a `butler push` worker subprocess and brokers its output as
// butlerd notifications. The worker is killed if the RPC's context is
// cancelled (via exec.CommandContext).
func Push(rc *butlerd.RequestContext, params butlerd.WharfPushParams) (*butlerd.WharfPushResult, error) {
	args := buildPushArgs(params)
	source := params.Source
	if source == "" {
		source = "butlerd"
	}
	result, err := runPushWorker(rc, params.ProfileID, args, source)
	if err != nil {
		return nil, err
	}

	channel := result.Channel
	if channel == "" {
		channel = params.Channel
	}
	return &butlerd.WharfPushResult{
		BuildID: result.BuildID,
		Channel: channel,
		Skipped: result.Skipped,
	}, nil
}

// runPushWorker handles the shared subprocess lifecycle for both
// Wharf.Push and Wharf.PushPreview: spawn `butler` with the given args,
// stream JSON events as butlerd notifications, return the final result
// event. pushSource is forwarded as BUTLER_PUSH_SOURCE so the worker can
// tag the originating client when calling the API; pass "" to skip
// (push-preview never reaches CreateBuild).
func runPushWorker(rc *butlerd.RequestContext, profileID int64, args []string, pushSource string) (*pushResult, error) {
	consumer := rc.Consumer

	profile, _ := rc.ProfileClient(profileID)

	selfPath, err := os.Executable()
	if err != nil {
		return nil, errors.Wrap(err, "resolving butler executable path")
	}

	consumer.Infof("Spawning butler push worker: %s %v", selfPath, args)

	cmd := exec.CommandContext(rc.Ctx, selfPath, args...)
	cmd.Env = append(os.Environ(), "BUTLER_API_KEY="+profile.APIKey)
	if pushSource != "" {
		cmd.Env = append(cmd.Env, "BUTLER_PUSH_SOURCE="+pushSource)
	}
	cmd.Stderr = loggerwriter.New(consumer, "err")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "opening stdout pipe")
	}

	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "starting butler push worker")
	}

	var result pushResult
	var lastErr string
	gotResult := false
	scanStdout(rc, stdout, &result, &gotResult, &lastErr)
	waitErr := cmd.Wait()

	if waitErr != nil {
		if lastErr != "" {
			return nil, errors.New(lastErr)
		}
		// rc.Ctx cancelled produces a "signal: killed" wait error; surface
		// the cancellation cause instead so the client sees a clean cancel.
		if rc.Ctx.Err() != nil {
			return nil, errors.Wrap(rc.Ctx.Err(), "push cancelled")
		}
		return nil, errors.Wrap(waitErr, "butler push worker failed")
	}
	if !gotResult {
		return nil, errors.New("butler push worker completed without emitting a result")
	}
	return &result, nil
}

// buildPushArgs only emits flags that diverge from butler's CLI defaults,
// so a zero-valued WharfPushParams produces the same behaviour as a bare
// `butler push <src> <target> --json`.
func buildPushArgs(p butlerd.WharfPushParams) []string {
	specStr := fmt.Sprintf("%s:%s", p.Target, p.Channel)
	args := []string{"push", p.Src, specStr, "--json"}

	if p.UserVersion != "" {
		args = append(args, "--userversion", p.UserVersion)
	}
	if p.Hidden {
		args = append(args, "--hidden")
	}
	if p.IfChanged {
		args = append(args, "--if-changed")
	}
	if p.Dereference {
		args = append(args, "--dereference")
	}
	if p.FixPermissions != nil {
		args = append(args, "--fix-permissions="+strconv.FormatBool(*p.FixPermissions))
	}
	if p.AutoWrap != nil {
		args = append(args, "--auto-wrap="+strconv.FormatBool(*p.AutoWrap))
	}
	return args
}

func scanStdout(rc *butlerd.RequestContext, stdout io.Reader, result *pushResult, gotResult *bool, lastErr *string) {
	consumer := rc.Consumer
	scanner := bufio.NewScanner(stdout)
	// 1 MB cap is plenty — butler events are short, but oversized lines
	// would otherwise stall the scanner.
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var ev pushEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			consumer.Debugf("non-JSON push output: %s", scanner.Text())
			continue
		}
		switch ev.Type {
		case "progress":
			_ = messages.WharfPushProgress.Notify(rc, butlerd.WharfPushProgressNotification{
				Progress:      ev.Progress,
				ETA:           ev.ETA,
				BPS:           ev.BPS,
				ReadBytes:     ev.ReadBytes,
				TotalBytes:    ev.TotalBytes,
				UploadedBytes: ev.UploadedBytes,
				PatchBytes:    ev.PatchBytes,
			})
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
				*lastErr = ev.Message
				consumer.Errorf("%s", ev.Message)
			}
		case "result":
			*result = ev.Value
			*gotResult = true
		}
	}
}
