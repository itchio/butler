package checker

import (
	"bytes"
	"encoding/gob"
	"log"
	"os"
	"testing"
	"time"

	"github.com/itchio/httpkit/progress"
	"github.com/itchio/savior"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type MakeExtractorFunc func() savior.Extractor
type ShouldSaveFunc func() bool

var showSaviorConsumerOutput = os.Getenv("SAVIOR_CONSUMER") == "1"

func RunExtractorText(t *testing.T, makeExtractor MakeExtractorFunc, sink *Sink, shouldSave ShouldSaveFunc) {
	var c *savior.ExtractorCheckpoint
	var totalCheckpointSize int64

	sc := NewTestSaveConsumer(1*1024*1024, func(checkpoint *savior.ExtractorCheckpoint) (savior.AfterSaveAction, error) {
		if shouldSave() {
			c2, checkpointSize := roundtripEThroughGob(t, checkpoint)
			totalCheckpointSize += int64(checkpointSize)
			c = c2
			log.Printf("↓ saved @ %.0f%% (%s checkpoint, entry %d)", c.Progress*100, progress.FormatBytes(checkpointSize), c.EntryIndex)
			return savior.AfterSaveStop, nil
		}

		savior.Debugf("↷ Skipping over checkpoint at #%d", checkpoint.EntryIndex)
		return savior.AfterSaveContinue, nil
	})

	sink.Reset()

	var numProgressCalls int64
	var numJumpbacks int64
	var lastProgress float64
	consumer := &state.Consumer{
		OnProgress: func(progress float64) {
			if progress < lastProgress {
				numJumpbacks++
				log.Printf("mh, progress jumped back from %f to %f", lastProgress, progress)
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

	maxResumes := 128
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
		res, err := ex.Resume(c, sink)
		if err != nil {
			if errors.Cause(err) == savior.ErrStop {
				numResumes++
				continue
			}
			must(t, err)
		}

		// yay, we did it!
		totalDuration := time.Since(startTime)
		perSec := progress.FormatBPS(res.Size(), totalDuration)
		log.Printf(" ⇒ extracted %s @ %s/s (%s total)", res.Stats(), perSec, totalDuration)
		if numResumes > 0 {
			meanCheckpointSize := float64(totalCheckpointSize) / float64(numResumes)
			log.Printf(" ⇒ %d resumes, %s avg checkpoint", numResumes, progress.FormatBytes(int64(meanCheckpointSize)))
		} else {
			log.Printf(" ⇒ no resumes")
		}
		log.Printf(" ⇒ progress called %d times, %d jumpbacks", numProgressCalls, numJumpbacks)
		log.Printf(" ⇒ features: %s", ex.Features())

		break
	}

	assert.NoError(t, sink.Validate())
}

func roundtripEThroughGob(t *testing.T, c *savior.ExtractorCheckpoint) (*savior.ExtractorCheckpoint, int64) {
	saveBuf := new(bytes.Buffer)
	enc := gob.NewEncoder(saveBuf)
	err := enc.Encode(c)
	must(t, err)

	buflen := saveBuf.Len()

	c2 := &savior.ExtractorCheckpoint{}
	err = gob.NewDecoder(saveBuf).Decode(c2)
	must(t, err)

	return c2, int64(buflen)
}
