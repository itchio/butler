package pwr

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/itchio/wharf/tlc"

	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pools"
	"github.com/itchio/wharf/pools/nullpool"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

// MaxWoundSize is how large AggregateWounds will let an aggregat
// wound get before passing it along to its consumer. The idea is, when
// we're verifying a large file, we can start healing it before it's done
// verifying. In practice, this would require sharing file descriptors,
// and so it isn't done yet.
const MaxWoundSize int64 = 4 * 1024 * 1024 // 4MB

// ValidatorContext holds both input and output parameters to the validation
// process (checking that a container corresponds to its signature: that all
// directories exist, symlinks exist and point to the right destinations, files
// exist and have the right content)
type ValidatorContext struct {
	WoundsPath string
	HealPath   string
	NumWorkers int

	Consumer *state.Consumer

	// FailFast makes Validate return Wounds as errors and stop checking
	FailFast bool

	// Result

	// internal
	Wounds         chan *Wound
	WoundsConsumer WoundsConsumer
}

const floatIntFactor = 10000.0

// Validate checks the directory at target using the container info and hashes
// contained in signature. FailFast mode returns an error on the first corruption
// seen, other modes write wounds to a file or for a wounds consumer, like a healer.
func (vctx *ValidatorContext) Validate(ctx context.Context, target string, signature *SignatureInfo) error {
	if vctx.Consumer == nil {
		vctx.Consumer = &state.Consumer{}
	}

	numWorkers := vctx.NumWorkers
	if numWorkers == 0 {
		numWorkers = runtime.NumCPU() + 1
	}

	vctx.Wounds = make(chan *Wound, 1024)
	workerErrs := make(chan error, numWorkers)
	consumerErrs := make(chan error, 1)
	cancelled := make(chan struct{})

	var woundsStateConsumer *state.Consumer
	var healerProgress float64
	var bytesDone int64
	var progressMutex sync.Mutex

	updateProgress := func() {
		progressMutex.Lock()
		if woundsStateConsumer == nil {
			scanProgress := float64(bytesDone) / float64(signature.Container.Size)
			vctx.Consumer.Progress(scanProgress)
		} else {
			vctx.Consumer.Progress(healerProgress)
		}
		progressMutex.Unlock()
	}

	onProgress := func(delta int64) {
		progressMutex.Lock()
		bytesDone += delta
		progressMutex.Unlock()
		updateProgress()
	}

	if vctx.FailFast {
		if vctx.WoundsPath != "" {
			return fmt.Errorf("ValidatorContext: FailFast is not compatible with WoundsPath")
		}
		if vctx.HealPath != "" {
			return fmt.Errorf("ValidatorContext: FailFast is not compatible with HealPath")
		}

		vctx.WoundsConsumer = &WoundsGuardian{}
	} else if vctx.WoundsPath != "" {
		vctx.WoundsConsumer = &WoundsWriter{
			WoundsPath: vctx.WoundsPath,
		}
	} else if vctx.HealPath != "" {
		// healers can deal with "everything missing"
		err := os.MkdirAll(target, 0755)
		if err != nil {
			return err
		}

		healer, err := NewHealer(vctx.HealPath, target)
		if err != nil {
			return err
		}

		woundsStateConsumer = &state.Consumer{
			OnProgress: func(progress float64) {
				progressMutex.Lock()
				healerProgress = progress
				progressMutex.Unlock()
				updateProgress()
			},
			OnProgressLabel: func(label string) {
				vctx.Consumer.ProgressLabel(label)
			},
		}
		healer.SetConsumer(woundsStateConsumer)

		vctx.WoundsConsumer = healer
	} else {
		vctx.WoundsConsumer = &WoundsPrinter{
			Consumer: vctx.Consumer,
		}
	}

	go func() {
		consumerErrs <- vctx.WoundsConsumer.Do(ctx, signature.Container, vctx.Wounds)

		// throw away wounds until closed
		for range vctx.Wounds {
			// muffin
		}
	}()

	// validate dirs and symlinks first
	for dirIndex, dir := range signature.Container.Dirs {
		path := filepath.Join(target, filepath.FromSlash(dir.Path))
		stats, err := os.Lstat(path)
		if err != nil {
			if IsNotExist(err) {
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
			retErr = errors.WithStack(err)
			return
		}

		errs <- retErr
	}()

	validatingPool := &ValidatingPool{
		Pool:      nullpool.New(signature.Container),
		Container: signature.Container,
		Signature: signature,

		Wounds: vctx.Wounds,
		WoundsFilter: func(wounds chan *Wound) chan *Wound {
			return AggregateWounds(wounds, MaxWoundSize)
		},
	}

	doOne := func(fileIndex int64) error {
		file := signature.Container.Files[fileIndex]

		var reader io.Reader
		reader, err = targetPool.GetReader(fileIndex)
		if err != nil {
			if IsNotExist(err) {
				// whole file is missing
				wound := &Wound{
					Kind:  WoundKind_FILE,
					Index: fileIndex,
					Start: 0,
					End:   file.Size,
				}
				onProgress(file.Size)

				select {
				case vctx.Wounds <- wound:
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
			case vctx.Wounds <- wound:
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

// AssertValid validates target in FailFast mode - it's a shorthand
// so that setting up ValidatorContext isn't needed
func AssertValid(target string, signature *SignatureInfo) error {
	vctx := &ValidatorContext{
		FailFast: true,
		Consumer: &state.Consumer{},
	}

	err := vctx.Validate(context.Background(), target, signature)
	if err != nil {
		return err
	}

	return nil
}

func AssertNoGhosts(target string, signature *SignatureInfo) error {
	refContainer := signature.Container
	gotContainer, err := tlc.WalkAny(target, &tlc.WalkOpts{})
	if err != nil {
		return err
	}

	return refContainer.EnsureEqual(gotContainer)
}

// IsNotExist is a variant of os.IsNotExist that works with nested errors
func IsNotExist(err error) bool {
	return os.IsNotExist(errors.Cause(err))
}
