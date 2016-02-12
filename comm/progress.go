package comm

import "github.com/cheggaaa/pb"

var bar *pb.ProgressBar

func StartProgress() {
	// percentages, to the 1/100th
	bar = pb.New64(100 * 100)
	bar.ShowPercent = false
	bar.ShowCounters = false
	bar.SetMaxWidth(80)

	bar.BarStart = "|"
	bar.BarEnd = "|"
	bar.Current = "#"
	bar.CurrentN = "#"
	bar.Empty = "-"

	bar.ShowFinalTime = false

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
