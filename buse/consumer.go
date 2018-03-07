package buse

import (
	"context"
	"os"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/state"
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
			params.Conn.Notify(params.Ctx, "Log", &LogNotification{
				Level:   LogLevel(level),
				Message: msg,
			})
		},
	}

	return c, nil
}
