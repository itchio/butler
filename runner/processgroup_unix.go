// +build !windows

package runner

import (
	"context"
	"os/exec"
	"syscall"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/state"
)

func SetupProcessGroup(consumer *state.Consumer, cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return nil
}

func WaitProcessGroup(consumer *state.Consumer, cmd *exec.Cmd, ctx context.Context) error {
	waitDone := make(chan error)
	go func() {
		waitDone <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		consumer.Infof("Force closing...")
		pgid, err := syscall.Getpgid(cmd.Process.Pid)
		if err == nil && pgid != 0 {
			consumer.Infof("Killing all processes in group %d", pgid)
			err = syscall.Kill(-pgid, syscall.SIGTERM)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			return errors.Wrap(err, 0)
		} else {
			if err != nil {
				consumer.Infof("Could not get group of process %d: %s", err.Error())
			} else {
				consumer.Infof("Process %d had no group", cmd.Process.Pid)
			}
			consumer.Infof("Killing single process %d", cmd.Process.Pid)
			err = syscall.Kill(cmd.Process.Pid, syscall.SIGTERM)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
	case err := <-waitDone:
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return nil
}
