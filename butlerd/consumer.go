package butlerd

import (
	"context"
	"os"

	"github.com/itchio/butler/comm"
	"github.com/itchio/headway/state"
	"github.com/pkg/errors"
)

type NewStateConsumerParams struct {
	// Mandatory
	Conn Conn
	Ctx  context.Context

	// Optional
	LogFile *os.File
}

func NewStateConsumer(params *NewStateConsumerParams) (*state.Consumer, error) {
	if params.Conn == nil {
		return nil, errors.New("NewConsumer: missing Conn")
	}

	if params.Ctx == nil {
		return nil, errors.New("NewConsumer: missing Ctx")
	}

	c := &state.Consumer{
		OnMessage: func(level, msg string) {
			err := params.Conn.Notify(params.Ctx, "Log", LogNotification{
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
