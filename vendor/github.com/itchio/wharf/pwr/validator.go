package pwr

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pools"
	"github.com/itchio/wharf/pools/nullpool"
	"github.com/itchio/wharf/wsync"
)

const MaxWoundSize int64 = 4 * 1024 * 1024 // 4MB

type ValidatorContext struct {
	WoundsPath string
	NumWorkers int

	Consumer *StateConsumer

	// FailFast makes Validate return Wounds as errors and stop checking
	FailFast bool

	// Result
	TotalCorrupted int64

	// internal
	TargetPool wsync.Pool
	Wounds     chan *Wound
}

func (vctx *ValidatorContext) Validate(target string, signature *SignatureInfo) error {
	var woundsConsumer WoundsConsumer

	vctx.Wounds = make(chan *Wound)
	errs := make(chan error)
	done := make(chan bool)

	if vctx.FailFast {
		if vctx.WoundsPath != "" {
			return fmt.Errorf("Validate: FailFast is not compatible with WoundsPath")
		}

		woundsConsumer = &WoundsGuardian{}
	} else if vctx.WoundsPath == "" {
		woundsConsumer = &WoundsPrinter{
			Consumer: vctx.Consumer,
		}
	} else {
		woundsConsumer = &WoundsWriter{
			WoundsPath: vctx.WoundsPath,
		}
	}

	go func() {
		err := woundsConsumer.Do(signature.Container, vctx.Wounds)
		if err != nil {
			errs <- err
			return
		}
		done <- true
	}()

	doneBytes := make(chan int64)

	go func() {
		done := int64(0)

		for chunkSize := range doneBytes {
			done += chunkSize
			if vctx.Consumer != nil {
				vctx.Consumer.Progress(float64(done) / float64(signature.Container.Size))
			}
		}
	}()

	numWorkers := vctx.NumWorkers
	if numWorkers == 0 {
		numWorkers = runtime.NumCPU() + 1
	}

	fileIndices := make(chan int64)

	for i := 0; i < numWorkers; i++ {
		go vctx.validate(target, signature, fileIndices, done, errs, doneBytes)
	}

	for fileIndex := range signature.Container.Files {
		fileIndices <- int64(fileIndex)
	}

	close(fileIndices)

	// wait for all workers to finish
	for i := 0; i < numWorkers; i++ {
		select {
		case err := <-errs:
			return err
		case <-done:
			// good!
		}
	}

	close(doneBytes)
	close(vctx.Wounds)

	// wait for wounds writer to finish
	select {
	case err := <-errs:
		return err
	case <-done:
		// good!
	}

	return nil
}

func (vctx *ValidatorContext) validate(target string, signature *SignatureInfo, fileIndices chan int64, done chan bool, errs chan error, doneBytes chan int64) {
	targetPool, err := pools.New(signature.Container, target)
	if err != nil {
		errs <- err
		return
	}

	aggregateOut := make(chan *Wound)
	relayDone := make(chan bool)
	go func() {
		for w := range aggregateOut {
			vctx.Wounds <- w
		}
		relayDone <- true
	}()

	wounds := AggregateWounds(aggregateOut, MaxWoundSize)

	validatingPool := &ValidatingPool{
		Pool:      nullpool.New(signature.Container),
		Container: signature.Container,
		Signature: signature,

		Wounds: wounds,
	}

	for fileIndex := range fileIndices {
		file := signature.Container.Files[fileIndex]

		var reader io.Reader
		reader, err = targetPool.GetReader(fileIndex)
		if err != nil {
			if os.IsNotExist(err) {
				doneBytes <- file.Size

				// that's one big wound
				wounds <- &Wound{
					FileIndex: fileIndex,
					Start:     0,
					End:       file.Size,
				}
				continue
			} else {
				errs <- err
				return
			}
		}

		var writer io.WriteCloser
		writer, err = validatingPool.GetWriter(fileIndex)
		if err != nil {
			errs <- errors.Wrap(err, 1)
			return
		}

		lastCount := int64(0)
		countingWriter := counter.NewWriterCallback(func(count int64) {
			diff := count - lastCount
			doneBytes <- diff
			lastCount = count
		}, writer)

		var writtenBytes int64
		writtenBytes, err = io.Copy(countingWriter, reader)
		if err != nil {
			errs <- errors.Wrap(err, 1)
			return
		}

		err = writer.Close()
		if err != nil {
			errs <- errors.Wrap(err, 1)
			return
		}

		if writtenBytes != file.Size {
			doneBytes <- (file.Size - writtenBytes)
			wounds <- &Wound{
				FileIndex: fileIndex,
				Start:     writtenBytes,
				End:       file.Size,
			}
		}
	}

	err = targetPool.Close()
	if err != nil {
		errs <- errors.Wrap(err, 1)
		return
	}

	close(wounds)
	<-relayDone

	done <- true
}
