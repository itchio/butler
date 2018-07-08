package butlerd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

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
	tries  int64

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

	var method string
	methodObj, hasMethod := intermediate["method"]
	if hasMethod {
		method = methodObj.(string)
	} else {
		// then it must be a notification
		allowFailures = true
	}

	if s.cid == "" {
		errMsg := fmt.Sprintf("Server tried to call '%s', but no CID was specified ('X-CID' header is not set)", method)
		return HTTPError(428, errMsg)
	}

	// notifications or server-side requests are sent to event-stream
	var fs *httpFeedStream
	var ok bool

	fs, ok = s.hh.getFeedStream(s.cid)
	if !ok {
		if allowFailures {
			return nil
		}
		// allow 200ms of slack
		for s.tries < 20 {
			s.tries++
			fs, ok = s.hh.getFeedStream(s.cid)
			if ok {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	if !ok {
		if allowFailures {
			return nil
		}
		errMsg := fmt.Sprintf("Server tried to call '%s', but nobody is listening to the feed for CID '%s'", method, s.cid)
		return HTTPError(428, errMsg)
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
	idString := s.r.Header.Get("x-id")
	if idString == "" {
		return HTTPError(400, "Missing request ID x-id")
	}
	id, err := strconv.ParseInt(idString, 10, 64)
	if err != nil {
		return HTTPError(400, "x-id must be an integer")
	}

	body, err := ioutil.ReadAll(s.r.Body)
	if err != nil {
		return err
	}

	s.readCh = make(chan []byte, 1)
	req := map[string]interface{}{
		"id":     id,
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
