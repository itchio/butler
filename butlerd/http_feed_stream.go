package butlerd

import (
	"context"
	"fmt"
	"net/http"
)

type responseWriter interface {
	http.ResponseWriter
	http.Flusher
}

type httpFeedStream struct {
	cid string

	w  responseWriter
	r  *http.Request
	hh *httpHandler

	ctx    context.Context
	cancel context.CancelFunc

	id int64

	requestCh chan []byte
}

func (s *httpFeedStream) emit(data string) error {
	id := s.id
	s.id++
	return s.emitMsg(fmt.Sprintf("id: %d\ndata: %s", id, data))
}

var twoLf = []byte{'\n', '\n'}

func (s *httpFeedStream) emitMsg(data string) error {
	_, err := s.w.Write([]byte(data))
	if err != nil {
		return err
	}
	_, err = s.w.Write(twoLf)
	if err != nil {
		return err
	}

	s.w.(http.Flusher).Flush()

	return nil
}

func (s *httpFeedStream) Wait(parentCtx context.Context) error {
	s.requestCh = make(chan []byte)

	s.hh.putFeedStream(s.cid, s)
	defer func() {
		s.hh.removeFeedStream(s.cid)
		if cs, ok := s.hh.getCallStream(s.cid); ok {
			cs.cancelWith(424, "Feed closed while server was awaiting response")
		}
	}()

	ctx, cancel := context.WithCancel(parentCtx)
	s.ctx = ctx
	s.cancel = cancel
	defer cancel()

	s.w.Header().Set("content-type", "text/event-stream")
	s.w.Header().Set("cache-control", "no-cache")
	s.w.WriteHeader(200)

	err := s.emitMsg("event: open")
	if err != nil {
		return err
	}

	for {
		select {
		case payload := <-s.requestCh:
			s.emit(string(payload))
		case <-s.ctx.Done():
			return nil
		}
	}
}
