package pwr

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/go-errors/errors"
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
	var woundsWriter *WoundsWriter
	vctx.Wounds = make(chan *Wound)
	errs := make(chan error)
	done := make(chan bool)

	countedWounds := vctx.countWounds(vctx.Wounds)

	if vctx.FailFast {
		if vctx.WoundsPath != "" {
			return fmt.Errorf("Validate: FailFast is not compatibel with WoundsPath")
		}

		go func() {
			for w := range countedWounds {
				errs <- fmt.Errorf(w.PrettyString(signature.Container))
			}
			done <- true
		}()
	} else if vctx.WoundsPath == "" {
		woundsPrinter := &WoundsPrinter{
			Wounds: countedWounds,
		}

		go func() {
			err := woundsPrinter.Do(signature, vctx.Consumer)
			if err != nil {
				errs <- err
				return
			}
			done <- true
		}()
	} else {
		woundsWriter = &WoundsWriter{
			Wounds: countedWounds,
		}

		go func() {
			err := woundsWriter.Do(signature, vctx.WoundsPath)
			if err != nil {
				errs <- err
				return
			}
			done <- true
		}()
	}

	numWorkers := vctx.NumWorkers
	if numWorkers == 0 {
		numWorkers = runtime.NumCPU() + 1
	}

	fileIndices := make(chan int64)

	for i := 0; i < numWorkers; i++ {
		go vctx.validate(target, signature, fileIndices, done, errs)
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

func (vctx *ValidatorContext) countWounds(inWounds chan *Wound) chan *Wound {
	outWounds := make(chan *Wound)

	go func() {
		for wound := range inWounds {
			vctx.TotalCorrupted += (wound.End - wound.Start)
			outWounds <- wound
		}

		close(outWounds)
	}()

	return outWounds
}

func (vctx *ValidatorContext) validate(target string, signature *SignatureInfo, fileIndices chan int64, done chan bool, errs chan error) {
	targetPool, err := pools.New(signature.Container, target)
	if err != nil {
		errs <- err
		return
	}

	wounds := AggregateWounds(vctx.Wounds, MaxWoundSize)

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

		var writtenBytes int64
		writtenBytes, err = io.Copy(writer, reader)
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

	done <- true
}
