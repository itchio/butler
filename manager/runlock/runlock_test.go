package runlock_test

import (
	"context"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/itchio/butler/manager/runlock"
	"github.com/itchio/wharf/wtest"
	"github.com/stretchr/testify/assert"
)

func Test_Runlock(t *testing.T) {
	assert := assert.New(t)

	installFolder, err := ioutil.TempDir("", "runlock-test-installfolder")
	wtest.Must(t, err)

	ctx := context.Background()

	var steps []string
	var mutex sync.Mutex
	done := func(step string) {
		mutex.Lock()
		steps = append(steps, step)
		mutex.Unlock()
	}

	rl1 := runlock.New(nil, installFolder)
	wtest.Must(t, rl1.Lock(ctx, "rl1"))
	done("r1-lock")

	go func() {
		time.Sleep(1000 * time.Millisecond)
		wtest.Must(t, rl1.Unlock())
	}()

	rl2 := runlock.New(nil, installFolder)
	timeoutCtx, cancel := context.WithTimeout(ctx, 600*time.Millisecond)
	defer cancel()
	err = rl2.Lock(timeoutCtx, "rl2")
	if err == nil {
		panic("expected first rl2 lock to time out")
	}
	done("r2-timeout")

	timeoutCtx, cancel = context.WithTimeout(ctx, 600*time.Millisecond)
	defer cancel()
	err = rl2.Lock(timeoutCtx, "rl2")
	done("r2-lock")

	wtest.Must(t, rl2.Unlock())

	assert.EqualValues([]string{
		"r1-lock",
		"r2-timeout",
		"r2-lock",
	}, steps)
}
