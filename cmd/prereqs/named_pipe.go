// +build windows

package prereqs

import (
	"encoding/json"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/state"
	"github.com/natefinch/npipe"
)

type NamedPipe struct {
	conn *npipe.PipeConn
}

func NewNamedPipe(pipePath string) (*NamedPipe, error) {
	np := &NamedPipe{}

	if pipePath != "" {
		conn, err := npipe.Dial(pipePath)
		if err != nil {
			comm.Warnf("Could not dial pipe %s: %s", pipePath, err.Error())
		} else {
			np.conn = conn
		}
	}

	return np, nil
}

func (np NamedPipe) Consumer() *state.Consumer {
	return &state.Consumer{
		OnMessage: func(level string, message string) {
			comm.Logl(level, message)

			contents, err := json.Marshal(&PrereqLogEntry{
				Type:    "log",
				Message: message,
			})
			if err != nil {
				comm.Warnf("could not marshal log message: %s", err.Error())
				return
			}

			err = np.writeLine([]byte(contents))
			if err != nil {
				comm.Warnf("could not send log message: %s", err.Error())
				return
			}
		},
	}
}

func (np NamedPipe) WriteState(taskName string, status string) error {
	msg := PrereqState{
		Type:   "state",
		Name:   taskName,
		Status: status,
	}
	comm.Result(&msg)

	contents, err := json.Marshal(&msg)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return np.writeLine(contents)
}

func (np NamedPipe) writeLine(contents []byte) error {
	if np.conn == nil {
		return nil
	}

	contents = append(contents, '\n')

	_, err := np.conn.Write(contents)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
