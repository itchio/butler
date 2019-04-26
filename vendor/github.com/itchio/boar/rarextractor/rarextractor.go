package rarextractor

import (
	"path/filepath"
	"runtime"
	"time"

	"github.com/pkg/errors"

	"github.com/itchio/dmcunrar-go/dmcunrar"
	"github.com/itchio/httpkit/progress"
	"github.com/itchio/savior"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

type RarExtractor interface {
	savior.Extractor
}

type rarExtractor struct {
	file eos.File

	consumer     *state.Consumer
	saveConsumer savior.SaveConsumer

	archive *dmcunrar.Archive

	initialProgress float64
	progress        float64

	freed bool
}

var _ RarExtractor = (*rarExtractor)(nil)

func New(file eos.File, consumer *state.Consumer) (RarExtractor, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	archive, err := dmcunrar.OpenArchive(file, stat.Size())
	if err != nil {
		return nil, err
	}

	re := &rarExtractor{
		file:         file,
		archive:      archive,
		consumer:     consumer,
		saveConsumer: savior.NopSaveConsumer(),
	}
	runtime.SetFinalizer(re, func(re *rarExtractor) {
		re.free()
	})
	return re, nil
}

func (re *rarExtractor) SetConsumer(consumer *state.Consumer) {
	re.consumer = consumer
}

func (re *rarExtractor) SetSaveConsumer(saveConsumer savior.SaveConsumer) {
	re.saveConsumer = saveConsumer
}

func (re *rarExtractor) Entries() []*savior.Entry {
	if re.freed {
		return nil
	}

	var entries []*savior.Entry

	numEntries := re.archive.GetFileCount()
	for i := int64(0); i < numEntries; i++ {
		entry, err := re.getEntry(i)
		if err == nil {
			entries = append(entries, entry)
		}
	}
	return entries
}

func (re *rarExtractor) getEntry(i int64) (*savior.Entry, error) {
	filename, err := re.archive.GetFilename(i)
	if err != nil {
		return nil, err
	}
	filename = filepath.ToSlash(filename)

	e := &savior.Entry{
		CanonicalPath: filename,
		Kind:          savior.EntryKindFile,
		Mode:          0644,
	}

	stat := re.archive.GetFileStat(i)
	if stat != nil {
		e.UncompressedSize = stat.GetUncompressedSize()
	}

	if re.archive.FileIsDirectory(i) {
		e.Kind = savior.EntryKindDir
		e.Mode = 0755
	} else {
		supportErr := re.archive.FileIsSupported(i) != nil
		if supportErr {
			return nil, errors.Errorf("rar: an entry cannot be decompressed (%s)", filename)
		}
	}

	return e, nil
}

func (re *rarExtractor) Resume(checkpoint *savior.ExtractorCheckpoint, sink savior.Sink) (*savior.ExtractorResult, error) {
	if re.freed {
		return nil, errors.New("cannot use freed rarExtractor")
	}

	isFresh := false

	if checkpoint == nil {
		isFresh = true
		re.consumer.Infof("→ Starting fresh extraction")
		checkpoint = &savior.ExtractorCheckpoint{
			EntryIndex: 0,
		}
	} else {
		re.consumer.Infof("↻ Resuming @ %.1f%%", checkpoint.Progress*100)
		re.initialProgress = checkpoint.Progress
	}

	numEntries := re.archive.GetFileCount()
	var totalBytes int64

	var entries []*savior.Entry
	for i := int64(0); i < numEntries; i++ {
		entry, err := re.getEntry(i)
		if err != nil {
			return nil, err
		}
		totalBytes += entry.UncompressedSize
		entries = append(entries, entry)
	}

	if isFresh {
		re.consumer.Infof("⇓ Pre-allocating %s on disk", progress.FormatBytes(totalBytes))
		preallocateItem := func(entry *savior.Entry) error {
			if entry.Kind == savior.EntryKindFile {
				err := sink.Preallocate(entry)
				if err != nil {
					return errors.Wrap(err, "preallocating entries")
				}
			}
			return nil
		}

		preallocateStart := time.Now()
		for i, entry := range entries {
			err := preallocateItem(entry)
			if err != nil {
				return nil, errors.Wrapf(err, "preallocating item %d", i)
			}
		}
		preallocateDuration := time.Since(preallocateStart)
		re.consumer.Infof("⇒ Pre-allocated in %s, nothing can stop us now", preallocateDuration)
	}

	extractEntry := func(entryIndex int64, entry *savior.Entry) error {
		switch entry.Kind {
		case savior.EntryKindDir:
			return sink.Mkdir(entry)
		case savior.EntryKindFile:
			writer, err := sink.GetWriter(entry)
			if err != nil {
				return errors.Wrap(err, "getting writer for regular file")
			}

			cw := counter.NewWriterCallback(func(count int64) {
				if entry.UncompressedSize > 0 {
					localProgress := float64(count) / float64(totalBytes)
					re.progress = re.initialProgress + localProgress
					re.consumer.Progress(re.progress)
				}
			}, writer)

			ef := dmcunrar.NewExtractedFile(cw)
			defer ef.Free()

			err = re.archive.ExtractFile(ef, entryIndex)
			if err != nil {
				return errors.Wrapf(err, "while extracting (%s)", entry.CanonicalPath)
			}
			re.initialProgress = re.progress

			if re.saveConsumer.ShouldSave(cw.Count()) {
				checkpoint := &savior.ExtractorCheckpoint{
					EntryIndex: entryIndex + 1,
					Progress:   re.progress,
				}
				action, err := re.saveConsumer.Save(checkpoint)
				if err != nil {
					re.consumer.Warnf("7-zip extractor could not save checkpoint: %s", err.Error())
				}

				if action == savior.AfterSaveStop {
					// keep giving nil streams to 7-zip after this
					return savior.ErrStop
				}
			}

			return nil

		case savior.EntryKindSymlink:
			return errors.New("rar: symlinks are not supported")
		}
		return nil
	}

	for i := checkpoint.EntryIndex; i < int64(len(entries)); i++ {
		entry := entries[i]
		err := extractEntry(i, entry)
		if err != nil {
			return nil, err
		}
	}

	res := &savior.ExtractorResult{
		Entries: entries,
	}
	re.free()

	return res, nil
}

func (re *rarExtractor) free() {
	if re.freed {
		return
	}

	if re.archive != nil {
		re.archive.Free()
		re.archive = nil
	}

	re.freed = true
}

func (re *rarExtractor) Features() savior.ExtractorFeatures {
	return Features()
}

func Features() savior.ExtractorFeatures {
	return savior.ExtractorFeatures{
		Name:          "dmc_unrar",
		Preallocate:   true,
		// rar has no central directory, and interleaved blocks,
		// so let's not pretend we meaningfully pause/resume a
		// download, let's just force downloading to disk first.
		RandomAccess:  false,
		ResumeSupport: savior.ResumeSupportNone,
	}
}
