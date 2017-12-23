package checker

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"log"
	"testing"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/savior"
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
	assert.NoError(t, err)
	output := NewWriter(reference)

	// first try just copying
	_, err = io.Copy(output, source)
	assert.NoError(t, err)

	// now reset
	_, err = source.Resume(nil)
	assert.NoError(t, err)
	_, err = output.Seek(0, io.SeekStart)
	assert.NoError(t, err)

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
			log.Printf("%s ↓ made %s checkpoint @ %.2f%%", humanize.IBytes(uint64(c2.Offset)), humanize.IBytes(uint64(checkpointSize)), source.Progress()*100)

			newOffset, err := source.Resume(c2)
			must(t, err)

			log.Printf("%s ↻ resumed", humanize.IBytes(uint64(newOffset)))
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

func roundtripThroughGob(t *testing.T, c *savior.SourceCheckpoint) (*savior.SourceCheckpoint, int) {
	saveBuf := new(bytes.Buffer)
	enc := gob.NewEncoder(saveBuf)
	err := enc.Encode(c)
	must(t, err)

	buflen := saveBuf.Len()

	c2 := &savior.SourceCheckpoint{}
	err = gob.NewDecoder(saveBuf).Decode(c2)
	must(t, err)

	return c2, buflen
}
