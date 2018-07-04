package butlerd

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/sourcegraph/jsonrpc2"
)

type httpCallStream struct {
	cid    string
	method string

	w  http.ResponseWriter
	r  *http.Request
	hh *httpHandler

	jrh jsonrpc2.Handler

	ctx    context.Context
	cancel context.CancelFunc

	readCh chan []byte
}

var _ jsonrpc2.ObjectStream = (*httpCallStream)(nil)
var _ jsonrpc2.Handler = (*httpCallStream)(nil)

func (s *httpCallStream) ReadObject(v interface{}) error {
	select {
	case msg := <-s.readCh:
		return json.Unmarshal(msg, v)
	case <-s.ctx.Done():
		return io.EOF
	}
}

func (s *httpCallStream) WriteObject(obj interface{}) error {
	marshalled, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	intermediate := make(map[string]interface{})

	err = json.Unmarshal(marshalled, &intermediate)
	if err != nil {
		return err
	}

	_, hasError := intermediate["error"]
	_, hasResult := intermediate["result"]
	if hasError || hasResult {
		// responses are written as http responses
		s.w.Header().Set("content-type", "application/json")
		s.w.Header().Set("cache-control", "no-cache")
		s.w.WriteHeader(200)
		s.w.Write(marshalled)
		return nil
	}

	allowFailures := false
	_, hasMethod := intermediate["method"]
	if !hasMethod {
		// then it must be a notification
		allowFailures = true
	}

	if s.cid == "" {
		return HTTPError(428, "Server tried to send a request, but cid was not specified for this call")
	}

	// notifications or server-side requests are sent to event-stream
	fs, ok := s.hh.getFeedStream(s.cid)
	if !ok {
		if allowFailures {
			return nil
		}
		return HTTPError(428, "Server tried to send a request, but client is not listening to feed")
	}

	fs.requestCh <- marshalled
	return nil
}

func (s *httpCallStream) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	// handle asynchronously so we can process server->client requests
	go func() {
		defer s.cancel()
		s.jrh.Handle(ctx, conn, req)
	}()
}

func (s *httpCallStream) Close() error {
	s.cancel()
	return nil
}

func (s *httpCallStream) Wait(parentCtx context.Context) error {
	body, err := ioutil.ReadAll(s.r.Body)
	if err != nil {
		return err
	}

	s.readCh = make(chan []byte, 1)
	req := map[string]interface{}{
		"id":     0,
		"method": s.method,
		"params": json.RawMessage(body),
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return err
	}
	s.readCh <- reqJSON

	s.hh.putCallStream(s.cid, s)
	defer s.hh.removeCallStream(s.cid)

	ctx, cancel := context.WithCancel(parentCtx)
	s.ctx = ctx
	s.cancel = cancel
	defer cancel()

	conn := jsonrpc2.NewConn(ctx, s, s)
	<-conn.DisconnectNotify()
	return nil
}

func (s *httpCallStream) cancelWith(status int, msg string) {
	s.w.WriteHeader(status)
	s.w.Write([]byte(msg))
	s.cancel()
}
