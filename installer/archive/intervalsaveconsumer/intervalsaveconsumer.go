package intervalsaveconsumer

import (
	"context"
	"encoding/gob"
	"os"
	"time"

	"github.com/dchest/safefile"
	"github.com/go-errors/errors"
	"github.com/itchio/savior"
	"github.com/itchio/wharf/state"
)

type saveConsumer struct {
	statePath string
	interval  time.Duration
	consumer  *state.Consumer
	ctx       context.Context

	lastSave time.Time
}

var _ savior.SaveConsumer = (*saveConsumer)(nil)

var DefaultInterval = 1 * time.Second

func New(statePath string, interval time.Duration, consumer *state.Consumer, ctx context.Context) *saveConsumer {
	return &saveConsumer{
		statePath: statePath,
		interval:  interval,
		consumer:  consumer,

		lastSave: time.Now(),
		ctx:      ctx,
	}
}

func (sc *saveConsumer) Load() (*savior.ExtractorCheckpoint, error) {
	state := &savior.ExtractorCheckpoint{}

	stateFile, err := os.Open(sc.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, 0)
	}
	defer stateFile.Close()

	dec := gob.NewDecoder(stateFile)
	err = dec.Decode(state)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return state, nil
}

func (sc *saveConsumer) ShouldSave(n int64) bool {
	return time.Since(sc.lastSave) >= sc.interval
}

func (sc *saveConsumer) Save(state *savior.ExtractorCheckpoint) (savior.AfterSaveAction, error) {
	sc.lastSave = time.Now()

	err := func() error {
		stateFile, err := safefile.Create(sc.statePath, 0644)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		defer stateFile.Close()

		enc := gob.NewEncoder(stateFile)
		err = enc.Encode(state)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		err = stateFile.Commit()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}()
	if err != nil {
		sc.consumer.Warnf("saveconsumer: Could not persist extractor state: %s", err.Error())
	}

	var action savior.AfterSaveAction
	select {
	case <-sc.ctx.Done():
		sc.consumer.Warnf("saveconsumer: Stopping extractor")
		action = savior.AfterSaveStop
	default:
		action = savior.AfterSaveContinue
	}

	return action, nil
}
