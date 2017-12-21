package checker

import (
	"bytes"
	"encoding/gob"
	"log"
	"os"
	"testing"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/savior"
	"github.com/itchio/wharf/state"
)

type MakeExtractorFunc func() savior.Extractor
type ShouldSaveFunc func() bool

var showSaviorConsumerOutput = os.Getenv("SAVIOR_CONSUMER") == "1"

func RunExtractorText(t *testing.T, makeExtractor MakeExtractorFunc, shouldSave ShouldSaveFunc) {
	var c *savior.ExtractorCheckpoint
	var totalCheckpointSize int64

	sc := NewTestSaveConsumer(1*1024*1024, func(checkpoint *savior.ExtractorCheckpoint) (savior.AfterSaveAction, error) {
		if shouldSave() {
			c2, checkpointSize := roundtripEThroughGob(t, checkpoint)
			totalCheckpointSize += int64(checkpointSize)
			c = c2
			log.Printf("↓ saved @ %.0f%% (%s checkpoint)", c.Progress*100, humanize.IBytes(uint64(checkpointSize)))
			return savior.AfterSaveStop, nil
		}

		savior.Debugf("↷ Skipping over checkpoint at #%d", checkpoint.EntryIndex)
		return savior.AfterSaveContinue, nil
	})

	var numProgressCalls int64
	var numJumpbacks int64
	var lastProgress float64
	consumer := &state.Consumer{
		OnProgress: func(progress float64) {
			if progress < lastProgress {
				numJumpbacks++
				log.Printf("mh, progress jumped back from %.2f to %.2f", lastProgress*100, progress*100)
			}
			lastProgress = progress
			numProgressCalls++
		},
	}

	if showSaviorConsumerOutput {
		consumer.OnMessage = func(lvl string, message string) {
			if lvl != "debug" {
				log.Printf("[%s] %s", lvl, message)
			}
		}
	}

	startTime := time.Now()

	maxResumes := 24
	numResumes := 0
	for {
		if numResumes > maxResumes {
			t.Error("Too many resumes, something must be wrong")
			t.FailNow()
		}

		ex := makeExtractor()
		ex.SetSaveConsumer(sc)
		ex.SetConsumer(consumer)

		if c == nil {
			savior.Debugf("→ first resume")
		} else {
			savior.Debugf("↻ resumed @ %.0f%%", c.Progress*100)
		}
		res, err := ex.Resume(c)
		if err != nil {
			if err == savior.StopErr {
				numResumes++
				continue
			}
			must(t, err)
		}

		// yay, we did it!
		totalDuration := time.Since(startTime)
		perSec := humanize.IBytes(uint64(float64(res.Size()) / totalDuration.Seconds()))
		log.Printf(" ⇒ extracted %s @ %s/s (%s total)", res.Stats(), perSec, totalDuration)
		if numResumes > 0 {
			meanCheckpointSize := float64(totalCheckpointSize) / float64(numResumes)
			log.Printf(" ⇒ %d resumes, %s avg checkpoint", numResumes, humanize.IBytes(uint64(meanCheckpointSize)))
		} else {
			log.Printf(" ⇒ no resumes")
		}
		log.Printf(" ⇒ progress called %d times, %d jumpbacks", numProgressCalls, numJumpbacks)
		log.Printf(" ⇒ features: %s", ex.Features())

		break
	}
}

func roundtripEThroughGob(t *testing.T, c *savior.ExtractorCheckpoint) (*savior.ExtractorCheckpoint, int) {
	saveBuf := new(bytes.Buffer)
	enc := gob.NewEncoder(saveBuf)
	err := enc.Encode(c)
	must(t, err)

	buflen := saveBuf.Len()

	c2 := &savior.ExtractorCheckpoint{}
	err = gob.NewDecoder(saveBuf).Decode(c2)
	must(t, err)

	return c2, buflen
}
