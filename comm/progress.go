package comm

import (
	"os"
	"runtime"
	"strings"

	"github.com/cheggaaa/pb"
)

var bar *pb.ProgressBar

type progressTheme struct {
	BarStart string
	BarEnd   string
	Current  string
	Empty    string
}

var themes = map[string]*progressTheme{
	"unicode": {"▐", "▌", "▓", "░"},
	"ascii":   {"|", "|", "#", "-"},
	"cp437":   {"▐", "▌", "█", "░"},
}

func (th *progressTheme) apply(bar *pb.ProgressBar) {
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

func StartProgress() {
	// percentages, to the 1/100th
	bar = pb.New64(100 * 100)
	bar.ShowCounters = false
	bar.ShowFinalTime = false
	bar.SetMaxWidth(80)

	themes[getCharset()].apply(bar)

	if !settings.no_progress {
		bar.Start()
	}
}

func Progress(perc float64) {
	if settings.quiet {
		return
	}

	send("progress", jsonMessage{
		"percentage": perc,
	})
}

func setBarProgress(perc float64) {
	if bar == nil {
		StartProgress()
	}
	bar.Set64(int64(perc * 100.0))
}

func EndProgress() {
	if bar != nil {
		bar.Set64(10000)

		if !settings.no_progress {
			bar.Finish()
		}
		bar = nil
	}
}
