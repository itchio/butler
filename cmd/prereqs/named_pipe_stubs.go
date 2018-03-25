// +build !windows

package prereqs

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/state"
)

type NamedPipe struct {
}

func NewNamedPipe(pipePath string) (*NamedPipe, error) {
	np := &NamedPipe{}

	return np, nil
}

func (np NamedPipe) Consumer() *state.Consumer {
	return comm.NewStateConsumer()
}

func (np NamedPipe) WriteState(taskName string, status butlerd.PrereqStatus) error {
	msg := PrereqState{
		Type:   "state",
		Name:   taskName,
		Status: status,
	}
	comm.Result(&msg)

	return nil
}
