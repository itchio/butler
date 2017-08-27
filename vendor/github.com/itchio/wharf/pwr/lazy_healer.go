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

	for wound := range wounds {
		if lh.inner == nil {
			var err error
			lh.inner, err = lh.makeHealer()
			if err != nil {
				return errors.Wrap(err, 0)
			}

			lh.inner.SetNumWorkers(lh.numWorkers)
			lh.inner.SetConsumer(lh.consumer)
			lh.inner.SetLockMap(lh.lockmap)

			innerWounds = make(chan *Wound)
			innerResult = make(chan error)

			go func() {
				innerResult <- lh.inner.Do(container, innerWounds)
			}()
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
