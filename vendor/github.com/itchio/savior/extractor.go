package savior

import (
	"encoding/gob"
	"fmt"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/wharf/state"
)

type ExtractorCheckpoint struct {
	SourceCheckpoint *SourceCheckpoint
	EntryIndex       int64
	Entry            *Entry
	Progress         float64
	Data             interface{}
}

type ExtractorResult struct {
	Entries []*Entry
}

func (er *ExtractorResult) Stats() string {
	var numFiles, numDirs, numSymlinks int
	var totalBytes int64
	for _, entry := range er.Entries {
		switch entry.Kind {
		case EntryKindFile:
			numFiles++
		case EntryKindDir:
			numDirs++
		case EntryKindSymlink:
			numSymlinks++
		}
		totalBytes += entry.UncompressedSize
	}

	return fmt.Sprintf("%s (in %d files, %d dirs, %d symlinks)", humanize.IBytes(uint64(totalBytes)), numFiles, numDirs, numSymlinks)
}

func (er *ExtractorResult) Size() int64 {
	var totalBytes int64
	for _, entry := range er.Entries {
		totalBytes += entry.UncompressedSize
	}

	return totalBytes
}

type ExtractorFeatures struct {
	Name           string
	ResumeSupport  ResumeSupport
	Preallocate    bool
	RandomAccess   bool
	SourceFeatures *SourceFeatures
}

func (ef ExtractorFeatures) String() string {
	res := fmt.Sprintf("%s: resume=%s", ef.Name, ef.ResumeSupport)
	if ef.Preallocate {
		res += " +preallocate"
	}

	if ef.RandomAccess {
		res += " +randomaccess"
	}

	if ef.SourceFeatures != nil {
		res += fmt.Sprintf(" (via source %s: resume=%s)", ef.SourceFeatures.Name, ef.SourceFeatures.ResumeSupport)
	}

	return res
}

type ResumeSupport int

const (
	// While the extractor exposes Save/Resume, in practice, resuming
	// will probably waste I/O and processing redoing a lot of work
	// that was already done, so it's not recommended to run it against
	// a networked resource
	ResumeSupportNone ResumeSupport = 0
	// The extractor can save/resume between each entry, but not in the middle of an entry
	ResumeSupportEntry ResumeSupport = 1
	// The extractor can save/resume within an entry, on a deflate/bzip2 block boundary for example
	ResumeSupportBlock ResumeSupport = 2
)

func (rs ResumeSupport) String() string {
	switch rs {
	case ResumeSupportNone:
		return "none"
	case ResumeSupportEntry:
		return "entry"
	case ResumeSupportBlock:
		return "block"
	default:
		return "unknown resume support"
	}
}

type AfterSaveAction int

const (
	AfterSaveContinue AfterSaveAction = 1
	AfterSaveStop     AfterSaveAction = 2
)

type SaveConsumer interface {
	ShouldSave(copiedBytes int64) bool
	Save(checkpoint *ExtractorCheckpoint) (AfterSaveAction, error)
}

func NopConsumer() *state.Consumer {
	return &state.Consumer{
		OnMessage:        func(lvl string, msg string) {},
		OnProgressLabel:  func(label string) {},
		OnPauseProgress:  func() {},
		OnResumeProgress: func() {},
		OnProgress:       func(progress float64) {},
	}
}

type Extractor interface {
	SetSaveConsumer(saveConsumer SaveConsumer)
	SetConsumer(consumer *state.Consumer)
	Resume(checkpoint *ExtractorCheckpoint, sink Sink) (*ExtractorResult, error)
	Features() ExtractorFeatures
}

func init() {
	gob.Register(&ExtractorCheckpoint{})
}

type nopSaveConsumer struct{}

var _ SaveConsumer = (*nopSaveConsumer)(nil)

func NopSaveConsumer() SaveConsumer {
	return &nopSaveConsumer{}
}

func (nsc *nopSaveConsumer) ShouldSave(n int64) bool {
	return false
}

func (nsc *nopSaveConsumer) Save(checkpoint *ExtractorCheckpoint) (AfterSaveAction, error) {
	return AfterSaveContinue, nil
}
