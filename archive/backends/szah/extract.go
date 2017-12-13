package szah

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	humanize "github.com/dustin/go-humanize"
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
			consumer.Infof("szah: listing items")
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

					if item.GetBoolProperty(sz.PidIsDir) {
						return
					}

					itemSize := item.GetUInt64Property(sz.PidSize)
					totalUncompressedSize += int64(itemSize)
				}()
			}
			state.TotalUncompressedSize = totalUncompressedSize

			state.HasListedItems = true
			save(state, true)
		} else {
			consumer.Infof("szah: using cached item listing")
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

		consumer.Infof("queued %d/%d items for extraction", len(indices), state.ItemCount)

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

	sanePath := sanitizePath(item.GetStringProperty(sz.PidPath))
	outPath := filepath.Join(e.params.OutputPath, sanePath)

	if item.GetBoolProperty(sz.PidIsDir) {
		err := os.MkdirAll(outPath, 0755)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		e.state.NumDirs++

		return nil, nil
	}

	e.state.NumFiles++

	err := os.MkdirAll(filepath.Dir(outPath), 0755)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	uncompressedSize := item.GetUInt64Property(sz.PidSize)
	consumer.Infof(`→ %s (%s)`, sanePath, humanize.IBytes(uncompressedSize))

	contents := e.state.Contents

	nc := &notifyCloser{
		Writer: f,
		OnClose: func(totalBytes int64) {
			contents.Entries = append(contents.Entries, &archive.Entry{
				Name:             sanePath,
				UncompressedSize: totalBytes,
			})
			// FIXME: it'd be better for GetStream to give us the index of the entry
			// or for Item to have an index getter
			e.state.CurrentIndex++
			e.state.TotalDoneSize += totalBytes
			e.save(e.state, false)
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
	// TODO: do something smart for other formats ?
}

func sanitizePath(inPath string) string {
	outPath := filepath.ToSlash(inPath)

	if runtime.GOOS == "windows" {
		// Remove illegal character for windows paths, see
		// https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
		for i := byte(0); i <= 31; i++ {
			outPath = strings.Replace(outPath, string([]byte{i}), "_", -1)
		}
	}

	return outPath
}
