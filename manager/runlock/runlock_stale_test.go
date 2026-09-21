package runlock

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/itchio/headway/state"
	"github.com/itchio/wharf/wtest"
	"github.com/stretchr/testify/assert"
)

// A lock left by a process whose PID now belongs to something else is
// stale and gets replaced.
func Test_RunlockStalePID(t *testing.T) {
	installFolder, err := ioutil.TempDir("", "runlock-test-stale")
	wtest.Must(t, err)
	defer os.RemoveAll(installFolder)

	consumer := &state.Consumer{
		OnMessage: func(lvl string, msg string) { t.Logf("[%s] %s", lvl, msg) },
	}
	rl := New(consumer, installFolder).(*lock)

	// Our own PID is certainly alive, but the recorded process is not us.
	wtest.Must(t, rl.write(&runlockPayload{
		Task:      "stale",
		LockedAt:  time.Now().Format(time.RFC3339Nano),
		ButlerPID: int64(os.Getpid()),
		Identity:  "some-earlier-process",
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	wtest.Must(t, rl.Lock(ctx, "fresh"))

	rp, err := rl.read()
	wtest.Must(t, err)
	assert.EqualValues(t, "fresh", rp.Task)
	assert.EqualValues(t, os.Getpid(), rp.ButlerPID)
	assert.NotEmpty(t, rp.Identity)
	wtest.Must(t, rl.Unlock())
}

// A lock held by a live process is honored.
func Test_RunlockLivePID(t *testing.T) {
	installFolder, err := ioutil.TempDir("", "runlock-test-live")
	wtest.Must(t, err)
	defer os.RemoveAll(installFolder)

	consumer := &state.Consumer{
		OnMessage: func(lvl string, msg string) { t.Logf("[%s] %s", lvl, msg) },
	}
	rl := New(consumer, installFolder).(*lock)
	wtest.Must(t, rl.Lock(context.Background(), "held"))

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	err = New(consumer, installFolder).Lock(ctx, "waiting")
	assert.Error(t, err)
	wtest.Must(t, rl.Unlock())
}
