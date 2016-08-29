package uploader

import (
	"math/rand"
	"time"

	"github.com/itchio/wharf/pwr"
)

type RetryContext struct {
	tries    int
	maxTries int
	consumer *pwr.StateConsumer
}

func NewRetryContext(consumer *pwr.StateConsumer) *RetryContext {
	return &RetryContext{
		tries:    1,
		maxTries: 10,
		consumer: consumer,
	}
}

func (rc *RetryContext) ShouldTry() bool {
	return rc.tries < rc.maxTries
}

func (rc *RetryContext) Retry(message string) {
	rc.consumer.PauseProgress()
	rc.consumer.Infof("")
	rc.consumer.Infof("%s", message)

	// exponential backoff: 1, 2, 4, 8 seconds...
	delay := rc.tries * rc.tries
	// ...plus a random number of milliseconds. see https://cloud.google.com/storage/docs/exponential-backoff
	jitter := rand.Int() % 1000

	rc.consumer.Infof("Sleeping %d seconds then retrying", delay)
	time.Sleep(time.Second*time.Duration(delay) + time.Millisecond*time.Duration(jitter))
	rc.tries++

	rc.consumer.ResumeProgress()
}
