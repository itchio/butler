package main

import "github.com/cheggaaa/pb"

var bar *pb.ProgressBar

func StartProgress() {
	// percentages, to the 1/100th
	bar = pb.New64(100 * 100)
	bar.ShowCounters = false
	bar.SetMaxWidth(80)

	if !*appArgs.no_progress {
		bar.Start()
	}
}

func Progress(perc float64) {
	if *appArgs.quiet {
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

		if !*appArgs.no_progress {
			bar.Finish()
		}
		bar = nil
	}
}
