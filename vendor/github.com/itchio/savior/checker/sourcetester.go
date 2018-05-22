package checker

import (
	"bytes"
	"encoding/gob"
	"io"
	"log"
	"testing"

	"github.com/itchio/httpkit/progress"
	"github.com/itchio/savior"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func must(t *testing.T, err error) {
	if err != nil {
		assert.NoError(t, err)
		t.FailNow()
	}
}

func RunSourceTest(t *testing.T, source savior.Source, reference []byte) {
	numResumes := 0
	maxResumes := 128

	_, err := source.Resume(nil)
	must(t, err)
	output := NewWriter(reference)

	// first try just copying
	_, err = io.Copy(output, source)
	must(t, err)

	// now reset
	_, err = source.Resume(nil)
	must(t, err)
	_, err = output.Seek(0, io.SeekStart)
	must(t, err)

	totalCheckpoints := 0

	buf := make([]byte, 16*1024)
	var counter int64

	source.SetSourceSaveConsumer(&savior.CallbackSourceSaveConsumer{
		OnSave: func(c *savior.SourceCheckpoint) error {
			numResumes++
			if numResumes > maxResumes {
				must(t, errors.New("too many resumes, something must be wrong"))
			}

			c2, checkpointSize := roundtripThroughGob(t, c)

			totalCheckpoints++
			log.Printf("%s ↓ made %s checkpoint @ %.2f%% (byte %d)", progress.FormatBytes(c2.Offset), progress.FormatBytes(checkpointSize), source.Progress()*100, c2.Offset)

			newOffset, err := source.Resume(c2)
			must(t, err)

			log.Printf("%s ↻ resumed", progress.FormatBytes(newOffset))
			_, err = output.Seek(newOffset, io.SeekStart)
			must(t, err)

			return nil
		},
	})
	var threshold int64 = 256 * 1024 // 256KiB

	for {
		n, readErr := source.Read(buf)

		counter += int64(n)
		if counter > threshold {
			counter = 0
			source.WantSave()
		}

		_, err := output.Write(buf[:n])
		must(t, err)

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			must(t, readErr)
		}
	}

	log.Printf("→ %d checkpoints total", totalCheckpoints)
	assert.True(t, totalCheckpoints > 0, "had at least one checkpoint")
}

func roundtripThroughGob(t *testing.T, c *savior.SourceCheckpoint) (*savior.SourceCheckpoint, int64) {
	saveBuf := new(bytes.Buffer)
	enc := gob.NewEncoder(saveBuf)
	err := enc.Encode(c)
	must(t, err)

	buflen := saveBuf.Len()

	c2 := &savior.SourceCheckpoint{}
	err = gob.NewDecoder(saveBuf).Decode(c2)
	must(t, err)

	return c2, int64(buflen)
}
