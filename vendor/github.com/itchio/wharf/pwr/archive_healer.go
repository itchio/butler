package pwr

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/itchio/httpkit/progress"

	"github.com/itchio/wharf/ctxcopy"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/eos/option"
	"github.com/itchio/wharf/werrors"

	"github.com/itchio/arkive/zip"

	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/pools/zippool"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
)

// An ArchiveHealer can repair from a .zip file (remote or local)
type ArchiveHealer struct {
	// the directory we should heal
	Target string

	// an eos path for the archive
	ArchivePath string

	archiveFile    eos.File
	archiveFileErr error
	archiveLock    sync.Mutex
	archiveOnce    sync.Once

	// number of workers running in parallel
	NumWorkers int

	// A consumer to report progress to
	Consumer *state.Consumer

	// internal
	progressMutex  sync.Mutex
	totalCorrupted int64
	totalHealing   int64
	totalHealed    int64
	totalHealthy   int64
	hasWounds      bool

	container *tlc.Container

	lockMap LockMap
}

var _ Healer = (*ArchiveHealer)(nil)

type chunkHealedFunc func(chunkHealed int64)

// Do starts receiving from the wounds channel and healing
func (ah *ArchiveHealer) Do(parentCtx context.Context, container *tlc.Container, wounds chan *Wound) error {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	ah.container = container

	files := make(map[int64]bool)
	fileIndices := make(chan int64, len(container.Files))

	if ah.NumWorkers == 0 {
		// use a sensible default I/O-wise (whether we're reading from disk or network)
		ah.NumWorkers = 2
	}
	if ah.Consumer != nil {
		ah.Consumer.Debugf("archive healer: using %d workers", ah.NumWorkers)
	}

	targetPool := fspool.New(container, ah.Target)

	errs := make(chan error, ah.NumWorkers)

	onChunkHealed := func(healedChunk int64) {
		ah.progressMutex.Lock()
		ah.totalHealed += healedChunk
		ah.progressMutex.Unlock()
		ah.updateProgress()
	}

	defer func() {
		if ah.archiveFile != nil {
			ah.archiveFile.Close()
		}
	}()

	for i := 0; i < ah.NumWorkers; i++ {
		go func() {
			errs <- ah.heal(ctx, container, targetPool, fileIndices, onChunkHealed)
		}()
	}

	processWound := func(wound *Wound) error {
		if ah.Consumer != nil {
			ah.Consumer.Debugf("processing wound: %s", wound)
		}

		if !wound.Healthy() {
			ah.totalCorrupted += wound.Size()
			ah.hasWounds = true
		}

		switch wound.Kind {
		case WoundKind_DIR:
			dirEntry := container.Dirs[wound.Index]
			path := filepath.Join(ah.Target, filepath.FromSlash(dirEntry.Path))

			pErr := os.MkdirAll(path, 0755)
			if pErr != nil {
				return pErr
			}

		case WoundKind_SYMLINK:
			symlinkEntry := container.Symlinks[wound.Index]
			path := filepath.Join(ah.Target, filepath.FromSlash(symlinkEntry.Path))

			dir := filepath.Dir(path)
			pErr := os.MkdirAll(dir, 0755)
			if pErr != nil {
				return pErr
			}

			pErr = os.Symlink(symlinkEntry.Dest, path)
			if pErr != nil {
				return pErr
			}

		case WoundKind_FILE:
			if files[wound.Index] {
				// already queued
				return nil
			}

			file := container.Files[wound.Index]
			if ah.Consumer != nil {
				ah.Consumer.ProgressLabel(file.Path)
			}

			ah.progressMutex.Lock()
			ah.totalHealing += file.Size
			ah.progressMutex.Unlock()
			ah.updateProgress()
			files[wound.Index] = true

			select {
			case pErr := <-errs:
				return pErr
			case fileIndices <- wound.Index:
				// queued for work!
			}

		case WoundKind_CLOSED_FILE:
			if files[wound.Index] {
				// already healing whole file
			} else {
				fileSize := container.Files[wound.Index].Size

				// whole file was healthy
				if wound.End == fileSize {
					ah.progressMutex.Lock()
					ah.totalHealthy += fileSize
					ah.progressMutex.Unlock()
					ah.updateProgress()
				}
			}

		default:
			return fmt.Errorf("unknown wound kind: %d", wound.Kind)
		}

		return nil
	}

	for wound := range wounds {
		select {
		case <-ctx.Done():
			return werrors.ErrCancelled
		default:
			// keep going!
		}

		err := processWound(wound)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// queued everything
	close(fileIndices)

	// expecting up to NumWorkers done, some may still
	// send errors
	for i := 0; i < ah.NumWorkers; i++ {
		err := <-errs
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (ah *ArchiveHealer) openArchive() (eos.File, error) {
	ah.archiveLock.Lock()
	defer ah.archiveLock.Unlock()

	ah.archiveOnce.Do(func() {
		if ah.Consumer != nil {
			ah.Consumer.Debugf("opening archive for worker!")
		}

		file, err := eos.Open(ah.ArchivePath, option.WithConsumer(ah.Consumer))
		ah.archiveFile = file
		ah.archiveFileErr = err
	})
	return ah.archiveFile, ah.archiveFileErr
}

func (ah *ArchiveHealer) heal(ctx context.Context, container *tlc.Container, targetPool wsync.WritablePool,
	fileIndices chan int64, chunkHealed chunkHealedFunc) error {

	var sourcePool wsync.Pool
	var err error

	for {
		select {
		case <-ctx.Done():
			// something else stopped the healing
			return nil
		case fileIndex, ok := <-fileIndices:
			if !ok {
				// no more files to heal
				return nil
			}

			// lazily open file
			if sourcePool == nil {
				file, err := ah.openArchive()
				if err != nil {
					return errors.WithStack(err)
				}

				stat, err := file.Stat()
				if err != nil {
					return err
				}

				zipReader, err := zip.NewReader(file, stat.Size())
				if err != nil {
					return errors.WithStack(err)
				}

				sourcePool = zippool.New(container, zipReader)
				// sic: we're inside a for, not a function, so this correctly happens
				// when we actually return
				defer sourcePool.Close()
			}

			err = ah.healOne(ctx, sourcePool, targetPool, fileIndex, chunkHealed)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}
}

func (ah *ArchiveHealer) healOne(ctx context.Context, sourcePool wsync.Pool, targetPool wsync.WritablePool, fileIndex int64, chunkHealed chunkHealedFunc) error {
	if ah.lockMap != nil {
		lock := ah.lockMap[fileIndex]
		select {
		case <-lock:
			// keep going
		case <-ctx.Done():
			return werrors.ErrCancelled
		}
	}

	var err error
	var reader io.Reader
	var writer io.WriteCloser

	if ah.Consumer != nil {
		f := ah.container.Files[fileIndex]
		ah.Consumer.Debugf("healing (%s) %s", f.Path, progress.FormatBytes(f.Size))
	}

	reader, err = sourcePool.GetReader(fileIndex)
	if err != nil {
		return err
	}

	writer, err = targetPool.GetWriter(fileIndex)
	if err != nil {
		return err
	}
	defer writer.Close()

	lastCount := int64(0)
	cw := counter.NewWriterCallback(func(count int64) {
		chunk := count - lastCount
		chunkHealed(chunk)
		lastCount = count
	}, writer)

	_, err = ctxcopy.Do(ctx, cw, reader)
	if err != nil {
		return err
	}

	return err
}

// HasWounds returns true if the healer ever received wounds
func (ah *ArchiveHealer) HasWounds() bool {
	return ah.hasWounds
}

// TotalCorrupted returns the total amount of corrupted data
// contained in the wounds this healer has received. Dirs
// and symlink wounds have 0-size, use HasWounds to know
// if there were any wounds at all.
func (ah *ArchiveHealer) TotalCorrupted() int64 {
	return ah.totalCorrupted
}

// TotalHealed returns the total amount of data written to disk
// to repair the wounds. This might be more than TotalCorrupted,
// since ArchiveHealer always redownloads whole files, even if
// they're just partly corrupted
func (ah *ArchiveHealer) TotalHealed() int64 {
	return ah.totalHealed
}

// SetNumWorkers may be called before Do to adjust the concurrency
// of ArchiveHealer (how many files it'll try to heal in parallel)
func (ah *ArchiveHealer) SetNumWorkers(numWorkers int) {
	ah.NumWorkers = numWorkers
}

// SetConsumer gives this healer a consumer to report progress to
func (ah *ArchiveHealer) SetConsumer(consumer *state.Consumer) {
	ah.Consumer = consumer
}

func (ah *ArchiveHealer) updateProgress() {
	if ah.Consumer == nil {
		return
	}

	ah.progressMutex.Lock()
	progress := float64(ah.totalHealthy+ah.totalHealed) / float64(ah.container.Size)
	ah.Consumer.Progress(progress)
	ah.progressMutex.Unlock()
}

func (ah *ArchiveHealer) SetLockMap(lockMap LockMap) {
	ah.lockMap = lockMap
}
