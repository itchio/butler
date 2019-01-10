package runlock

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/werrors"
)

type Lock interface {
	Lock(ctx context.Context, task string) error
	Unlock() error
}

type lock struct {
	consumer      *state.Consumer
	installFolder string
}

type runlockPayload struct {
	Task      string `json:"task"`
	LockedAt  string `json:"lockedAt"`
	ButlerPID int64  `json:"butlerPID"`
}

func New(consumer *state.Consumer, installFolder string) Lock {
	rl := &lock{
		consumer:      consumer,
		installFolder: installFolder,
	}
	return rl
}

func (rl *lock) Lock(ctx context.Context, task string) error {
	printed := false

	isLocked := func() bool {
		rp, _ := rl.read()
		if rp == nil {
			return false
		}
		proc, _ := os.FindProcess(int(rp.ButlerPID))
		if proc != nil {
			// still running
			proc.Release()
		} else {
			// not running anymore
			rl.Unlock()
			return false
		}

		if !printed {
			printed = true
			rl.consumer.Debugf("Waiting (%s) for %s", rl.file(), task)
		}
		return true
	}

	if isLocked() {
	waitLoop:
		for {
			select {
			case <-time.After(1 * time.Second):
				if !isLocked() {
					break waitLoop
				}
			case <-ctx.Done():
				return werrors.ErrCancelled
			}
		}
	}

	rl.consumer.Debugf("Locking (%s) for %s", rl.file(), task)
	return rl.write(&runlockPayload{
		Task:      task,
		LockedAt:  time.Now().Format(time.RFC3339Nano),
		ButlerPID: int64(os.Getpid()),
	})
}

func (rl *lock) Unlock() error {
	return os.RemoveAll(rl.file())
}

func (rl *lock) file() string {
	return filepath.Join(rl.installFolder, ".itch", "runlock.json")
}

func (rl *lock) write(rp *runlockPayload) error {
	contents, err := json.Marshal(rp)
	if err != nil {
		return err
	}

	file := rl.file()
	err = os.MkdirAll(filepath.Dir(file), 0755)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, contents, 0644)
}

func (rl *lock) read() (*runlockPayload, error) {
	var rp runlockPayload
	contents, err := ioutil.ReadFile(rl.file())
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(contents, &rp)
	if err != nil {
		return nil, err
	}
	return &rp, nil
}
