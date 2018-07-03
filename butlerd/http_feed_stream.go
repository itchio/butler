package butlerd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type httpFeedStream struct {
	w http.ResponseWriter
	r *http.Request

	ctx    context.Context
	cancel context.CancelFunc
	id     int64

	requestCh chan requestMsg
}

func (s *httpFeedStream) emit(data string) error {
	id := s.id
	s.id++
	return s.emitMsg(fmt.Sprintf("id: %d\ndata: %s", id, data))
}

var twoLf = []byte{'\n', '\n'}

func (s *httpFeedStream) emitMsg(data string) error {
	s.log("emitting %s", data)
	_, err := s.w.Write([]byte(data))
	if err != nil {
		return err
	}
	_, err = s.w.Write(twoLf)
	if err != nil {
		return err
	}

	if flusher, ok := s.w.(http.Flusher); ok {
		flusher.Flush()
	} else {
		s.log("could not flush :(")
	}

	return nil
}

func (s *httpFeedStream) relay() {
	for {
		select {
		case rm := <-s.requestCh:
			s.log("got requestMsg: %s", string(rm.Payload))
			payload, err := json.Marshal(rm)
			if err != nil {
				panic(err)
			}
			s.emit(string(payload))
		case <-s.ctx.Done():
			s.log("relay stopped")
			return
		}
	}
}

func (s *httpFeedStream) Wait(parentCtx context.Context) error {
	ctx, cancel := context.WithCancel(parentCtx)
	s.ctx = ctx
	s.cancel = cancel
	defer cancel()

	s.w.Header().Set("content-type", "text/event-stream")
	s.w.Header().Set("cache-control", "no-cache")
	s.w.WriteHeader(200)

	go s.relay()
	err := s.emitMsg("event: open")
	if err != nil {
		return err
	}

	s.log("Feed opened")
	<-ctx.Done()
	s.log("Feed closed")

	return nil
}

func (s *httpFeedStream) log(format string, a ...interface{}) {
	log.Printf("[%s] %s", "[feed stream]", fmt.Sprintf(format, a...))
}
