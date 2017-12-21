package bah

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/itchio/wharf/counter"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/arkive/zip"
	"github.com/itchio/kompress/flate"
	"github.com/itchio/wharf/archiver"

	"github.com/itchio/butler/archive"
)

type ExtractState struct {
	HasListedItems        bool
	ItemCount             int64
	TotalDoneSize         int64
	TotalUncompressedSize int64
	CurrentIndex          int64
	Contents              *archive.Contents
	TotalCheckpoints      int64

	NumFiles    int64
	NumDirs     int64
	NumSymlinks int64

	FlateCheckpoint *flate.Checkpoint
	StoreCheckpoint *StoreCheckpoint
}

// buffer size when extracting flate entries
const bufSize = 64 * 1024

// we try to make a checkpoint after reading `threshold` bytes
// it's usually a few more KBs before we can make one
const threshold int64 = 4 * 1024 * 1024

type StoreCheckpoint struct {
	// Store method is just uncompressed files, so the
	// read and write offsets are the same
	Offset int64
}

func (es *ExtractState) ResetCheckpoints() {
	es.FlateCheckpoint = nil
	es.StoreCheckpoint = nil
}

func (h *Handler) Extract(params *archive.ExtractParams) (*archive.Contents, error) {
	save := archive.ThrottledSave(params)
	consumer := params.Consumer

	state := &ExtractState{
		Contents: &archive.Contents{},
	}

	err := params.Load(state)
	if err != nil {
		consumer.Infof("bah: could not load state: %s", err.Error())
		consumer.Infof("bah: ...starting from beginning!")
	}

	stats, err := params.File.Stat()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	reader, err := zip.NewReader(params.File, stats.Size())
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if !state.HasListedItems {
		for _, zf := range reader.File {
			state.TotalUncompressedSize += int64(zf.UncompressedSize64)
			state.ItemCount++
		}

		state.HasListedItems = true
		save(state, true)
	}

	windows := runtime.GOOS == "windows"

	for state.CurrentIndex < state.ItemCount {
		zf := reader.File[state.CurrentIndex]
		zei := getZipEntryInfo(params, zf)

		info := zf.FileInfo()
		wasNormalFile := false

		if info.IsDir() {
			state.NumDirs++

			err := archiver.Mkdir(zei.FileName)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}
		} else if zei.Mode&os.ModeSymlink > 0 && !windows {
			state.NumSymlinks++

			err := doSymlink(params, zf, zei)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}
		} else {
			wasNormalFile = true
			state.NumFiles++

			consumer.Debugf(`→ %s (%s)`, zei.CanonicalName, humanize.IBytes(uint64(zf.UncompressedSize64)))

			switch zf.Method {
			// we know how to save/resume that!
			case zip.Deflate:
				err = extractDeflate(save, params, state, zf, zei)
				if err != nil {
					return nil, errors.Wrap(err, 0)
				}

			// we know how to save/resume that!
			case zip.Store:
				err = extractStore(save, params, state, zf, zei)
				if err != nil {
					return nil, errors.Wrap(err, 0)
				}

			// we don't know how to save/resume that!
			default:
				err = extractOther(save, params, state, zf, zei)
				if err != nil {
					return nil, errors.Wrap(err, 0)
				}
			}
		}

		if wasNormalFile {
			// we don't list directories/symlinks
			state.Contents.Entries = append(state.Contents.Entries, &archive.Entry{
				Name:             zei.CanonicalName,
				UncompressedSize: int64(zf.UncompressedSize64),
			})
			state.TotalDoneSize += int64(zf.UncompressedSize64)
		}
		state.CurrentIndex++
		state.ResetCheckpoints()
		save(state, false)
	}

	consumer.Statf("extracted %d items successfully", state.ItemCount)
	consumer.Statf("%d files (%d flate checkpoints), %d dirs, %d symlinks", state.NumFiles, state.TotalCheckpoints, state.NumDirs, state.NumSymlinks)

	return state.Contents, nil
}

type ZipEntryInfo struct {
	CanonicalName string
	FileName      string
	Mode          os.FileMode
}

func getZipEntryInfo(params *archive.ExtractParams, zf *zip.File) *ZipEntryInfo {
	filename := filepath.Join(params.OutputPath, filepath.FromSlash(zf.Name))
	info := zf.FileInfo()

	return &ZipEntryInfo{
		CanonicalName: filepath.ToSlash(zf.Name),
		FileName:      filename,
		Mode:          info.Mode(),
	}
}

func extractDeflate(save archive.ThrottledSaveFunc, params *archive.ExtractParams, state *ExtractState, zf *zip.File, zei *ZipEntryInfo) error {
	consumer := params.Consumer

	var fr flate.SaverReader
	var err error
	var offset int64

	sr, err := getSectionReader(params, zf)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if state.FlateCheckpoint != nil {
		_, err = sr.Seek(state.FlateCheckpoint.Roffset, io.SeekStart)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		fr, err = state.FlateCheckpoint.Resume(sr)
		if err != nil {
			consumer.Warnf("bah: Could not resume from flate checkpoint: %s", err.Error())
		} else {
			// cool, we resumed!
			offset = state.FlateCheckpoint.Woffset
		}
	}

	if fr == nil {
		fr = flate.NewSaverReader(sr)
	}

	writer, err := getWriter(params, zf, zei, offset)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer writer.Close()

	cw := getCountingWriter(params, state, zf, writer, offset)

	var readBytes int64
	buf := make([]byte, bufSize)

	for {
		n, err := fr.Read(buf)
		if n > 0 {
			readBytes += int64(n)
			_, wErr := cw.Write(buf[:n])
			if wErr != nil {
				return errors.Wrap(wErr, 0)
			}
		}

		if err != nil {
			if err == io.EOF {
				// we're done!
				break
			} else if err == flate.ReadyToSaveError {
				// ooh.
				c, err := fr.Save()
				if err != nil {
					consumer.Warnf("bah: Could not make flate checkpoint: ", err.Error())
					// oh well, keep moving
				} else {
					err = writer.Sync()
					if err != nil {
						return errors.Wrap(err, 0)
					}

					state.FlateCheckpoint = c
					state.TotalCheckpoints++
					if save(state, false) {
						consumer.Debugf("↓ Saved (flate) %s / %s into %s", humanize.IBytes(uint64(c.Woffset)), humanize.IBytes(uint64(zf.UncompressedSize64)), path.Base(zei.CanonicalName))
					}
				}
			} else {
				// an actual error!
				return errors.Wrap(err, 0)
			}
		}

		if readBytes > threshold {
			fr.WantSave()
			readBytes = 0
		}
	}

	// we did it!
	return nil
}

func extractStore(save archive.ThrottledSaveFunc, params *archive.ExtractParams, state *ExtractState, zf *zip.File, zei *ZipEntryInfo) error {
	consumer := params.Consumer

	sr, err := getSectionReader(params, zf)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	// The `Store` method has no compression, so the
	// read and write offsets are the same
	var offset int64
	if state.StoreCheckpoint != nil {
		offset = state.StoreCheckpoint.Offset

		_, err := sr.Seek(offset, io.SeekStart)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	writer, err := getWriter(params, zf, zei, offset)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	cw := getCountingWriter(params, state, zf, writer, offset)

	var readBytes int64
	buf := make([]byte, bufSize)
	for {
		n, err := sr.Read(buf)
		if n > 0 {
			readBytes += int64(n)
			offset += int64(n)

			_, wErr := cw.Write(buf[:n])
			if wErr != nil {
				return errors.Wrap(wErr, 0)
			}
		}

		if err != nil {
			if err == io.EOF {
				// we're done
				break
			} else {
				// an actual error!
				return errors.Wrap(err, 0)
			}
		}

		if readBytes > threshold {
			err = writer.Sync()
			if err != nil {
				return errors.Wrap(err, 0)
			}

			state.StoreCheckpoint = &StoreCheckpoint{
				Offset: offset,
			}
			if save(state, false) {
				consumer.Debugf("↓ Saved (store) %s / %s into %s", humanize.IBytes(uint64(offset)), humanize.IBytes(uint64(zf.UncompressedSize64)), path.Base(zei.CanonicalName))
			}

			readBytes = 0
		}
	}

	// we did it!
	return nil
}

func extractOther(save archive.ThrottledSaveFunc, params *archive.ExtractParams, state *ExtractState, zf *zip.File, zei *ZipEntryInfo) error {
	// extracting an entry which we don't know how to save/resume, so we'll have
	// to start over from the start of that entry if we're interrupted
	reader, err := zf.Open()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	writer, err := getWriter(params, zf, zei, 0)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer writer.Close()

	cw := getCountingWriter(params, state, zf, writer, 0)

	_, err = io.Copy(cw, reader)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func doSymlink(params *archive.ExtractParams, zf *zip.File, zei *ZipEntryInfo) error {
	consumer := params.Consumer

	rc, err := zf.Open()
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer rc.Close()

	linkname, err := ioutil.ReadAll(rc)
	if err != nil {
		consumer.Warnf("bah: error while reading symlink (skipping): %s", err.Error())
		return nil
	}

	err = archiver.Symlink(string(linkname), zei.FileName, params.Consumer)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func getWriter(params *archive.ExtractParams, zf *zip.File, zei *ZipEntryInfo, woffset int64) (*os.File, error) {
	parent := filepath.Dir(zei.FileName)
	err := archiver.Mkdir(parent)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	flag := os.O_CREATE | os.O_WRONLY
	writer, err := os.OpenFile(zei.FileName, flag, zei.Mode|archiver.ModeMask)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	// just make sure the file has the right mode
	// (if it already did exist, it would keep its old mode)
	err = os.Chmod(zei.FileName, zei.Mode|archiver.ModeMask)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = writer.Truncate(int64(zf.UncompressedSize64))
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if woffset != 0 {
		params.Consumer.Infof("↺ Resuming %s / %s into %s", humanize.IBytes(uint64(woffset)), humanize.IBytes(uint64(zf.UncompressedSize64)), path.Base(zei.CanonicalName))
		_, err := writer.Seek(woffset, io.SeekStart)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	return writer, nil
}

func getCountingWriter(params *archive.ExtractParams, state *ExtractState, zf *zip.File, w io.Writer, woffset int64) io.Writer {
	var totalBytes = state.TotalUncompressedSize
	var initialDoneBytes = state.TotalDoneSize + woffset
	var doneBytes = initialDoneBytes

	sendProgress := func() {
		progress := float64(doneBytes) / float64(totalBytes)
		params.Consumer.Progress(progress)
	}

	return counter.NewWriterCallback(func(newCount int64) {
		doneBytes = initialDoneBytes + newCount
		sendProgress()
	}, w)
}

func getSectionReader(params *archive.ExtractParams, zf *zip.File) (*io.SectionReader, error) {
	dataOffset, err := zf.DataOffset()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	compressedSize := int64(zf.CompressedSize64)
	sr := io.NewSectionReader(params.File, dataOffset, compressedSize)
	return sr, nil
}
