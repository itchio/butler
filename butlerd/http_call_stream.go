package butlerd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/sourcegraph/jsonrpc2"
)

type httpCallStream struct {
	w      http.ResponseWriter
	r      *http.Request
	jrh    jsonrpc2.Handler
	ctx    context.Context
	cancel context.CancelFunc
	hh     *httpHandler
	cid    int64

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

type requestMsg struct {
	CID     int64           `json:"cid"`
	Payload json.RawMessage `json:"payload"`
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

	// notifications or server-side requests are sent to event-stream
	// FIXME: we shouldn't hang if the client stops subscribing from the stream
	fs, ok := s.hh.getFeedStream("")
	if !ok {
		return HTTPError(428, "Need to be listening on feed")
	}

	fs.requestCh <- requestMsg{
		CID:     s.cid,
		Payload: json.RawMessage(marshalled),
	}
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

func (s *httpCallStream) enqueue(obj interface{}) {
	bs, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	s.readCh <- bs
}

func (s *httpCallStream) Wait(parentCtx context.Context) error {
	ctx, cancel := context.WithCancel(parentCtx)
	s.ctx = ctx
	s.cancel = cancel
	defer cancel()

	s.readCh = make(chan []byte, 1)

	body, err := ioutil.ReadAll(s.r.Body)
	if err != nil {
		return err
	}
	var msg json.RawMessage = body

	method := strings.TrimLeft(s.r.URL.Path, "/")

	if method == "@Reply" {
		var rm requestMsg
		err := json.Unmarshal(body, &rm)
		if err != nil {
			return err
		}
		cs, ok := s.hh.getCallStream(rm.CID)
		if !ok {
			return HTTPError(404, fmt.Sprintf("Reply to unknown CID (call identifier) %d", rm.CID))
		}
		cs.readCh <- rm.Payload
		s.w.WriteHeader(204)
		return nil
	}

	s.enqueue(&jsonrpc2.Request{
		ID:     jsonrpc2.ID{Num: 0},
		Method: method,
		Params: &msg,
	})

	conn := jsonrpc2.NewConn(ctx, s, s)
	<-conn.DisconnectNotify()
	return nil
}

func (s *httpCallStream) log(format string, a ...interface{}) {
	log.Printf("[%s] %s", "[call stream]", fmt.Sprintf(format, a...))
}
