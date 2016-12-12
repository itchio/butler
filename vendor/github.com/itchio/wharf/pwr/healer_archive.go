package pwr

import (
	"github.com/itchio/arkive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/pools/zippool"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

// An ArchiveHealer can repair from a .zip file (remote or local)
type ArchiveHealer struct {
	// the directory we should heal
	Target string

	// the file
	File eos.File

	// number of workers running in parallel
	NumWorkers int

	// A consumer to report progress to
	Consumer *state.Consumer

	// internal
	totalCorrupted int64
	totalHealing   int64
	totalHealed    int64
	totalHealthy   int64
	hasWounds      bool

	container *tlc.Container
}

var _ Healer = (*ArchiveHealer)(nil)

type chunkHealedFunc func(chunkHealed int64)

// Do starts receiving from the wounds channel and healing
func (ah *ArchiveHealer) Do(container *tlc.Container, wounds chan *Wound) error {
	ah.container = container

	files := make(map[int64]bool)
	fileIndices := make(chan int64)

	if ah.NumWorkers == 0 {
		ah.NumWorkers = runtime.NumCPU() + 1
	}

	defer ah.File.Close()

	stat, err := ah.File.Stat()
	if err != nil {
		return err
	}

	zipReader, err := zip.NewReader(ah.File, stat.Size())
	if err != nil {
		return errors.Wrap(err, 1)
	}

	targetPool := fspool.New(container, ah.Target)

	errs := make(chan error)
	done := make(chan bool, ah.NumWorkers)
	cancelled := make(chan struct{})

	onChunkHealed := func(healedChunk int64) {
		atomic.AddInt64(&ah.totalHealed, healedChunk)
		ah.updateProgress()
	}

	for i := 0; i < ah.NumWorkers; i++ {
		go ah.heal(container, zipReader, stat.Size(), targetPool, fileIndices, errs, done, cancelled, onChunkHealed)
	}

	processWound := func(wound *Wound) error {
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

			atomic.AddInt64(&ah.totalHealing, file.Size)
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
					atomic.AddInt64(&ah.totalHealthy, fileSize)
				}
			}

		default:
			return fmt.Errorf("unknown wound kind: %d", wound.Kind)
		}

		return nil
	}

	for wound := range wounds {
		err = processWound(wound)
		if err != nil {
			close(fileIndices)
			close(cancelled)
			return errors.Wrap(err, 1)
		}
	}

	// queued everything
	close(fileIndices)

	// expecting up to NumWorkers done, some may still
	// send errors
	for i := 0; i < ah.NumWorkers; i++ {
		select {
		case err = <-errs:
			close(cancelled)
			return errors.Wrap(err, 1)
		case <-done:
			// good!
		}
	}

	return nil
}

func (ah *ArchiveHealer) heal(container *tlc.Container, zipReader *zip.Reader, zipSize int64,
	targetPool wsync.WritablePool,
	fileIndices chan int64, errs chan error, done chan bool, cancelled chan struct{}, chunkHealed chunkHealedFunc) {

	var sourcePool wsync.Pool
	var err error

	sourcePool = zippool.New(container, zipReader)
	defer sourcePool.Close()

	for {
		select {
		case <-cancelled:
			// something else stopped the healing
			return
		case fileIndex, ok := <-fileIndices:
			if !ok {
				// no more files to heal
				done <- true
				return
			}

			err = ah.healOne(sourcePool, targetPool, fileIndex, chunkHealed)
			if err != nil {
				select {
				case <-cancelled:
					// already cancelled, no need for more errors
					return
				case errs <- err:
					return
				}
			}
		}
	}
}

func (ah *ArchiveHealer) healOne(sourcePool wsync.Pool, targetPool wsync.WritablePool, fileIndex int64, chunkHealed chunkHealedFunc) error {
	var err error
	var reader io.Reader
	var writer io.WriteCloser

	reader, err = sourcePool.GetReader(fileIndex)
	if err != nil {
		return err
	}

	writer, err = targetPool.GetWriter(fileIndex)
	if err != nil {
		return err
	}

	lastCount := int64(0)
	cw := counter.NewWriterCallback(func(count int64) {
		chunk := count - lastCount
		chunkHealed(chunk)
		lastCount = count
	}, writer)

	_, err = io.Copy(cw, reader)
	if err != nil {
		return err
	}

	err = writer.Close()
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

	totalHealthy := atomic.LoadInt64(&ah.totalHealthy)
	totalHealed := atomic.LoadInt64(&ah.totalHealed)

	progress := float64(totalHealthy+totalHealed) / float64(ah.container.Size)
	ah.Consumer.Progress(progress)
}
