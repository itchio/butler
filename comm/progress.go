package comm

import (
	"fmt"
	"time"

	"github.com/itchio/httpkit/progress"
	"github.com/itchio/wharf/state"
)

var tracker *progress.Tracker

var lastProgressAlpha = 0.0

func ApplyTheme(bar *progress.Bar, th *state.ProgressTheme) {
	bar.BarStart = th.BarStart
	bar.BarEnd = th.BarEnd
	bar.Current = th.Current
	bar.CurrentN = th.Current
	bar.Empty = th.Empty
}

const maxLabelLength = 40

// ProgressLabel sets the string printed next to the progress indicator
func ProgressLabel(label string) {
	if tracker == nil {
		return
	}

	if len(label) > maxLabelLength {
		label = fmt.Sprintf("...%s", label[len(label)-(maxLabelLength-3):])
	}
	tracker.Bar().Postfix(label)
}

// StartProgress begins a period in which progress is regularly printed
func StartProgress() {
	StartProgressWithTotalBytes(0)
}

// StartProgressWithTotalBytes begins a period in which progress is regularly printed,
// and bps (bytes per second) is estimated from the total size given
func StartProgressWithTotalBytes(totalBytes int64) {
	if tracker != nil {
		// Already in-progress
		return
	}

	tracker = progress.NewTracker()
	bar := tracker.Bar()

	bar.ShowCounters = false
	bar.ShowFinalTime = false
	bar.TimeBoxWidth = 8
	bar.BarWidth = 20
	bar.SetMaxWidth(80)

	tracker.SetTotalBytes(totalBytes)
	tracker.SetProgress(lastProgressAlpha)

	if settings.noProgress || settings.json {
		// use bar for ETA, but don't print
		tracker.SetSilent(true)
	}

	ApplyTheme(bar, state.GetTheme())
	tracker.Start()
}

// PauseProgress temporarily stops printing the progress bar
func PauseProgress() {
	if tracker != nil {
		tracker.Pause()
	}
}

// ResumeProgress resumes printing the progress bar after PauseProgress was called
func ResumeProgress() {
	if tracker != nil {
		tracker.Resume()
	}
}

var lastJsonPrintTime time.Time
var maxJsonPrintDuration = 500 * time.Millisecond

// Progress sets the completion of a task whose progress is being printed
// It only has an effect if StartProgress was already called.
func Progress(alpha float64) {
	lastProgressAlpha = alpha

	if tracker == nil {
		return
	}

	tracker.SetProgress(alpha)

	if lastJsonPrintTime.IsZero() {
		lastJsonPrintTime = time.Now()
	}
	printDuration := time.Since(lastJsonPrintTime)

	if printDuration > maxJsonPrintDuration {
		lastJsonPrintTime = time.Now()
		send("progress", JsonMessage{
			"progress":   alpha,
			"percentage": alpha * 100.0,
			"eta":        tracker.ETA().Seconds(),
			"bps":        tracker.BPS(),
		})
	}
}

// ProgressScale sets the scale on which the progress bar is displayed. This can be useful
// when the progress value evolves in another interval than [0, 1]
func ProgressScale(scale float64) {
	if settings.quiet {
		return
	}

	if tracker != nil {
		tracker.Bar().SetScale(scale)
	}
}

// EndProgress stops refreshing the progress bar and erases it.
func EndProgress() {
	if tracker != nil {
		tracker.SetProgress(1.0)
		tracker.Finish()
		tracker = nil
	}
}
