
package checker

import (
	"github.com/itchio/savior"
)

type OnSaveFunc func(checkpoint *savior.ExtractorCheckpoint) (savior.AfterSaveAction, error)

type counterSaveConsumer struct {
	c int64

	threshold int64
	onSave    OnSaveFunc
}

var _ savior.SaveConsumer = (*counterSaveConsumer)(nil)

func NewTestSaveConsumer(threshold int64, onSave OnSaveFunc) savior.SaveConsumer {
	return &counterSaveConsumer{
		threshold: threshold,
		onSave:    onSave,
	}
}

func (csc *counterSaveConsumer) ShouldSave(n int64) bool {
	csc.c += n
	return csc.c > csc.threshold
}

func (csc *counterSaveConsumer) Save(checkpoint *savior.ExtractorCheckpoint) (savior.AfterSaveAction, error) {
	csc.c = 0
	if checkpoint != nil {
		return csc.onSave(checkpoint)
	}

	return savior.AfterSaveContinue, nil
}
