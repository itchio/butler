package retrycontext

import (
	"math/rand"
	"time"

	"github.com/itchio/httpkit/neterr"

	"github.com/itchio/wharf/state"
)

type Context struct {
	Settings Settings

	Tries     int
	LastError error
}

type Settings struct {
	MaxTries int
	Consumer *state.Consumer
	NoSleep  bool
}

func New(settings Settings) *Context {
	return &Context{
		Tries:    0,
		Settings: settings,
	}
}

func NewDefault() *Context {
	return New(Settings{
		MaxTries: 10,
	})
}

/**
 ShouldTry must be used in a loop, like so:

 ----------------------------------------
 for rc.ShouldRetry() {
	 err := someOperation()
	 if err != nil {
		 if isRetriable(err) {
			 rc.Retry(err.Error())
			 continue
		 }
	 }

	 // succeeded!
	 return nil // escape from loop
 }

 // THIS IS IMPORTANT
 return errors.New("task: too many failures, giving up")
 ----------------------------------------

 If you forget to return an error after the loop,
 if there are too many errors you'll just keep running
*/
func (rc *Context) ShouldTry() bool {
	return rc.Tries < rc.Settings.MaxTries
}

func (rc *Context) Retry(err error) {
	rc.LastError = err

	if rc.Settings.Consumer != nil {
		rc.Settings.Consumer.PauseProgress()
		if neterr.IsNetworkError(err) {
			rc.Settings.Consumer.Infof("having network troubles...")
		} else {
			rc.Settings.Consumer.Infof("%v", err)
		}
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
