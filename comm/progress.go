package comm

import (
	"fmt"
	"time"

	"github.com/itchio/headway/probar"
	"github.com/itchio/headway/tracker"
	"github.com/itchio/httpkit/timeout"
)

var globalBar probar.Bar
var globalTracker tracker.Tracker

var lastProgressAlpha = 0.0

const maxLabelLength = 40

// ProgressLabel sets the string printed next to the progress indicator
func ProgressLabel(label string) {
	if globalBar == nil {
		return
	}

	if len(label) > maxLabelLength {
		label = fmt.Sprintf("...%s", label[len(label)-(maxLabelLength-3):])
	}
	globalBar.SetPostfix(label)
}

// StartProgress begins a period in which progress is regularly printed
func StartProgress() {
	StartProgressWithTotalBytes(0)
}

// StartProgressWithTotalBytes begins a period in which progress is regularly printed,
// and bps (bytes per second) is estimated from the total size given
func StartProgressWithTotalBytes(totalBytes int64) {
	if globalTracker != nil {
		// Already in-progress
		return
	}

	trackerOpts := tracker.Opts{
		Value: lastProgressAlpha,
	}
	if totalBytes > 0 {
		trackerOpts.ByteAmount = &tracker.ByteAmount{Value: totalBytes}
	}
	globalTracker = tracker.New(trackerOpts)

	if settings.noProgress || settings.json {
		// don't build a bar
	} else {
		globalBar = probar.New(globalTracker, probar.Opts{})
	}
}

// PauseProgress temporarily stops printing the progress bar
func PauseProgress() {
	if globalTracker != nil {
		globalTracker.Pause()
	}
}

// ResumeProgress resumes printing the progress bar after PauseProgress was called
func ResumeProgress() {
	if globalTracker != nil {
		globalTracker.Resume()
	}
}

var lastJSONPrintTime time.Time
var maxJSONPrintDuration = 500 * time.Millisecond

// Progress sets the completion of a task whose progress is being printed
// It only has an effect if StartProgress was already called.
func Progress(alpha float64) {
	lastProgressAlpha = alpha

	if globalTracker == nil {
		return
	}

	globalTracker.SetProgress(alpha)

	if lastJSONPrintTime.IsZero() {
		lastJSONPrintTime = time.Now()
	}
	printDuration := time.Since(lastJSONPrintTime)

	if printDuration > maxJSONPrintDuration {
		lastJSONPrintTime = time.Now()
		eta := 0.0
		bps := 0.0
		stats := globalTracker.Stats()
		if stats != nil {
			if stats.TimeLeft() != nil {
				eta = stats.TimeLeft().Seconds()
			}
			if stats.BPS() != nil {
				bps = stats.BPS().Value
			} else {
				bps = timeout.GetBPS()
			}
		}

		send("progress", JsonMessage{
			"progress": alpha,
			"eta":      eta,
			"bps":      bps,
		})
	}
}

// ProgressScale sets the scale on which the progress bar is displayed. This can be useful
// when the progress value evolves in another interval than [0, 1]
func ProgressScale(scale float64) {
	if settings.quiet {
		return
	}

	if globalBar != nil {
		globalBar.SetScale(scale)
	}
}

// EndProgress stops refreshing the progress bar and erases it.
func EndProgress() {
	if globalTracker != nil {
		globalTracker.Finish()
		globalTracker = nil
	}
	globalBar = nil
}
