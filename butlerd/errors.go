package butlerd

import (
	"fmt"

	"github.com/go-errors/errors"
	"github.com/sourcegraph/jsonrpc2"
)

type Error interface {
	error
	AsJsonRpc2() *jsonrpc2.Error
}

type RpcError struct {
	Code    int64
	Message string
}

var _ Error = (*RpcError)(nil)

func StandardRpcError(Code int64) Error {
	var message string = "Unknown error"
	switch Code {
	case jsonrpc2.CodeParseError:
		message = "Parse error"
	case jsonrpc2.CodeInvalidRequest:
		message = "Invalid request"
	case jsonrpc2.CodeMethodNotFound:
		message = "Method not found"
	case jsonrpc2.CodeInvalidParams:
		message = "Invalid params"
	case jsonrpc2.CodeInternalError:
		message = "Internal error"
	}
	return &RpcError{Code: Code, Message: message}
}

func (re *RpcError) AsJsonRpc2() *jsonrpc2.Error {
	return &jsonrpc2.Error{
		Code:    re.Code,
		Message: re.Message,
	}
}

func (re *RpcError) Error() string {
	return fmt.Sprintf("RPC error %d: %s", re.Code, re.Message)
}

//

type ErrAborted struct{}

var _ Error = (*ErrAborted)(nil)

func (e *ErrAborted) AsJsonRpc2() *jsonrpc2.Error {
	return &jsonrpc2.Error{
		Code:    int64(CodeOperationAborted),
		Message: "operation aborted",
	}
}

func (e *ErrAborted) Error() string {
	return "operation aborted"
}

//

type ErrCancelled struct{}

var _ Error = (*ErrCancelled)(nil)

func (e *ErrCancelled) AsJsonRpc2() *jsonrpc2.Error {
	return &jsonrpc2.Error{
		Code:    int64(CodeOperationCancelled),
		Message: "operation cancelled",
	}
}

func (e *ErrCancelled) Error() string {
	return "operation cancelled"
}

//

func AsButlerdError(err error) (Error, bool) {
	if err == nil {
		return nil, false
	}

	if se, ok := err.(*errors.Error); ok {
		return AsButlerdError(se.Err)
	}

	if ee, ok := err.(Error); ok {
		return ee, true
	}

	return nil, false
}
