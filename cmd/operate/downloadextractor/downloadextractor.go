package downloadextractor

import (
	"os"
	"path/filepath"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/dl"
	"github.com/itchio/httpkit/httpfile"
	"github.com/itchio/savior"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

// a dummy extractor that only copies from the source

type downloadExtractor struct {
	file     eos.File
	destName string

	sc       savior.SaveConsumer
	consumer *state.Consumer
}

var _ savior.Extractor = (*downloadExtractor)(nil)

func New(file eos.File, destName string) *downloadExtractor {
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
			consumer.Warnf("invalid checkpoint, starting over")
			checkpoint = nil
		}
	}

	if checkpoint == nil {
		stats, err := de.file.Stat()
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		entry := &savior.Entry{
			CanonicalPath:    de.destName,
			UncompressedSize: stats.Size(),
			Kind:             savior.EntryKindFile,
			Mode:             os.FileMode(0644),
		}

		consumer.Infof("pre-allocating %s", humanize.IBytes(uint64(entry.UncompressedSize)))
		err = sink.Preallocate(entry)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		checkpoint = &savior.ExtractorCheckpoint{
			Entry: entry,
		}
	}

	entry := checkpoint.Entry

	offset, err := src.Resume(checkpoint.SourceCheckpoint)
	if err != nil {
		consumer.Warnf("could not resume source, starting over: %s", err.Error())
	} else {
		consumer.Infof("resuming @ %s", humanize.IBytes(uint64(offset)))
	}
	checkpoint.Entry.WriteOffset = offset

	copier := savior.NewCopier(de.sc)

	dest, err := sink.GetWriter(entry)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	defer dest.Close()

	src.SetSourceSaveConsumer(&savior.CallbackSourceSaveConsumer{
		OnSave: func(sourceCheckpoint *savior.SourceCheckpoint) error {
			checkpoint.SourceCheckpoint = sourceCheckpoint
			return nil
		},
	})

	err = copier.Do(&savior.CopyParams{
		Entry:   entry,
		Src:     src,
		Dst:     dest,
		Savable: src,

		MakeCheckpoint: func() (*savior.ExtractorCheckpoint, error) {
			consumer.Infof("saving @ %.2f%%", src.Progress()*100.0)

			err = dest.Sync()
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}

			return checkpoint, nil
		},

		EmitProgress: func() {
			consumer.Progress(src.Progress())
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	integrityCheck := func() error {
		hf, ok := de.file.(*httpfile.HTTPFile)
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
			return errors.Wrap(err, 0)
		}

		return nil
	}

	err = integrityCheck()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &savior.ExtractorResult{
		Entries: []*savior.Entry{
			entry,
		},
	}
	return res, nil
}
