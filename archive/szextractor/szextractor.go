package szextractor

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/savior"
	"github.com/itchio/sevenzip-go/sz"
	"github.com/itchio/wharf/archiver"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

var dontEnsureDeps = os.Getenv("BUTLER_NO_DEPS") == "1"
var ensuredDeps = false

type SzExtractor interface {
	savior.Extractor
	GetFormat() string
}

type szExtractor struct {
	file eos.File

	consumer     *state.Consumer
	saveConsumer savior.SaveConsumer

	lib     *sz.Lib
	archive *sz.Archive
	in      *sz.InStream
	format  string

	resumeSupport savior.ResumeSupport

	initialProgress float64
	progress        float64

	freed bool
}

var _ SzExtractor = (*szExtractor)(nil)

func New(file eos.File, consumer *state.Consumer) (SzExtractor, error) {
	se := &szExtractor{
		file:          file,
		consumer:      consumer,
		resumeSupport: savior.ResumeSupportEntry, // start by being optimistic
		saveConsumer:  savior.NopSaveConsumer(),
	}
	runtime.SetFinalizer(se, func(se *szExtractor) {
		se.free()
	})

	err := EnsureDeps(consumer)
	if err != nil {
		return nil, errors.Wrap(err, "ensuring 7-zip deps")
	}

	lib, err := sz.NewLib()
	if err != nil {
		return nil, errors.Wrap(err, "opening 7-zip library")
	}
	se.lib = lib

	stats, err := file.Stat()
	if err != nil {
		return nil, errors.Wrap(err, "stat'ing file")
	}

	ext := nameToExt(stats.Name())

	in, err := sz.NewInStream(file, ext, stats.Size())
	if err != nil {
		return nil, errors.Wrap(err, "creating 7-zip input stream")
	}
	se.in = in

	// try by extension first
	consumer.Debugf("Trying by extension '%s'", ext)
	a, err := lib.OpenArchive(in, false)
	if err != nil {
		// try by signature next
		_, err = in.Seek(0, io.SeekStart)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		consumer.Debugf("Trying by signature")
		a, err = lib.OpenArchive(in, true)
		if err != nil {
			// With the current libc7zip setup, 7-zip will refuse to
			// extract some self-extracting installers - for those,
			// we need to give it the `.cab` extension instead
			// Maybe the multivolume interface takes care of that?
			// Command-line `7z` has no issue with them.
			if ext == "exe" {
				consumer.Debugf("Trying by extension 'cab'")

				// if it was an .exe, try with a .cab extension
				in.Free()

				ext = "cab"

				in, err := sz.NewInStream(file, ext, stats.Size())
				if err != nil {
					return nil, errors.Wrap(err, "creating input stream")
				}

				a, err = lib.OpenArchive(in, false) // by ext
				if err != nil {
					return nil, errors.Wrap(err, "opening archive with 7-zip")
				}
			} else {
				// well, we're out of options
				return nil, errors.Errorf("could not open with 7-zip: %s", stats.Name())
			}
		}
	}
	se.archive = a

	se.format = a.GetArchiveFormat()
	if se.format == "7z" {
		// .7z is a known non-resumable format - resuming means a lot
		// of extra IO and decompression work on already-extracted blocks,
		// so we just don't want to do it on-the-fly
		se.resumeSupport = savior.ResumeSupportNone
	}

	return se, nil
}

func (se *szExtractor) GetFormat() string {
	return se.format
}

func (se *szExtractor) SetConsumer(consumer *state.Consumer) {
	se.consumer = consumer
}

func (se *szExtractor) SetSaveConsumer(saveConsumer savior.SaveConsumer) {
	se.saveConsumer = saveConsumer
}

func (se *szExtractor) Entries() []*savior.Entry {
	if se.freed {
		return nil
	}

	var entries []*savior.Entry

	numEntries, err := se.archive.GetItemCount()
	if err != nil {
		return nil
	}

	for i := int64(0); i < numEntries; i++ {
		item := se.archive.GetItem(int64(i))
		entries = append(entries, szEntry(item))
		item.Free()
	}
	return entries
}

func (se *szExtractor) Resume(checkpoint *savior.ExtractorCheckpoint, sink savior.Sink) (*savior.ExtractorResult, error) {
	if se.freed {
		return nil, errors.New("cannot use freed szExtractor")
	}

	isFresh := false

	if checkpoint == nil {
		isFresh = true
		se.consumer.Infof("→ Starting fresh extraction")
		checkpoint = &savior.ExtractorCheckpoint{
			EntryIndex: 0,
		}
	} else {
		se.consumer.Infof("↻ Resuming @ %.1f%%", checkpoint.Progress*100)
		se.initialProgress = checkpoint.Progress
	}

	numEntries, err := se.archive.GetItemCount()
	if err != nil {
		return nil, errors.Wrap(err, "getting item count")
	}

	var totalBytes int64
	var indices []int64
	prepareItem := func(i int64) {
		item := se.archive.GetItem(int64(i))
		defer item.Free()
		entry := szEntry(item)
		totalBytes += entry.UncompressedSize

		if int64(i) >= checkpoint.EntryIndex {
			indices = append(indices, int64(i))
		}
	}

	for i := int64(0); i < numEntries; i++ {
		prepareItem(i)
	}

	if len(indices) > 0 {
		if isFresh {
			se.consumer.Infof("⇓ Pre-allocating %s on disk", humanize.IBytes(uint64(totalBytes)))
			preallocateItem := func(i int64) error {
				item := se.archive.GetItem(int64(i))
				defer item.Free()
				entry := szEntry(item)

				if entry.Kind == savior.EntryKindFile {
					err = sink.Preallocate(entry)
					if err != nil {
						return errors.Wrap(err, "preallocating entries")
					}
				}
				return nil
			}

			preallocateStart := time.Now()
			for i := int64(0); i < numEntries; i++ {
				err = preallocateItem(i)
				if err != nil {
					return nil, errors.Wrapf(err, "preallocating item %d", i)
				}
			}
			preallocateDuration := time.Since(preallocateStart)
			se.consumer.Infof("⇒ Pre-allocated in %s, nothing can stop us now", preallocateDuration)
		}

		sc := &szCallbacks{
			se:   se,
			sink: sink,
		}
		ec, err := sz.NewExtractCallback(sc)
		if err != nil {
			return nil, errors.Wrap(err, "creating extract callback")
		}

		err = se.archive.ExtractSeveral(indices, ec)
		if err != nil {
			return nil, errors.Wrap(err, "extracting several files")
		}

		if sc.stopped {
			return nil, savior.ErrStop
		}
	} else {
		se.consumer.Infof("Nothing to do! (all items extracted)")
	}

	// compile list of entries
	res := &savior.ExtractorResult{
		Entries: []*savior.Entry{},
	}
	listEntry := func(i int64) {
		item := se.archive.GetItem(i)
		defer item.Free()
		entry := szEntry(item)
		res.Entries = append(res.Entries, entry)
	}
	for i := int64(0); i < numEntries; i++ {
		listEntry(i)
	}

	se.consumer.Statf("Extracted %s", res.Stats())

	se.free()

	return res, nil
}

func (se *szExtractor) Features() savior.ExtractorFeatures {
	return savior.ExtractorFeatures{
		Name:          "sz",
		Preallocate:   true, // some formats might have a 0 UncompressedSize?
		RandomAccess:  true, // that's not true for .tar, but we don't use sz for that
		ResumeSupport: se.resumeSupport,
	}
}

// implement sz.ExtractCallbackFuncs

type szCallbacks struct {
	se      *szExtractor
	sink    savior.Sink
	stopped bool
}

var _ sz.ExtractCallbackFuncs = (*szCallbacks)(nil)

func (sc *szCallbacks) SetProgress(complete int64, total int64) {
	if sc.stopped {
		return
	}

	se := sc.se

	if total > 0 {
		thisRunProgress := float64(complete) / float64(total)
		actualProgress := se.initialProgress + (1.0-se.initialProgress)*thisRunProgress
		se.progress = actualProgress
		se.consumer.Progress(actualProgress)
	}
	// TODO: some formats don't have 'total' value, should we do
	// something smart there?
}

func (sc *szCallbacks) GetStream(item *sz.Item) (*sz.OutStream, error) {
	if sc.stopped {
		return nil, nil
	}

	se := sc.se

	entry := szEntry(item)
	entryIndex := item.GetArchiveIndex()

	if entry.Kind == savior.EntryKindDir {
		err := sc.sink.Mkdir(entry)
		if err != nil {
			return nil, errors.Wrap(err, "creating directory")
		}

		// don't give a stream for a dir
		return nil, nil
	}

	if entry.Kind == savior.EntryKindSymlink {
		if entry.Linkname != "" {
			// cool, it was in the metadata, let's just do it now
			err := sc.sink.Symlink(entry, entry.Linkname)
			if err != nil {
				return nil, errors.Wrap(err, "creating symbolic link (metadata)")
			}

			// and not give a stream to 7-zip
			return nil, nil
		}

		// so we have a sylink and the linkname is in the content.
		// let's extract to an in-memory buffer and symlink on close
		buf := new(bytes.Buffer)
		nc := &notifyCloser{
			Writer: buf,
			OnClose: func(totalBytes int64) error {
				err := sc.sink.Symlink(entry, buf.String())
				if err != nil {
					return errors.Wrap(err, "creating symbolic link (contents)")
				}

				return nil
			},
		}
		// give the in-memory writer to 7-zip
		return sz.NewOutStream(nc)
	}

	// if we reached this point, it's a regular file
	writer, err := sc.sink.GetWriter(entry)
	if err != nil {
		return nil, errors.Wrap(err, "getting writer for regular file")
	}

	nc := &notifyCloser{
		Writer: writer,
		OnClose: func(totalBytes int64) error {
			if se.saveConsumer.ShouldSave(totalBytes) {
				checkpoint := &savior.ExtractorCheckpoint{
					EntryIndex: entryIndex + 1,
					Progress:   se.progress,
				}
				action, err := se.saveConsumer.Save(checkpoint)
				if err != nil {
					se.consumer.Warnf("7-zip extractor could not save checkpoint: %s", err.Error())
				}

				if action == savior.AfterSaveStop {
					// keep giving nil streams to 7-zip after this
					sc.stopped = true
				}
			}

			return nil
		},
	}

	return sz.NewOutStream(nc)
}

// internal methods

func (se *szExtractor) free() {
	if se.freed {
		return
	}

	if se.archive != nil {
		se.archive.Close()
		se.archive.Free()
		se.archive = nil
	}

	if se.in != nil {
		se.in.Free()
		se.in = nil
	}

	if se.lib != nil {
		se.lib.Free()
		se.lib = nil
	}

	se.freed = true
}

func nameToExt(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if ext != "" {
		ext = ext[1:] // strip "."
	}
	return ext
}

type entryKind int

const (
	// those are reverse-engineered from printing out
	// the 'attrib' param of entries in .zip and .7z files
	entryKindDir     entryKind = 0x4
	entryKindFile    entryKind = 0x8
	entryKindSymlink entryKind = 0xa
)

func szEntry(item *sz.Item) *savior.Entry {
	var kind = entryKindFile
	var mode os.FileMode = 0644

	if attr, ok := item.GetUInt64Property(sz.PidPosixAttrib); ok {
		var kindmask uint64 = 0x0000f000
		var modemask uint64 = 0x000001ff
		kind = entryKind((attr & kindmask) >> (3 * 4))
		mode = os.FileMode(attr&modemask) & archiver.LuckyMode
	}

	if attr, ok := item.GetUInt64Property(sz.PidAttrib); ok {
		var kindmask uint64 = 0xf0000000
		var modemask uint64 = 0x01ff0000
		kind = entryKind((attr & kindmask) >> (7 * 4))
		mode = os.FileMode((attr&modemask)>>(4*4)) & archiver.LuckyMode
	}

	if isDir, _ := item.GetBoolProperty(sz.PidIsDir); isDir {
		kind = entryKindDir
	} else if _, ok := item.GetStringProperty(sz.PidSymLink); ok {
		kind = entryKindSymlink
	}

	switch kind {
	case entryKindDir: // that's ok
	case entryKindFile: // that's ok too
	case entryKindSymlink: // that's fine
	default:
		// uh oh, default to file
		kind = entryKindFile
	}

	if kind == entryKindFile {
		mode |= archiver.ModeMask
	}

	name, _ := item.GetStringProperty(sz.PidPath)
	name = sanitizePath(name)
	uncompressedSize, _ := item.GetUInt64Property(sz.PidSize)
	compressedSize, _ := item.GetUInt64Property(sz.PidPackSize)
	linkname, _ := item.GetStringProperty(sz.PidSymLink)

	entry := &savior.Entry{
		CanonicalPath:    name,
		CompressedSize:   int64(compressedSize),
		UncompressedSize: int64(uncompressedSize),
		Mode:             mode,
		Linkname:         linkname, // will only be set for .tar files
	}

	switch kind {
	case entryKindDir:
		entry.Kind = savior.EntryKindDir
	case entryKindSymlink:
		entry.Kind = savior.EntryKindSymlink
	default:
		entry.Kind = savior.EntryKindFile
	}

	return entry
}

func sanitizePath(inPath string) string {
	outPath := filepath.ToSlash(inPath)

	if runtime.GOOS == "windows" {
		// Replace illegal character for windows paths with underscores, see
		// https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
		// (N.B: that's what the 7-zip CLI seems to do)
		for i := byte(0); i <= 31; i++ {
			outPath = strings.Replace(outPath, string([]byte{i}), "_", -1)
		}
	}

	return outPath
}
