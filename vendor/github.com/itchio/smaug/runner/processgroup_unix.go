// +build !windows

package runner

import (
	"context"
	"os/exec"
	"syscall"

	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type processGroup struct {
	consumer *state.Consumer
	cmd      *exec.Cmd
	ctx      context.Context
}

func NewProcessGroup(consumer *state.Consumer, cmd *exec.Cmd, ctx context.Context) (*processGroup, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	pg := &processGroup{
		consumer: consumer,
		cmd:      cmd,
		ctx:      ctx,
	}
	return pg, nil
}

func (pg *processGroup) AfterStart() error {
	return nil
}

func (pg *processGroup) Wait() error {
	waitDone := make(chan error)
	go func() {
		waitDone <- pg.cmd.Wait()
	}()

	pid := pg.cmd.Process.Pid

	select {
	case <-pg.ctx.Done():
		pg.consumer.Infof("Force closing...")
		pgid, err := syscall.Getpgid(pid)
		if err == nil && pgid != 0 {
			pg.consumer.Infof("Killing all processes in group %d", pgid)
			err = syscall.Kill(-pgid, syscall.SIGTERM)
			if err != nil {
				return errors.WithStack(err)
			}

			return errors.WithStack(err)
		} else {
			if err != nil {
				pg.consumer.Infof("Could not get group of process %d: %s", err.Error())
			} else {
				pg.consumer.Infof("Process %d had no group", pid)
			}
			pg.consumer.Infof("Killing single process %d", pid)
			err = syscall.Kill(pid, syscall.SIGTERM)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	case err := <-waitDone:
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
