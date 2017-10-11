package operate

import (
	"fmt"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
)

type OperationError struct {
	Type      string `json:"type"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Operation string `json:"operation"`
}

func (oe *OperationError) Error() string {
	return fmt.Sprintf("command %s error %s: %s", oe.Operation, oe.Code, oe.Message)
}

func (oe *OperationError) Throw() error {
	oe.Type = "command-error"
	comm.Result(oe)

	return errors.Wrap(oe, 1)
}
