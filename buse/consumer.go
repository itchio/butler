package buse

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

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
			if params.LogFile != nil {
				payload, err := json.Marshal(map[string]interface{}{
					"time":  currentTimeMillis(),
					"name":  "butler",
					"level": butlerLevelToItchLevel(level),
					"msg":   msg,
				})
				if err == nil {
					fmt.Fprintf(params.LogFile, "%s\n", string(payload))
				} else {
					fmt.Fprintf(params.LogFile, "could not marshal json log entry: %s\n", err.Error())
				}
			}
			params.Conn.Notify(params.Ctx, "Log", &LogNotification{
				Level:   LogLevel(level),
				Message: msg,
			})
		},
	}

	return c, nil
}

func butlerLevelToItchLevel(level string) int {
	switch level {
	case "fatal":
		return 60
	case "error":
		return 50
	case "warning":
		return 40
	case "info":
		return 30
	case "debug":
		return 20
	case "trace":
		return 10
	default:
		return 30 // default
	}
}

func currentTimeMillis() int64 {
	timeUtc := time.Now().UTC()
	nanos := timeUtc.Nanosecond()
	millis := timeUtc.Unix() * 1000
	millis += int64(nanos) / 1000000
	return millis
}
