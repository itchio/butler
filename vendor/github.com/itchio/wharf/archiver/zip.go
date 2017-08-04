package archiver

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"

	"github.com/itchio/arkive/zip"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/state"
)

func ExtractZip(readerAt io.ReaderAt, size int64, dir string, settings ExtractSettings) (*ExtractResult, error) {
	dirCount := 0
	regCount := 0
	symlinkCount := 0

	reader, err := zip.NewReader(readerAt, size)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	var totalSize int64
	for _, file := range reader.File {
		totalSize += int64(file.UncompressedSize64)
	}

	var doneSize uint64
	var lastDoneIndex int = -1

	func() {
		if settings.ResumeFrom == "" {
			return
		}

		resBytes, resErr := ioutil.ReadFile(settings.ResumeFrom)
		if resErr != nil {
			if !errors.Is(resErr, os.ErrNotExist) {
				settings.Consumer.Warnf("Couldn't read resume file: %s", resErr.Error())
			}
			return
		}

		lastDone64, resErr := strconv.ParseInt(string(resBytes), 10, 64)
		if resErr != nil {
			settings.Consumer.Warnf("Couldn't parse resume file: %s", resErr.Error())
			return
		}

		lastDoneIndex = int(lastDone64)
		settings.Consumer.Infof("Resuming from file %d", lastDoneIndex)
	}()

	warnedAboutWrite := false

	writeProgress := func(fileIndex int) {
		if settings.ResumeFrom == "" {
			return
		}

		payload := fmt.Sprintf("%d", fileIndex)

		wErr := ioutil.WriteFile(settings.ResumeFrom, []byte(payload), 0644)
		if wErr != nil {
			if !warnedAboutWrite {
				warnedAboutWrite = true
				settings.Consumer.Warnf("Couldn't save resume file: %s", wErr.Error())
			}
			return
		}
	}

	defer func() {
		if settings.ResumeFrom == "" {
			return
		}

		rErr := os.Remove(settings.ResumeFrom)
		if rErr != nil {
			settings.Consumer.Warnf("Couldn't remove resume file: %s", rErr.Error())
		}
	}()

	if settings.OnUncompressedSizeKnown != nil {
		settings.OnUncompressedSizeKnown(totalSize)
	}

	windows := runtime.GOOS == "windows"

	numWorkers := settings.Concurrency
	if numWorkers < 0 {
		numWorkers = runtime.NumCPU() - 1
	}
	if numWorkers < 1 {
		numWorkers = 1
	}
	settings.Consumer.Infof("Using %d workers", numWorkers)

	fileIndices := make(chan int)
	errs := make(chan error, numWorkers)

	updateProgress := func() {
		ds := atomic.LoadUint64(&doneSize)
		settings.Consumer.Progress(float64(ds) / float64(totalSize))
	}

	for i := 0; i < numWorkers; i++ {
		go func() {
			reader, err := zip.NewReader(readerAt, size)
			if err != nil {
				errs <- errors.Wrap(err, 1)
				return
			}

			for fileIndex := range fileIndices {
				file := reader.File[fileIndex]

				if fileIndex <= lastDoneIndex {
					settings.Consumer.Debugf("Skipping file %d")
					if settings.OnEntryDone != nil {
						settings.OnEntryDone(filepath.ToSlash(file.Name))
					}
					atomic.AddUint64(&doneSize, file.UncompressedSize64)
					updateProgress()
					continue
				}

				err = func() error {
					rel := file.Name
					filename := path.Join(dir, filepath.FromSlash(rel))

					info := file.FileInfo()
					mode := info.Mode()

					if info.IsDir() {
						if settings.DryRun {
							// muffin
						} else {
							err = Mkdir(filename)
							if err != nil {
								return errors.Wrap(err, 1)
							}
						}
						dirCount++
					} else if mode&os.ModeSymlink > 0 && !windows {
						fileReader, fErr := file.Open()
						if fErr != nil {
							return errors.Wrap(fErr, 1)
						}
						defer fileReader.Close()

						linkname, lErr := ioutil.ReadAll(fileReader)
						if settings.DryRun {
							// muffin
						} else {
							lErr = Symlink(string(linkname), filename, settings.Consumer)
							if lErr != nil {
								return errors.Wrap(lErr, 1)
							}
						}
						symlinkCount++
					} else {
						regCount++

						fileReader, fErr := file.Open()
						if fErr != nil {
							return errors.Wrap(fErr, 1)
						}
						defer fileReader.Close()

						settings.Consumer.Debugf("extract %s", filename)
						var lastOffset int64
						countingReader := counter.NewReaderCallback(func(offset int64) {
							doneRecently := offset - lastOffset
							lastOffset = offset
							atomic.AddUint64(&doneSize, uint64(doneRecently))
							updateProgress()
						}, fileReader)

						if settings.DryRun {
							_, err = io.Copy(ioutil.Discard, countingReader)
							if err != nil {
								return errors.Wrap(err, 1)
							}
						} else {
							err = CopyFile(filename, os.FileMode(mode&LuckyMode|ModeMask), countingReader)
							if err != nil {
								return errors.Wrap(err, 1)
							}
						}
					}

					return nil
				}()
				if err != nil {
					errs <- errors.Wrap(err, 1)
					return
				}
				writeProgress(fileIndex)

				if settings.OnEntryDone != nil {
					settings.OnEntryDone(filepath.ToSlash(file.Name))
				}
			}

			errs <- nil
		}()
	}

	for fileIndex := range reader.File {
		select {
		case fileIndices <- fileIndex:
			// sent work, yay!
		case err := <-errs:
			// abort everything
			close(fileIndices)
			return nil, err
		}
	}

	close(fileIndices)
	for i := 0; i < numWorkers; i++ {
		err := <-errs
		if err != nil {
			return nil, err
		}
	}

	return &ExtractResult{
		Dirs:     dirCount,
		Files:    regCount,
		Symlinks: symlinkCount,
	}, nil
}

func CompressZip(archiveWriter io.Writer, dir string, consumer *state.Consumer) (*CompressResult, error) {
	var err error
	var uncompressedSize int64
	var compressedSize int64

	archiveCounter := counter.NewWriter(archiveWriter)

	zipWriter := zip.NewWriter(archiveCounter)
	defer zipWriter.Close()
	defer func() {
		if zipWriter != nil {
			if zErr := zipWriter.Close(); err == nil && zErr != nil {
				err = errors.Wrap(zErr, 1)
			}
		}
	}()

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		name, wErr := filepath.Rel(dir, path)
		if wErr != nil {
			return wErr
		}

		if name == "." {
			// don't add '.' to zip
			return nil
		}

		name = filepath.ToSlash(name)

		fh, wErr := zip.FileInfoHeader(info)
		if wErr != nil {
			return wErr
		}

		fh.Name = name

		writer, wErr := zipWriter.CreateHeader(fh)
		if wErr != nil {
			return wErr
		}

		if info.IsDir() {
			// good!
		} else if info.Mode()&os.ModeSymlink > 0 {
			dest, wErr := os.Readlink(path)
			if wErr != nil {
				return wErr
			}

			_, wErr = writer.Write([]byte(dest))
			if wErr != nil {
				return wErr
			}
		} else if info.Mode().IsRegular() {
			reader, wErr := os.Open(path)
			if wErr != nil {
				return wErr
			}
			defer reader.Close()

			copiedBytes, wErr := io.Copy(writer, reader)
			if wErr != nil {
				return wErr
			}

			uncompressedSize += copiedBytes
		}

		return nil
	})

	err = zipWriter.Close()
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	zipWriter = nil

	compressedSize = archiveCounter.Count()

	return &CompressResult{
		UncompressedSize: uncompressedSize,
		CompressedSize:   compressedSize,
	}, err
}
