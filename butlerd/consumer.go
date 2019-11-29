package butlerd

import (
	"os"

	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/itchio/butler/comm"
	"github.com/itchio/headway/state"
	"github.com/pkg/errors"
)

type NewStateConsumerParams struct {
	// Mandatory
	Conn jsonrpc2.Conn

	// Optional
	LogFile *os.File
}

func NewStateConsumer(params *NewStateConsumerParams) (*state.Consumer, error) {
	if params.Conn == nil {
		return nil, errors.New("NewConsumer: missing Conn")
	}

	c := &state.Consumer{
		OnMessage: func(level, msg string) {
			err := params.Conn.Notify("Log", LogNotification{
				Level:   LogLevel(level),
				Message: msg,
			})
			if err != nil {
				comm.Warnf("Failed to notify: %#v", err)
			}
		},
	}

	return c, nil
}
