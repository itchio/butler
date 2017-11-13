package comm

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/itchio/butler/pb"
	"github.com/itchio/butler/progress"
)

var counter *progress.Counter

var lastProgressAlpha = 0.0

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
	if counter == nil {
		return
	}

	if len(label) > maxLabelLength {
		label = fmt.Sprintf("...%s", label[len(label)-(maxLabelLength-3):])
	}
	counter.Bar().Postfix(label)
}

// StartProgress begins a period in which progress is regularly printed
func StartProgress() {
	StartProgressWithTotalBytes(0)
}

// StartProgressWithTotalBytes begins a period in which progress is regularly printed,
// and bps (bytes per second) is estimated from the total size given
func StartProgressWithTotalBytes(totalBytes int64) {
	if counter != nil {
		// Already in-progress
		return
	}

	counter = progress.NewCounter()
	bar := counter.Bar()

	bar.ShowCounters = false
	bar.ShowFinalTime = false
	bar.TimeBoxWidth = 8
	bar.BarWidth = 20
	bar.SetMaxWidth(80)

	counter.SetTotalBytes(totalBytes)
	counter.SetProgress(lastProgressAlpha)

	if settings.noProgress || settings.json {
		// use bar for ETA, but don't print
		counter.SetSilent(true)
	}

	theme.apply(bar)
	counter.Start()
}

// PauseProgress temporarily stops printing the progress bar
func PauseProgress() {
	if counter != nil {
		counter.Pause()
	}
}

// ResumeProgress resumes printing the progress bar after PauseProgress was called
func ResumeProgress() {
	if counter != nil {
		counter.Resume()
	}
}

var lastJsonPrintTime time.Time
var maxJsonPrintDuration = 500 * time.Millisecond

// Progress sets the completion of a task whose progress is being printed
// It only has an effect if StartProgress was already called.
func Progress(alpha float64) {
	lastProgressAlpha = alpha

	if counter == nil {
		return
	}

	counter.SetProgress(alpha)

	if lastJsonPrintTime.IsZero() {
		lastJsonPrintTime = time.Now()
	}
	printDuration := time.Since(lastJsonPrintTime)

	if printDuration > maxJsonPrintDuration {
		lastJsonPrintTime = time.Now()
		send("progress", jsonMessage{
			"progress":   alpha,
			"percentage": alpha * 100.0,
			"eta":        counter.ETA().Seconds(),
			"bps":        counter.BPS(),
		})
	}
}

// ProgressScale sets the scale on which the progress bar is displayed. This can be useful
// when the progress value evolves in another interval than [0, 1]
func ProgressScale(scale float64) {
	if settings.quiet {
		return
	}

	if counter != nil {
		counter.Bar().SetScale(scale)
	}
}

// EndProgress stops refreshing the progress bar and erases it.
func EndProgress() {
	if counter != nil {
		counter.SetProgress(1.0)
		counter.Finish()
		counter = nil
	}
}
