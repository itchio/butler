package szah

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/wharf/archiver"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
	"github.com/itchio/sevenzip-go/sz"
)

type ExtractState struct {
	HasListedItems        bool
	ItemCount             int64
	TotalDoneSize         int64
	TotalUncompressedSize int64
	CurrentIndex          int64
	Contents              *archive.Contents

	NumFiles    int64
	NumDirs     int64
	NumSymlinks int64
}

type ech struct {
	params          *archive.ExtractParams
	initialProgress float64
	state           *ExtractState
	save            archive.ThrottledSaveFunc
}

func (h *Handler) Extract(params *archive.ExtractParams) (*archive.Contents, error) {
	save := archive.ThrottledSave(params)
	consumer := params.Consumer
	state := &ExtractState{
		Contents: &archive.Contents{},
	}

	err := withArchive(params.Consumer, params.File, func(a *sz.Archive) error {
		err := params.Load(state)
		if err != nil {
			consumer.Infof("szah: could not load state: %s", err.Error())
			consumer.Infof("szah: ...starting from beginning!")
		}

		if !state.HasListedItems {
			consumer.Infof("Listing items...")
			itemCount, err := a.GetItemCount()
			if err != nil {
				return errors.Wrap(err, 0)
			}
			state.ItemCount = itemCount

			var totalUncompressedSize int64
			for i := int64(0); i < itemCount; i++ {
				func() {
					item := a.GetItem(i)
					if item == nil {
						return
					}
					defer item.Free()

					ei := decodeEntryInfo(item)
					if ei.kind == entryKindFile {
						if itemSize, ok := item.GetUInt64Property(sz.PidSize); ok {
							// if we can't get the item size well.. that's not great
							// but it shouldn't impede anything.
							totalUncompressedSize += int64(itemSize)
						}
					}
				}()
			}
			state.TotalUncompressedSize = totalUncompressedSize

			state.HasListedItems = true
			save(state, true)
		} else {
			consumer.Infof("Using cached item listing")
		}

		if params.OnUncompressedSizeKnown != nil {
			params.OnUncompressedSizeKnown(state.TotalUncompressedSize)
		}

		ec, err := sz.NewExtractCallback(&ech{
			params:          params,
			state:           state,
			initialProgress: float64(state.TotalDoneSize) / float64(state.TotalUncompressedSize),
			save:            save,
		})
		if err != nil {
			return errors.Wrap(err, 0)
		}
		defer ec.Free()

		var indices []int64
		for i := state.CurrentIndex; i < state.ItemCount; i++ {
			indices = append(indices, i)
		}
		if len(indices) == 0 {
			consumer.Infof("nothing (0 items) to extract!")
			return nil
		}

		consumer.Infof("Queued %d / %d items for extraction", len(indices), state.ItemCount)

		err = a.ExtractSeveral(indices, ec)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Statf("extracted %d items successfully", state.ItemCount)
	consumer.Statf("%d files, %d dirs, %d symlinks", state.NumFiles, state.NumDirs, state.NumSymlinks)

	return state.Contents, nil
}

func (e *ech) GetStream(item *sz.Item) (*sz.OutStream, error) {
	consumer := e.params.Consumer
	itemIndex := item.GetArchiveIndex()

	itemPath, ok := item.GetStringProperty(sz.PidPath)
	if !ok {
		return nil, errors.New("can't get item path")
	}

	sanePath := sanitizePath(itemPath)
	outPath := filepath.Join(e.params.OutputPath, sanePath)

	ei := decodeEntryInfo(item)

	contents := e.state.Contents
	finish := func(totalBytes int64, createEntry bool) {
		if createEntry {
			contents.Entries = append(contents.Entries, &archive.Entry{
				Name:             sanePath,
				UncompressedSize: totalBytes,
			})
		}

		e.state.CurrentIndex = itemIndex + 1
		e.state.TotalDoneSize += totalBytes
		e.save(e.state, false)
	}

	windows := runtime.GOOS == "windows"

	if ei.kind == entryKindDir {
		e.state.NumDirs++

		err := os.MkdirAll(outPath, 0755)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		finish(0, false)

		// giving 7-zip a null stream will make it skip the entry
		return nil, nil
	}

	if ei.kind == entryKindSymlink && !windows {
		e.state.NumSymlinks++

		// is the link name stored as a property?
		if linkname, ok := item.GetStringProperty(sz.PidSymLink); ok {
			// cool!
			err := archiver.Symlink(linkname, outPath, consumer)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}

			finish(0, false)
			return nil, nil
		}

		// the link name is stored as the file contents, so
		// we extract to an in-memory buffer
		buf := new(bytes.Buffer)
		nc := &notifyCloser{
			Writer: buf,
			OnClose: func(totalBytes int64) error {
				linkname := buf.Bytes()

				err := archiver.Symlink(string(linkname), outPath, consumer)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				finish(totalBytes, false)
				return nil
			},
		}

		return sz.NewOutStream(nc)
	}

	// if we end up here, it's a regular file
	e.state.NumFiles++

	uncompressedSize, _ := item.GetUInt64Property(sz.PidSize)
	consumer.Infof(`→ %s (%s)`, sanePath, humanize.IBytes(uncompressedSize))

	err := os.MkdirAll(filepath.Dir(outPath), 0755)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	flag := os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	f, err := os.OpenFile(outPath, flag, ei.mode)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	nc := &notifyCloser{
		Writer: f,
		OnClose: func(totalBytes int64) error {
			finish(totalBytes, true)
			return nil
		},
	}
	return sz.NewOutStream(nc)
}

func (e *ech) SetProgress(complete int64, total int64) {
	if total > 0 {
		thisRunProgress := float64(complete) / float64(total)
		actualProgress := e.initialProgress + (1.0-e.initialProgress)*thisRunProgress
		e.params.Consumer.Progress(actualProgress)
	}
	// TODO: some formats don't have 'total' value, should we do
	// something smart there?
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
