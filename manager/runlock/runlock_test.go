package runlock_test

import (
	"context"
	"io/ioutil"
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
	done := func(step string) {
		steps = append(steps, step)
	}

	rl1 := runlock.New(nil, installFolder)
	wtest.Must(t, rl1.Lock(ctx, "rl1"))
	done("r1-lock")

	go func() {
		time.Sleep(1000 * time.Millisecond)
		wtest.Must(t, rl1.Unlock())
		done("r1-unlock")
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
	done("r2-unlock")

	assert.EqualValues([]string{
		"r1-lock",
		"r2-timeout",
		"r1-unlock",
		"r2-lock",
		"r2-unlock",
	}, steps)
}
