package taskgroup

import (
	"context"
	"fmt"

	"github.com/itchio/wharf/werrors"
	"github.com/pkg/errors"
)

// Task describes some work to be done
type Task func() error

// Do runs all tasks, and returns when 1) all tasks have
// completed without errors 2) the context has been canceled
// (returns werrors.ErrCancelled), 3) as soon as one of the
// tasks has returned a non-nil error
func Do(ctx context.Context, tasks ...Task) error {
	n := len(tasks)

	done := make(chan error, n)
	for i, task := range tasks {
		go func(task Task) {
			err := task()
			if err != nil {
				done <- errors.WithMessage(err, fmt.Sprintf("task %d", i+1))
				return
			}
			done <- nil
		}(task)
	}

	for i := 0; i < n; i++ {
		select {
		case err := <-done:
			if err != nil {
				return errors.WithStack(err)
			}
		case <-ctx.Done():
			return werrors.ErrCancelled
		}
	}
	return nil
}
