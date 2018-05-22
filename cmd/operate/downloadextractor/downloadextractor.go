package downloadextractor

import (
	"os"
	"path/filepath"

	"github.com/itchio/butler/cmd/dl"
	"github.com/itchio/httpkit/htfs"
	"github.com/itchio/httpkit/progress"
	"github.com/itchio/savior"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

// a dummy extractor that only copies from the source

type downloadExtractor struct {
	file     eos.File
	destName string

	sc       savior.SaveConsumer
	consumer *state.Consumer
}

var _ savior.Extractor = (*downloadExtractor)(nil)

// New creates a new extractor that in fact just downloads a single file,
// but also benefits from savior's resumable goodness.
func New(file eos.File, destName string) savior.Extractor {
	return &downloadExtractor{
		file:     file,
		destName: destName,
	}
}

func (de *downloadExtractor) SetConsumer(consumer *state.Consumer) {
	de.consumer = consumer
}

func (de *downloadExtractor) SetSaveConsumer(sc savior.SaveConsumer) {
	de.sc = sc
}

func (de *downloadExtractor) Features() savior.ExtractorFeatures {
	return savior.ExtractorFeatures{
		Name: "download",
		// assuming a "good" http(s) server, ie:
		//   - Returns a positive, non-zero content-length header
		//   - Supports byte range requests
		Preallocate:   true,
		RandomAccess:  true,
		ResumeSupport: savior.ResumeSupportBlock,
	}
}

func (de *downloadExtractor) Resume(checkpoint *savior.ExtractorCheckpoint, sink savior.Sink) (*savior.ExtractorResult, error) {
	consumer := de.consumer
	src := seeksource.FromFile(de.file)

	if checkpoint != nil {
		if checkpoint.SourceCheckpoint != nil && checkpoint.Entry != nil {
			// cool, we'll do that
		} else {
			consumer.Warnf("Invalid checkpoint, starting over")
			checkpoint = nil
		}
	}

	if checkpoint == nil {
		stats, err := de.file.Stat()
		if err != nil {
			return nil, errors.Wrap(err, "stat'ing file to download")
		}
		entry := &savior.Entry{
			CanonicalPath:    de.destName,
			UncompressedSize: stats.Size(),
			Kind:             savior.EntryKindFile,
			Mode:             os.FileMode(0644),
		}

		consumer.Infof("⇓ Pre-allocating %s on disk", progress.FormatBytes(entry.UncompressedSize))
		err = sink.Preallocate(entry)
		if err != nil {
			return nil, errors.Wrap(err, "preallocating")
		}

		checkpoint = &savior.ExtractorCheckpoint{
			Entry: entry,
		}
	}

	entry := checkpoint.Entry

	offset, err := src.Resume(checkpoint.SourceCheckpoint)
	if err != nil {
		consumer.Warnf("Could not resume source, starting over: %s", err.Error())
	} else {
		consumer.Infof("↻ Resuming @ %s", progress.FormatBytes(offset))
	}
	checkpoint.Entry.WriteOffset = offset

	copier := savior.NewCopier(de.sc)

	dest, err := sink.GetWriter(entry)
	if err != nil {
		return nil, errors.Wrap(err, "getting writer")
	}
	defer dest.Close()

	var stopError error

	src.SetSourceSaveConsumer(&savior.CallbackSourceSaveConsumer{
		OnSave: func(sourceCheckpoint *savior.SourceCheckpoint) error {
			checkpoint.SourceCheckpoint = sourceCheckpoint

			err = dest.Sync()
			if err != nil {
				return errors.Wrap(err, "syncing file")
			}

			checkpoint.Progress = src.Progress()

			action, err := de.sc.Save(checkpoint)
			if err != nil {
				return errors.Wrap(err, "saving checkpoint")
			}

			if action == savior.AfterSaveStop {
				copier.Stop()
				stopError = savior.ErrStop
			}

			return nil
		},
	})

	err = copier.Do(&savior.CopyParams{
		Entry:   entry,
		Src:     src,
		Dst:     dest,
		Savable: src,

		EmitProgress: func() {
			consumer.Progress(src.Progress())
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "downloading")
	}

	if stopError != nil {
		return nil, savior.ErrStop
	}

	integrityCheck := func() error {
		hf, ok := de.file.(*htfs.File)
		if !ok {
			consumer.Infof("Not performing integrity checks (not an HTTP resource)")
			return nil
		}

		fs, ok := sink.(*savior.FolderSink)
		if !ok {
			consumer.Infof("Not performing integrity checks (not a folder sink)")
			return nil
		}

		header := hf.GetHeader()
		if header == nil {
			consumer.Infof("Not performing integrity checks (no header)")
			return nil
		}

		targetPath := filepath.Join(fs.Directory, de.destName)

		// the caller may (should tbh) retry in case of integrity failures
		err = dl.CheckIntegrity(consumer, header, entry.UncompressedSize, targetPath)
		if err != nil {
			return errors.Wrap(err, "checking size, hashes etc.")
		}

		return nil
	}

	err = integrityCheck()
	if err != nil {
		return nil, errors.Wrap(err, "performing integrity checks")
	}

	res := &savior.ExtractorResult{
		Entries: []*savior.Entry{
			entry,
		},
	}
	return res, nil
}
