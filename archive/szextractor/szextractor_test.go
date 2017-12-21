package szextractor_test

import (
	"bytes"
	"log"
	"os"
	"testing"
	"time"

	"github.com/itchio/butler/archive/szextractor"
	"github.com/itchio/savior"
	"github.com/itchio/savior/checker"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
	"github.com/stretchr/testify/assert"
)

func must(t *testing.T, err error) {
	if err != nil {
		assert.NoError(t, err)
		t.FailNow()
	}
}

func TestSzExtractor(t *testing.T) {
	sink := checker.MakeTestSinkAdvanced(40)
	zipBytes := checker.MakeZip(t, sink)

	file := &memoryFile{
		br:   bytes.NewReader(zipBytes),
		name: "szextractor_test.zip",
	}

	initialConsumer := &state.Consumer{
		OnMessage: func(lvl string, message string) {
			log.Printf("[%s] %s", lvl, message)
		},
	}

	makeExtractor := func() savior.Extractor {
		ex, err := szextractor.New(file, initialConsumer)
		must(t, err)
		return ex
	}

	log.Printf("Testing szextractor on .zip, no resumes")
	checker.RunExtractorText(t, makeExtractor, sink, func() bool {
		return false
	})

	log.Printf("Testing szextractor on .zip, all resumes")
	checker.RunExtractorText(t, makeExtractor, sink, func() bool {
		return true
	})

	log.Printf("Testing szextractor on .zip, every other")
	i := 0
	checker.RunExtractorText(t, makeExtractor, sink, func() bool {
		i++
		return i%2 == 0
	})
}

type memoryFile struct {
	br   *bytes.Reader
	name string
}

var _ eos.File = (*memoryFile)(nil)

func (mf *memoryFile) Close() error {
	// all hail the all-seeing eye of the GC
	return nil
}

func (mf *memoryFile) Read(buf []byte) (int, error) {
	return mf.br.Read(buf)
}

func (mf *memoryFile) ReadAt(buf []byte, offset int64) (int, error) {
	return mf.br.ReadAt(buf, offset)
}

func (mf *memoryFile) Seek(offset int64, whence int) (int64, error) {
	return mf.br.Seek(offset, whence)
}

func (mf *memoryFile) Stat() (os.FileInfo, error) {
	return &memoryFileInfo{mf}, nil
}

// memoryFileInfo implements os.FileInfo for memoryfiles
type memoryFileInfo struct {
	mf *memoryFile
}

var _ os.FileInfo = (*memoryFileInfo)(nil)

func (mfi *memoryFileInfo) Name() string {
	return mfi.mf.name
}

func (mfi *memoryFileInfo) Size() int64 {
	return mfi.mf.br.Size()
}

func (mfi *memoryFileInfo) Mode() os.FileMode {
	return os.FileMode(0)
}

func (mfi *memoryFileInfo) ModTime() time.Time {
	return time.Now()
}

func (mfi *memoryFileInfo) IsDir() bool {
	return false
}

func (mfi *memoryFileInfo) Sys() interface{} {
	return nil
}
