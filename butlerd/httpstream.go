package butlerd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/sourcegraph/jsonrpc2"
)

type httpHandler struct {
	jrh jsonrpc2.Handler
}

var _ http.Handler = (*httpHandler)(nil)

func (hh *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := httpStream{
		w:   w,
		r:   r,
		jrh: hh.jrh,
	}
	s.Wait(ctx)
}

//----------------------------------------

type httpStream struct {
	w           http.ResponseWriter
	r           *http.Request
	jrh         jsonrpc2.Handler
	initialRead bool
	id          int64
	ctx         context.Context
	cancel      context.CancelFunc
}

var _ jsonrpc2.ObjectStream = (*httpStream)(nil)
var _ jsonrpc2.Handler = (*httpStream)(nil)

func (s *httpStream) ReadObject(v interface{}) error {
	log.Printf("ReadObject called")
	if !s.initialRead {
		log.Printf("ReadObject initialRead")
		s.initialRead = true

		body, err := ioutil.ReadAll(s.r.Body)
		if err != nil {
			return err
		}
		log.Printf("ReadObject body = %s", string(body))

		err = json.Unmarshal(body, v)
		if err != nil {
			return err
		}

		return nil
	}

	<-s.ctx.Done()
	return io.EOF
}

func (s *httpStream) WriteObject(obj interface{}) error {
	log.Printf("WriteObject called")

	marshalled, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	log.Printf("WriteObject marshalled: %s", string(marshalled))

	return s.emit(string(marshalled))
}

func (s *httpStream) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	defer func() {
		log.Printf("Done handling!")
		s.cancel()
	}()
	log.Printf("Handling...")
	s.jrh.Handle(ctx, conn, req)
}

func (s *httpStream) Close() error {
	s.cancel()
	return nil
}

func (s *httpStream) emit(data string) error {
	payload := fmt.Sprintf("id: %d\ndata: %s\n\n", s.id, data)
	s.id++
	_, err := s.w.Write([]byte(payload))
	return err
}

func (s *httpStream) Wait(parentCtx context.Context) {
	if s.r.Method != "POST" {
		s.w.WriteHeader(400)
		s.w.Write([]byte("All requests must be POST"))
		return
	}

	ctx, cancel := context.WithCancel(parentCtx)
	s.ctx = ctx
	s.cancel = cancel
	defer cancel()

	s.w.Header().Set("content-type", "text/event-stream")
	s.w.Header().Set("cache-control", "no-cache")
	s.w.WriteHeader(200)

	conn := jsonrpc2.NewConn(ctx, s, s)
	<-conn.DisconnectNotify()
	log.Printf("Got disconnect notify")
}
