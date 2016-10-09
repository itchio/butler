package retrycontext

import (
	"math/rand"
	"time"

	"github.com/itchio/wharf/state"
)

type Context struct {
	Settings Settings

	Tries       int
	LastMessage string
}

type Settings struct {
	MaxTries int
	Consumer *state.Consumer
	NoSleep  bool
}

func New(settings Settings) *Context {
	return &Context{
		Tries:    1,
		Settings: settings,
	}
}

func NewDefault() *Context {
	return New(Settings{
		MaxTries: 15,
	})
}

func (rc *Context) ShouldTry() bool {
	return rc.Tries < rc.Settings.MaxTries
}

func (rc *Context) Retry(message string) {
	rc.LastMessage = message

	if rc.Settings.Consumer != nil {
		rc.Settings.Consumer.PauseProgress()
		rc.Settings.Consumer.Infof("")
		rc.Settings.Consumer.Infof("%s", message)
	}

	// exponential backoff: 1, 2, 4, 8 seconds...
	delay := rc.Tries * rc.Tries
	// ...plus a random number of milliseconds.
	// see https://cloud.google.com/storage/docs/exponential-backoff
	jitter := rand.Int() % 1000

	if rc.Settings.Consumer != nil {
		rc.Settings.Consumer.Infof("Sleeping %d seconds then retrying", delay)
	}

	if !rc.Settings.NoSleep {
		time.Sleep(time.Second*time.Duration(delay) + time.Millisecond*time.Duration(jitter))
	}

	rc.Tries++

	if rc.Settings.Consumer != nil {
		rc.Settings.Consumer.ResumeProgress()
	}
}
