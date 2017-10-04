package cave

import (
	"fmt"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
)

type CommandError struct {
	Type      string `json:"type"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Operation string `json:"operation"`
}

func (ce *CommandError) Error() string {
	return fmt.Sprintf("command %s error %s: %s", ce.Operation, ce.Code, ce.Message)
}

func (ce *CommandError) Throw() error {
	ce.Type = "command-error"
	comm.Result(ce)

	return errors.Wrap(ce, 1)
}
