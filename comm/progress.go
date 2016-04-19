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
	if bar != nil {
		// Already in-progress
		return
	}

	if settings.noProgress || settings.json {
		// Don't want a bar, ever.
		return
	}

	// shows percentages, to the 1/100th
	bar = pb.New64(100 * 100)
	bar.AlwaysUpdate = true
	bar.RefreshRate = 250 * time.Millisecond
	bar.ShowCounters = false
	bar.ShowFinalTime = false
	bar.TimeBoxWidth = 8
	bar.BarWidth = 20
	bar.SetMaxWidth(80)

	theme.apply(bar)
	bar.Start()
}

func PauseProgress() {
	if bar != nil {
		bar.AlwaysUpdate = false
	}
}

func ResumeProgress() {
	if bar != nil {
		bar.AlwaysUpdate = true
	}
}

// Progress sets the completion of a task whose progress is being printed
// It only has an effect if StartProgress was already called.
func Progress(perc float64) {
	if settings.quiet {
		return
	}

	send("progress", jsonMessage{
		"percentage": perc * 100.0,
	})
}

func ProgressScale(scale float64) {
	if settings.quiet {
		return
	}

	if bar != nil {
		bar.SetScale(scale)
	}
}

func setBarProgress(perc float64) {
	if bar != nil {
		bar.Set64(int64(perc * 10000.0))
	}
}

// EndProgress stops refreshing the progress bar and erases it.
func EndProgress() {
	if bar != nil {
		bar.Set64(10000)

		if !settings.noProgress {
			bar.Postfix("")
			bar.Finish()
		}
		bar = nil
	}
}
