package pwr

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pools"
	"github.com/itchio/wharf/pools/nullpool"
	"github.com/itchio/wharf/state"
)

const MaxWoundSize int64 = 4 * 1024 * 1024 // 4MB

type ValidatorContext struct {
	WoundsPath string
	NumWorkers int

	Consumer *state.Consumer

	// FailFast makes Validate return Wounds as errors and stop checking
	FailFast bool

	// Result
	TotalCorrupted int64

	// internal
	Wounds         chan *Wound
	WoundsConsumer WoundsConsumer
}

func (vctx *ValidatorContext) Validate(target string, signature *SignatureInfo) error {
	if vctx.Consumer == nil {
		vctx.Consumer = &state.Consumer{}
	}

	numWorkers := vctx.NumWorkers
	if numWorkers == 0 {
		numWorkers = runtime.NumCPU() + 1
	}

	vctx.Wounds = make(chan *Wound)
	workerErrs := make(chan error, numWorkers)
	consumerErrs := make(chan error, 1)
	cancelled := make(chan struct{})

	if vctx.FailFast {
		if vctx.WoundsPath != "" {
			return fmt.Errorf("Validate: FailFast is not compatible with WoundsPath")
		}

		vctx.WoundsConsumer = &WoundsGuardian{}
	} else if vctx.WoundsPath == "" {
		vctx.WoundsConsumer = &WoundsPrinter{
			Consumer: vctx.Consumer,
		}
	} else {
		vctx.WoundsConsumer = &WoundsWriter{
			WoundsPath: vctx.WoundsPath,
		}
	}

	go func() {
		consumerErrs <- vctx.WoundsConsumer.Do(signature.Container, vctx.Wounds)

		// throw away wounds until closed
		for {
			select {
			case _, ok := <-vctx.Wounds:
				if !ok {
					return
				}
			}
		}
	}()

	bytesDone := int64(0)
	onProgress := func(delta int64) {
		atomic.AddInt64(&bytesDone, delta)
		vctx.Consumer.Progress(float64(atomic.LoadInt64(&bytesDone)) / float64(signature.Container.Size))
	}

	// validate dirs and symlinks first
	for dirIndex, dir := range signature.Container.Dirs {
		path := filepath.Join(target, filepath.FromSlash(dir.Path))
		stats, err := os.Lstat(path)
		if err != nil {
			if os.IsNotExist(err) {
				vctx.Wounds <- &Wound{
					Kind:  WoundKind_DIR,
					Index: int64(dirIndex),
				}
				continue
			} else {
				return err
			}
		}

		if !stats.IsDir() {
			vctx.Wounds <- &Wound{
				Kind:  WoundKind_DIR,
				Index: int64(dirIndex),
			}
			continue
		}
	}

	for symlinkIndex, symlink := range signature.Container.Symlinks {
		path := filepath.Join(target, filepath.FromSlash(symlink.Path))
		dest, err := os.Readlink(path)
		if err != nil {
			if os.IsNotExist(err) {
				vctx.Wounds <- &Wound{
					Kind:  WoundKind_SYMLINK,
					Index: int64(symlinkIndex),
				}
				continue
			} else {
				return err
			}
		}

		if dest != filepath.FromSlash(symlink.Dest) {
			vctx.Wounds <- &Wound{
				Kind:  WoundKind_SYMLINK,
				Index: int64(symlinkIndex),
			}
			continue
		}
	}

	fileIndices := make(chan int64)

	for i := 0; i < numWorkers; i++ {
		go vctx.validate(target, signature, fileIndices, workerErrs, onProgress, cancelled)
	}

	var retErr error
	sending := true

	for fileIndex := range signature.Container.Files {
		if !sending {
			break
		}

		select {
		case workerErr := <-workerErrs:
			workerErrs <- nil
			retErr = workerErr
			close(cancelled)
			sending = false

		case consumerErr := <-consumerErrs:
			consumerErrs <- nil
			retErr = consumerErr
			close(cancelled)
			sending = false

		case fileIndices <- int64(fileIndex):
			// just queued another file
		}
	}

	close(fileIndices)

	// wait for all workers to finish
	for i := 0; i < numWorkers; i++ {
		err := <-workerErrs
		if err != nil {
			close(cancelled)
			if retErr == nil {
				retErr = err
			}
		}
	}

	close(vctx.Wounds)

	// wait for wound consumer to finish
	cErr := <-consumerErrs
	if cErr != nil {
		if retErr == nil {
			retErr = cErr
		}
	}

	return retErr
}

type onProgressFunc func(delta int64)

func (vctx *ValidatorContext) validate(target string, signature *SignatureInfo, fileIndices chan int64,
	errs chan error, onProgress onProgressFunc, cancelled chan struct{}) {

	var retErr error

	targetPool, err := pools.New(signature.Container, target)
	if err != nil {
		errs <- err
		return
	}

	defer func() {
		err := targetPool.Close()
		if err != nil {
			retErr = errors.Wrap(err, 1)
			return
		}

		errs <- retErr
	}()

	aggregateOut := make(chan *Wound)
	relayDone := make(chan bool)
	go func() {
		for w := range aggregateOut {
			vctx.Wounds <- w
		}
		relayDone <- true
	}()

	wounds := AggregateWounds(aggregateOut, MaxWoundSize)
	defer func() {
		// signal no more wounds are going to be sent
		close(wounds)
		// wait for all of them to be relayed
		<-relayDone
	}()

	validatingPool := &ValidatingPool{
		Pool:      nullpool.New(signature.Container),
		Container: signature.Container,
		Signature: signature,

		Wounds: wounds,
	}

	doOne := func(fileIndex int64) error {
		file := signature.Container.Files[fileIndex]

		var reader io.Reader
		reader, err = targetPool.GetReader(fileIndex)
		if err != nil {
			if os.IsNotExist(err) {
				// whole file is missing
				wound := &Wound{
					Kind:  WoundKind_FILE,
					Index: fileIndex,
					Start: 0,
					End:   file.Size,
				}
				onProgress(file.Size)

				select {
				case wounds <- wound:
				case <-cancelled:
				}
				return nil
			}
			return err
		}

		var writer io.WriteCloser
		writer, err = validatingPool.GetWriter(fileIndex)
		if err != nil {
			return err
		}

		defer writer.Close()

		lastCount := int64(0)
		countingWriter := counter.NewWriterCallback(func(count int64) {
			delta := count - lastCount
			onProgress(delta)
			lastCount = count
		}, writer)

		var writtenBytes int64
		writtenBytes, err = io.Copy(countingWriter, reader)
		if err != nil {
			return err
		}

		if writtenBytes != file.Size {
			onProgress(file.Size - writtenBytes)
			wound := &Wound{
				Kind:  WoundKind_FILE,
				Index: fileIndex,
				Start: writtenBytes,
				End:   file.Size,
			}

			select {
			case wounds <- wound:
			case <-cancelled:
			}
		}

		return nil
	}

	for {
		select {
		case fileIndex, ok := <-fileIndices:
			if !ok {
				// no more work
				return
			}

			err := doOne(fileIndex)
			if err != nil {
				if retErr == nil {
					retErr = err
				}
				return
			}
		case <-cancelled:
			// cancelled
			return
		}
	}
}

func AssertValid(target string, signature *SignatureInfo) error {
	vctx := &ValidatorContext{
		FailFast: true,
		Consumer: &state.Consumer{},
	}

	err := vctx.Validate(target, signature)
	if err != nil {
		return err
	}

	return nil
}
