package butlerd

import (
	"fmt"

	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/itchio/savior"
)

type Error interface {
	error
	RpcErrorCode() int64
	RpcErrorMessage() string
	RpcErrorData() map[string]interface{}
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

func (re *RpcError) RpcErrorCode() int64 {
	return re.Code
}

func (re *RpcError) RpcErrorMessage() string {
	return re.Message
}

func (re *RpcError) RpcErrorData() map[string]interface{} {
	return nil
}

func (re *RpcError) Error() string {
	return fmt.Sprintf("RPC error %d: %s", re.Code, re.Message)
}

//

type causer interface {
	Cause() error
}

func AsButlerdError(err error) (Error, bool) {
	if err == nil {
		return nil, false
	}

	if err == savior.ErrStop {
		return CodeOperationCancelled, true
	}

	if se, ok := err.(causer); ok {
		return AsButlerdError(se.Cause())
	}

	if ee, ok := err.(Error); ok {
		return ee, true
	}

	return nil, false
}
