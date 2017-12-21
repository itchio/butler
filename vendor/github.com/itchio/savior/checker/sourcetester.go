package checker

import (
	"bytes"
	"encoding/gob"
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
	for {
		n, readErr := source.Read(buf)

		_, err := output.Write(buf[:n])
		must(t, err)

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			must(t, readErr)
		}

		c, err := source.Save()
		must(t, err)

		if c != nil {
			c2, checkpointSize := roundtripThroughGob(t, c)

			totalCheckpoints++
			log.Printf("%s ↓ made %s checkpoint @ %.2f%%", humanize.IBytes(uint64(c2.Offset)), humanize.IBytes(uint64(checkpointSize)), source.Progress()*100)

			newOffset, err := source.Resume(c2)
			must(t, err)

			log.Printf("%s ↻ resumed", humanize.IBytes(uint64(newOffset)))
			_, err = output.Seek(newOffset, io.SeekStart)
			must(t, err)
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
