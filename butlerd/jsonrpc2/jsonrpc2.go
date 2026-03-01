package jsonrpc2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/homelight/json"
)

var debugEnabled = os.Getenv("BUTLER_JSON_RPC_DEBUG") == "1"

func debug(format string, args ...interface{}) {
	if debugEnabled {
		log.Printf("[jsonrpc debug] %s", fmt.Sprintf(format, args...))
	}
}

// A JSON-RPC 2.0 error, see https://www.jsonrpc.org/specification#error_object
type Error struct {
	Code    ErrorCode        `json:"code"`
	Message string           `json:"message"`
	Data    *json.RawMessage `json:"data"`
}

// implement Go standard Error interface
func (e *Error) Error() string {
	return fmt.Sprintf("json-rpc2: error %d: %s", e.Code, e.Message)
}

// Sets the free-form "data" field
func (e *Error) SetData(v interface{}) error {
	raw, err := EncodeJSON(v)
	if err != nil {
		return err
	}
	e.Data = &raw
	return nil
}

// Gets the free-form "data" field.
// Note: this will panic if Data is nil, check that first.
func (e *Error) GetData(v interface{}) error {
	return DecodeJSON(*e.Data, v)
}

type ID = int64
type ErrorCode = int64

// Standard JSON-RPC2 error codes, see https://www.jsonrpc.org/specification#error_object
const (
	CodeParseError     ErrorCode = -32700
	CodeInvalidRequest ErrorCode = -32600
	CodeMethodNotFound ErrorCode = -32601
	CodeInvalidParams  ErrorCode = -32602
	CodeInternalError  ErrorCode = -32603
)

type OutgoingCall func(msg Message)

type Conn interface {
	Call(method string, params interface{}, result interface{}) error
	Notify(method string, params interface{}) error
	Context() context.Context
	Close()
}

var _ Conn = (*connImpl)(nil)

type Transport interface {
	Read() ([]byte, error)
	Write(msg []byte) error
	Close() error
}

type connImpl struct {
	transport Transport
	ctx       context.Context
	cancel    context.CancelFunc

	handler Handler

	idSeed  ID
	idMutex sync.Mutex

	outgoingCalls      map[ID]OutgoingCall
	outgoingCallsMutex sync.Mutex

	closed           bool
	disconnectNotify chan struct{}
	closeMutex       sync.Mutex

	writeMutex sync.Mutex
}

func NewConn(parentCtx context.Context, transport Transport, handler Handler) *connImpl {
	ctx, cancel := context.WithCancel(parentCtx)

	conn := &connImpl{
		transport: transport,
		ctx:       ctx,
		cancel:    cancel,

		handler: handler,

		outgoingCalls:    make(map[ID]OutgoingCall),
		disconnectNotify: make(chan struct{}),

		idSeed: 0,

		closed: false,
	}

	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	go conn.receiveLoop()

	return conn
}

func (c *connImpl) generateID() ID {
	c.idMutex.Lock()
	defer c.idMutex.Unlock()

	result := c.idSeed
	c.idSeed += 1
	return result
}

func (c *connImpl) Context() context.Context {
	return c.ctx
}

func (c *connImpl) reply(id ID, res json.RawMessage) error {
	resText, err := EncodeJSON(res)
	if err != nil {
		return err
	}

	return c.send(Message{
		ID:     &id,
		Result: &resText,
	})
}

func (c *connImpl) replyWithError(id ID, rpcErr Error) error {
	return c.send(Message{
		ID:    &id,
		Error: &rpcErr,
	})
}

func (c *connImpl) warn(f string, args ...interface{}) {
	// TODO: allow subscribing to warnings
	log.Printf("json-rpc2: %s", fmt.Sprintf(f, args...))
}

func (c *connImpl) send(msg Message) error {
	msg.JsonRPC = "2.0"

	msgText, err := json.MarshalSafeCollections(msg)
	if err != nil {
		return err
	}

	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	err = c.transport.Write(msgText)
	if err != nil {
		return err
	}

	return nil
}

func (c *connImpl) receiveLoop() {
	defer c.Close()

	for {
		msgText, err := c.transport.Read()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				c.warn("%+v", err)
			}
			// we're done here
			return
		}

		var msg Message
		err = DecodeJSON(msgText, &msg)
		if err != nil {
			c.warn("%+v, for input %q", err, string(msgText))
			continue
		}
		c.handleIncomingMessage(msg)
	}
}

func (c *connImpl) handleIncomingMessage(msg Message) {
	if msg.JsonRPC != "2.0" {
		c.warn("received message lacking 'jsonrpc: \"2.0\"', ignoring")
		return
	}

	if msg.Method == nil {

		// no method = result or error
		if msg.ID != nil {
			id := *msg.ID

			c.outgoingCallsMutex.Lock()
			defer c.outgoingCallsMutex.Unlock()
			if oc, ok := c.outgoingCalls[id]; ok {
				delete(c.outgoingCalls, id)
				oc(msg)
			} else {
				c.warn("received message with ID %v, but we don't have a corresponding outoing call", id)
			}
		} else {
			c.warn("received message with no method nor ID, ignoring")
		}
	} else {
		method := *msg.Method

		// method set = request or notification
		if msg.ID == nil {
			// no ID = notification
			notif := Notification{
				Method: method,
				Params: msg.Params,
			}
			go c.handler.HandleNotification(c, notif)
		} else {
			id := *msg.ID

			// ID set = request
			req := Request{
				ID:     id,
				Method: method,
				Params: msg.Params,
			}
			go func() {
				res, reqErr := c.handler.HandleRequest(c, req)

				if reqErr != nil {
					var err error

					if rpcErr, ok := reqErr.(*Error); ok {
						err = c.replyWithError(id, *rpcErr)
					} else {
						rpcErr := Error{
							Code:    CodeInternalError,
							Message: "internal JSON-RPC 2.0 error",
							Data:    nil,
						}
						err = c.replyWithError(id, rpcErr)
					}
					if err != nil {
						c.warn("while replying with error: %+v", err)
						return
					}
				} else {
					resText, err := EncodeJSON(res)
					if err != nil {
						c.warn("while encoding result as JSON: %+v", err)
						return
					}

					err = c.reply(id, resText)
					if err != nil {
						c.warn("while replying with result: %+v", err)
						return
					}
				}
			}()
		}
	}
}

func (c *connImpl) Notify(method string, params interface{}) error {
	paramsText, err := EncodeJSON(params)
	if err != nil {
		return err
	}

	msg := Message{
		Method: &method,
		Params: &paramsText,
	}
	return c.send(msg)
}

func (c *connImpl) Call(method string, params interface{}, result interface{}) error {
	paramsText, err := EncodeJSON(params)
	if err != nil {
		return err
	}

	id := c.generateID()
	msg := Message{
		ID:     &id,
		Method: &method,
		Params: &paramsText,
	}

	done := make(chan error)

	f := func(msg Message) {
		done <- (func() error {
			if msg.Error != nil {
				return msg.Error
			}

			if msg.Result == nil {
				return errors.New("json-rpc2: invalid response: no 'error' nor 'result' field")
			}

			return DecodeJSON(*msg.Result, result)
		})()
	}
	c.outgoingCallsMutex.Lock()
	c.outgoingCalls[id] = f
	c.outgoingCallsMutex.Unlock()

	err = c.send(msg)
	if err != nil {
		return err
	}

	select {
	case err := <-done:
		return err
	case <-c.ctx.Done():
		return errors.New("json-rpc2: connection closed")
	}
}

func (c *connImpl) DisconnectNotify() chan struct{} {
	return c.disconnectNotify
}

func (c *connImpl) Close() {
	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()

	if !c.closed {
		c.closed = true
		c.cancel()
		c.transport.Close()
		close(c.disconnectNotify)
	}
}

type Message struct {
	// *must* be set to "2.0"
	JsonRPC string `json:"jsonrpc"`

	// can be nil if notification or id-less error
	ID *ID `json:"id,omitempty"`
	// can be nil if result / error
	Method *string `json:"method,omitempty"`
	// can be nil if response
	Params *json.RawMessage `json:"params,omitempty"`
	// can be nil if request / notification
	Result *json.RawMessage `json:"result,omitempty"`
	// can be nil if success
	Error *Error `json:"error,omitempty"`
}

// A JSON-RPC2 request, see https://www.jsonrpc.org/specification#request_object
type Request struct {
	ID     ID
	Method string
	Params *json.RawMessage
}

// A JSON-RPC2 notification, see https://www.jsonrpc.org/specification#notification
type Notification struct {
	Method string
	Params *json.RawMessage
}

// A Handler reacts to incoming requests or notifications
type Handler interface {
	// Handle a request, returning either a result or an error (but not both)
	// If both are returned, a non-nil Error takes priority
	HandleRequest(conn Conn, req Request) (interface{}, error)

	// Handle a notification, returning nothing.
	HandleNotification(conn Conn, notif Notification)
}

// Encode anything as JSON, but marshal nil slices as [],
// and nil maps as {}
func EncodeJSON(v interface{}) (json.RawMessage, error) {
	res, err := json.MarshalSafeCollections(v)
	rawMsg := json.RawMessage(res)
	return rawMsg, err
}

// Decode anything from JSON
func DecodeJSON(raw json.RawMessage, v interface{}) error {
	return json.Unmarshal(raw, v)
}
