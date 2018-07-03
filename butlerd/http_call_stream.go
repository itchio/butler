package butlerd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
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
	s.log("ReadObject called")

	select {
	case msg := <-s.readCh:
		s.log("ReadObject msg: %s", string(msg))
		return json.Unmarshal(msg, v)
	case <-s.ctx.Done():
		s.log("ReadObject EOF")
		return io.EOF
	}
}

type requestMsg struct {
	CID     int64           `json:"cid"`
	Payload json.RawMessage `json:"payload"`
}

func (s *httpCallStream) WriteObject(obj interface{}) error {
	s.log("WriteObject called with type %v", reflect.TypeOf(obj))

	marshalled, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	s.log("WriteObject marshalled: %s", string(marshalled))

	intermediate := make(map[string]interface{})

	err = json.Unmarshal(marshalled, &intermediate)
	if err != nil {
		return err
	}

	_, hasError := intermediate["error"]
	_, hasResult := intermediate["result"]
	if hasError || hasResult {
		s.log("It's a response!")

		s.w.Header().Set("content-type", "application/json")
		s.w.Header().Set("cache-control", "no-cache")
		s.w.WriteHeader(200)
		s.w.Write(marshalled)
		return nil
	}

	s.log("Must be a notification or a request, trying to send to feed...")
	fs, ok := s.hh.getFeedStream("")
	if !ok {
		return HTTPError(428, "Need to be listening on feed")
	}

	fs.requestCh <- requestMsg{
		CID:     s.cid,
		Payload: json.RawMessage(marshalled),
	}
	s.log("Sent to feed!")
	return nil
}

func (s *httpCallStream) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	go func() {
		defer s.cancel()

		s.log("Handling...")
		s.jrh.Handle(ctx, conn, req)
		s.log("Done handling!")
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
	s.log("method = %s", method)

	if method == "@Reply" {
		s.log("it's a reply!")
		var rm requestMsg
		err := json.Unmarshal(body, &rm)
		if err != nil {
			return err
		}
		s.log("unmarshalled reply: %#v", rm)
		cs, ok := s.hh.getCallStream(rm.CID)
		if !ok {
			return HTTPError(404, "callstream not found")
		}
		s.log("found callstream, sending payload %s", string(rm.Payload))
		cs.readCh <- rm.Payload
		s.w.WriteHeader(200)
		return nil
	}

	s.enqueue(&jsonrpc2.Request{
		ID:     jsonrpc2.ID{Num: 0},
		Method: method,
		Params: &msg,
	})

	s.log("Starting jsonrpc2 conn")
	conn := jsonrpc2.NewConn(ctx, s, s)
	<-conn.DisconnectNotify()
	s.log("Got disconnect notify")
	return nil
}

func (s *httpCallStream) log(format string, a ...interface{}) {
	log.Printf("[%s] %s", "[call stream]", fmt.Sprintf(format, a...))
}
