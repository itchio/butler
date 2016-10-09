package pwr

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/pools/zippool"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

type Healer interface {
	WoundsConsumer

	SetNumWorkers(int)
	TotalHealed() int64
}

func NewHealer(spec string, target string) (Healer, error) {
	tokens := strings.SplitN(spec, ",", 2)
	if len(tokens) != 2 {
		return nil, fmt.Errorf("Invalid healer spec: expected 'type,url' but got '%s'", spec)
	}

	healerType := tokens[0]
	healerURL := tokens[1]

	switch healerType {
	case "archive":
		file, err := eos.Open(healerURL)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		ah := &ArchiveHealer{
			File:   file,
			Target: target,
		}
		return ah, nil
	case "manifest":
		return nil, fmt.Errorf("Manifest healer: stub")
	}

	return nil, fmt.Errorf("Unknown healer type %s", healerType)
}

// ArchiveHealer

type ArchiveHealer struct {
	// the directory we should heal
	Target string

	// the file
	File eos.File

	// number of workers running in parallel
	NumWorkers int

	// internal
	totalCorrupted int64
	totalHealed    int64
	hasWounds      bool
}

var _ Healer = (*ArchiveHealer)(nil)

func (ah *ArchiveHealer) Do(container *tlc.Container, wounds chan *Wound) error {
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

	err = container.Prepare(ah.Target)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	targetPool := fspool.New(container, ah.Target)

	errs := make(chan error)
	done := make(chan bool, ah.NumWorkers)
	cancelled := make(chan struct{})

	healed := make(chan int64)
	countingDone := make(chan struct{})
	defer func() {
		close(healed)
		<-countingDone
	}()

	go func() {
		for healedChunk := range healed {
			ah.totalHealed += healedChunk
		}
		close(countingDone)
	}()

	for i := 0; i < ah.NumWorkers; i++ {
		go ah.heal(container, zipReader, stat.Size(), targetPool, fileIndices, errs, done, cancelled, healed)
	}

	processWound := func(wound *Wound) error {
		ah.totalCorrupted += wound.Size()

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

			files[wound.Index] = true

			select {
			case pErr := <-errs:
				return pErr
			case fileIndices <- wound.Index:
				// queued for work!
			}

		default:
			return fmt.Errorf("unknown wound kind: %d", wound.Kind)
		}

		return nil
	}

	for wound := range wounds {
		ah.hasWounds = true

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
	fileIndices chan int64, errs chan error, done chan bool, cancelled chan struct{}, healed chan int64) {

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

			var healedChunk int64
			healedChunk, err = ah.healOne(sourcePool, targetPool, fileIndex)
			if err != nil {
				select {
				case <-cancelled:
					// already cancelled, no need for more errors
					return
				case errs <- err:
					return
				}
			}

			healed <- healedChunk
		}
	}
}

func (ah *ArchiveHealer) healOne(sourcePool wsync.Pool, targetPool wsync.WritablePool, fileIndex int64) (int64, error) {
	var err error
	var reader io.Reader
	var writer io.WriteCloser
	var healedBytes int64

	reader, err = sourcePool.GetReader(fileIndex)
	if err != nil {
		return 0, err
	}

	writer, err = targetPool.GetWriter(fileIndex)
	if err != nil {
		return 0, err
	}

	healedBytes, err = io.Copy(writer, reader)
	if err != nil {
		return healedBytes, err
	}

	err = writer.Close()
	if err != nil {
		return healedBytes, err
	}

	return healedBytes, err
}

func (ah *ArchiveHealer) HasWounds() bool {
	return ah.hasWounds
}

func (ah *ArchiveHealer) TotalCorrupted() int64 {
	return ah.totalCorrupted
}

func (ah *ArchiveHealer) TotalHealed() int64 {
	return ah.totalHealed
}

func (ah *ArchiveHealer) SetNumWorkers(numWorkers int) {
	ah.NumWorkers = numWorkers
}
