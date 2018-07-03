package butlerd

import (
	"log"
	"net/http"
	"sync"

	"github.com/sourcegraph/jsonrpc2"
)

type httpHandler struct {
	jrh jsonrpc2.Handler

	feedStreams      map[string]*httpFeedStream
	feedStreamsMutex sync.Mutex

	callStreams      map[int64]*httpCallStream
	callStreamsMutex sync.Mutex

	handlerFunc http.HandlerFunc

	callHandlerId int64
}

var _ http.Handler = (*httpHandler)(nil)

func (hh *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if hh.handlerFunc == nil {
		hh.handlerFunc = H(hh.handle)
	}
	if hh.feedStreams == nil {
		hh.feedStreams = make(map[string]*httpFeedStream)
	}
	if hh.callStreams == nil {
		hh.callStreams = make(map[int64]*httpCallStream)
	}

	hh.handlerFunc(w, r)
}

func (hh *httpHandler) getFeedStream(key string) (*httpFeedStream, bool) {
	hh.feedStreamsMutex.Lock()
	defer hh.feedStreamsMutex.Unlock()

	s, ok := hh.feedStreams[key]
	return s, ok
}

func (hh *httpHandler) putFeedStream(key string, s *httpFeedStream) {
	hh.feedStreamsMutex.Lock()
	defer hh.feedStreamsMutex.Unlock()

	hh.feedStreams[key] = s
}

func (hh *httpHandler) removeFeedStream(key string) {
	hh.feedStreamsMutex.Lock()
	defer hh.feedStreamsMutex.Unlock()

	delete(hh.feedStreams, key)
}

func (hh *httpHandler) getCallStream(key int64) (*httpCallStream, bool) {
	hh.callStreamsMutex.Lock()
	defer hh.callStreamsMutex.Unlock()

	s, ok := hh.callStreams[key]
	return s, ok
}

func (hh *httpHandler) putCallStream(key int64, s *httpCallStream) {
	hh.callStreamsMutex.Lock()
	defer hh.callStreamsMutex.Unlock()

	hh.callStreams[key] = s
}

func (hh *httpHandler) removeCallStream(key int64) {
	hh.callStreamsMutex.Lock()
	defer hh.callStreamsMutex.Unlock()

	delete(hh.callStreams, key)
}

func (hh *httpHandler) handle(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	switch r.Method {
	case "GET":
		s := &httpFeedStream{
			w:         w,
			r:         r,
			requestCh: make(chan requestMsg),
		}
		key := ""
		hh.putFeedStream(key, s)
		defer hh.removeFeedStream(key)
		return s.Wait(ctx)
	case "POST":
		cid := hh.callHandlerId
		hh.callHandlerId++
		s := &httpCallStream{
			w:   w,
			r:   r,
			hh:  hh,
			jrh: hh.jrh,
			cid: cid,
		}
		hh.putCallStream(cid, s)
		defer hh.removeCallStream(cid)
		return s.Wait(ctx)
	default:
		log.Printf("Called with invalid method: %s", r.Method)
		return HTTPError(400, "Expected GET or POST")
	}
}
