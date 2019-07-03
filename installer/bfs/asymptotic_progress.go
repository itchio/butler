package bfs

import (
	"time"

	"github.com/itchio/headway/state"
)

func StartAsymptoticProgress(consumer *state.Consumer, cancel chan struct{}) {
	start := time.Now()

	go func() {
		for {
			select {
			case <-time.After(500 * time.Millisecond):
				x := time.Since(start).Seconds()
				// this function reaches 80% progress at 25 seconds,
				// always approaches 100% without ever quite reaching it
				progress := x / (6.0 + x)
				consumer.Progress(progress)
			case <-cancel:
				// we're done sending progress events
				return
			}
		}
	}()
}
