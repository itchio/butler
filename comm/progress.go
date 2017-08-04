package comm

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/itchio/butler/pb"
)

var bar *pb.ProgressBar
var lastProgressValue int64

// ProgressTheme contains all the characters we need to show progress
type ProgressTheme struct {
	BarStart        string
	BarEnd          string
	Current         string
	CurrentHalfTone string
	Empty           string
	OpSign          string
	StatSign        string
}

var themes = map[string]*ProgressTheme{
	"unicode": {"▐", "▌", "▓", "▒", "░", "•", "✓"},
	"ascii":   {"|", "|", "#", "=", "-", ">", "<"},
	"cp437":   {"▐", "▌", "█", "▒", "░", "∙", "√"},
}

func (th *ProgressTheme) apply(bar *pb.ProgressBar) {
	bar.BarStart = th.BarStart
	bar.BarEnd = th.BarEnd
	bar.Current = th.Current
	bar.CurrentN = th.Current
	bar.Empty = th.Empty
}

func getCharset() string {
	if runtime.GOOS == "windows" && os.Getenv("OS") != "CYGWIN" {
		return "cp437"
	}

	var utf8 = ".UTF-8"
	if strings.Contains(os.Getenv("LC_ALL"), utf8) ||
		os.Getenv("LC_CTYPE") == "UTF-8" ||
		strings.Contains(os.Getenv("LANG"), utf8) {
		return "unicode"
	}

	return "ascii"
}

var theme = themes[getCharset()]

// GetTheme returns the theme used to show progress
func GetTheme() *ProgressTheme {
	return theme
}

const maxLabelLength = 40

// ProgressLabel sets the string printed next to the progress indicator
func ProgressLabel(label string) {
	if bar == nil {
		return
	}

	if len(label) > maxLabelLength {
		label = fmt.Sprintf("...%s", label[len(label)-(maxLabelLength-3):])
	}
	bar.Postfix(label)
}

// StartProgress begins a period in which progress is regularly printed
func StartProgress() {
	StartProgressWithTotalBytes(0)
}

// StartProgressWithTotalBytes begins a period in which progress is regularly printed,
// and bps (bytes per second) is estimated from the total size given
func StartProgressWithTotalBytes(totalBytes int64) {
	if bar != nil {
		// Already in-progress
		return
	}

	// shows percentages, to the 1/100th
	bar = pb.New64(100 * 100)
	bar.AlwaysUpdate = true
	bar.RefreshRate = 125 * time.Millisecond
	bar.ShowCounters = false
	bar.ShowFinalTime = false
	bar.TimeBoxWidth = 8
	bar.BarWidth = 20
	bar.SetMaxWidth(80)
	bar.Set64(lastProgressValue)
	bar.TotalBytes = totalBytes

	lastBandwidthAlpha = valueToAlpha(lastProgressValue)

	if settings.noProgress || settings.json {
		// use bar for ETA, but don't print
		bar.NotPrint = true
	}

	theme.apply(bar)
	bar.Start()
}

func alphaToValue(alpha float64) int64 {
	return int64(alpha * 10000.0)
}

func valueToAlpha(val int64) float64 {
	return float64(val) / 10000.0
}

// PauseProgress temporarily stops printing the progress bar
func PauseProgress() {
	if bar != nil {
		bar.AlwaysUpdate = false
	}
}

// ResumeProgress resumes printing the progress bar after PauseProgress was called
func ResumeProgress() {
	if bar != nil {
		bar.AlwaysUpdate = true
	}
}

var maxBucketDuration = 1 * time.Second
var lastBandwidthUpdate time.Time
var lastBandwidthAlpha = 0.0
var bps float64

var lastJsonPrintTime time.Time
var maxJsonPrintDuration = 1 * time.Second

// Progress sets the completion of a task whose progress is being printed
// It only has an effect if StartProgress was already called.
func Progress(alpha float64) {
	msg := jsonMessage{
		"progress":   alpha,
		"percentage": alpha * 100.0,
	}

	if bar != nil {
		msg["eta"] = bar.TimeLeft.Seconds()

		if bar.TotalBytes != 0 {
			if lastBandwidthUpdate.IsZero() {
				lastBandwidthUpdate = time.Now()
			}
			bucketDuration := time.Since(lastBandwidthUpdate)

			if bucketDuration > maxBucketDuration {
				bytesSinceLastUpdate := float64(bar.TotalBytes) * (alpha - lastBandwidthAlpha)
				bps = bytesSinceLastUpdate / bucketDuration.Seconds()
				lastBandwidthUpdate = time.Now()
				lastBandwidthAlpha = alpha
			}

			msg["bps"] = bps
		}
	}

	setBarProgress(alpha)

	if lastJsonPrintTime.IsZero() {
		lastJsonPrintTime = time.Now()
	}
	printDuration := time.Since(lastJsonPrintTime)

	if printDuration > maxJsonPrintDuration {
		lastJsonPrintTime = time.Now()
		send("progress", msg)
	}
}

// ProgressScale sets the scale on which the progress bar is displayed. This can be useful
// when the progress value evolves in another interval than [0, 1]
func ProgressScale(scale float64) {
	if settings.quiet {
		return
	}

	if bar != nil {
		bar.SetScale(scale)
	}
}

func setBarProgress(alpha float64) {
	val := alphaToValue(alpha)
	lastProgressValue = val

	if bar != nil {
		bar.Set64(val)
	}
}

// EndProgress stops refreshing the progress bar and erases it.
func EndProgress() {
	lastProgressValue = 0
	bps = 0
	lastBandwidthAlpha = 0.0
	lastBandwidthUpdate = time.Time{}

	if bar != nil {
		bar.Set64(10000)

		if !settings.noProgress {
			bar.Postfix("")
			bar.Finish()
		}
		bar = nil
	}
}
