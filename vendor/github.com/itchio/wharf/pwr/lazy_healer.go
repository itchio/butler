package pwr

import (
	"github.com/go-errors/errors"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
)

type LazyHealer struct {
	numWorkers  int
	consumer    *state.Consumer
	lockmap     LockMap
	totalHealed int64

	inner      Healer
	makeHealer MakeHealer
}

var _ Healer = (*LazyHealer)(nil)

type MakeHealer func() (Healer, error)

func (lh *LazyHealer) Do(container *tlc.Container, wounds chan *Wound) error {
	var innerWounds chan *Wound
	var innerResult chan error
	var skippedWounds []*Wound

	var initialHealthy int64

	for wound := range wounds {
		if lh.inner == nil {
			if wound.Kind == WoundKind_CLOSED_FILE {
				// don't summon the inner healer if we're just
				// keeping track of which file we're at. it's just
				// a progress thing, no action is required.
				// it would be important if we already had an inner
				// healer, in which case we'd still want to relay it
				skippedWounds = append(skippedWounds, wound)
				initialHealthy += wound.End - wound.Start
				if lh.consumer != nil {
					lh.consumer.Progress(float64(initialHealthy) / float64(container.Size))
				}
				continue
			}

			var err error
			lh.inner, err = lh.makeHealer()
			if err != nil {
				return errors.Wrap(err, 0)
			}

			lh.inner.SetNumWorkers(lh.numWorkers)
			lh.inner.SetLockMap(lh.lockmap)

			innerWounds = make(chan *Wound)
			innerResult = make(chan error)

			go func() {
				innerResult <- lh.inner.Do(container, innerWounds)
			}()

			for _, skippedWound := range skippedWounds {
				innerWounds <- skippedWound
			}
			skippedWounds = nil

			// set consumer late so we don't have progress bar jumpbacks
			lh.inner.SetConsumer(lh.consumer)
		}

		innerWounds <- wound
	}

	if lh.inner != nil {
		close(innerWounds)
		innerErr := <-innerResult
		if innerErr != nil {
			return errors.Wrap(innerErr, 0)
		}
	}

	return nil
}

func (lh *LazyHealer) SetNumWorkers(numWorkers int) {
	lh.numWorkers = numWorkers
}

func (lh *LazyHealer) SetConsumer(consumer *state.Consumer) {
	lh.consumer = consumer
}

func (lh *LazyHealer) SetLockMap(lockmap LockMap) {
	lh.lockmap = lockmap
}

func (lh *LazyHealer) TotalHealed() int64 {
	if lh.inner != nil {
		return lh.inner.TotalHealed()
	}

	return 0
}

func (lh *LazyHealer) TotalCorrupted() int64 {
	if lh.inner != nil {
		return lh.inner.TotalCorrupted()
	}

	return 0
}

func (lh *LazyHealer) HasWounds() bool {
	if lh.inner != nil {
		return lh.inner.HasWounds()
	}

	return false
}
