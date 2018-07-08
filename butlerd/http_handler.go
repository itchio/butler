package butlerd

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/sourcegraph/jsonrpc2"
)

type httpHandler struct {
	jrh jsonrpc2.Handler

	feedStreams      map[string]*httpFeedStream
	feedStreamsMutex sync.Mutex

	callStreams      map[string]*httpCallStream
	callStreamsMutex sync.Mutex

	handlerFunc http.HandlerFunc

	secret string
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
		hh.callStreams = make(map[string]*httpCallStream)
	}

	hh.handlerFunc(w, r)
}

func (hh *httpHandler) getFeedStream(cid string) (*httpFeedStream, bool) {
	hh.feedStreamsMutex.Lock()
	defer hh.feedStreamsMutex.Unlock()

	s, ok := hh.feedStreams[cid]
	return s, ok
}

func (hh *httpHandler) putFeedStream(cid string, s *httpFeedStream) {
	hh.feedStreamsMutex.Lock()
	defer hh.feedStreamsMutex.Unlock()

	hh.feedStreams[cid] = s
}

func (hh *httpHandler) removeFeedStream(cid string) {
	hh.feedStreamsMutex.Lock()
	defer hh.feedStreamsMutex.Unlock()

	delete(hh.feedStreams, cid)
}

func (hh *httpHandler) getCallStream(cid string) (*httpCallStream, bool) {
	hh.callStreamsMutex.Lock()
	defer hh.callStreamsMutex.Unlock()

	s, ok := hh.callStreams[cid]
	return s, ok
}

func (hh *httpHandler) putCallStream(cid string, s *httpCallStream) {
	hh.callStreamsMutex.Lock()
	defer hh.callStreamsMutex.Unlock()

	hh.callStreams[cid] = s
}

func (hh *httpHandler) removeCallStream(cid string) {
	hh.callStreamsMutex.Lock()
	defer hh.callStreamsMutex.Unlock()

	delete(hh.callStreams, cid)
}

func (hh *httpHandler) handle(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	switch r.Method {
	case "GET":
		{
			if r.URL.Path != "/feed" {
				return HTTPError(404, "Not found")
			}

			secret := r.URL.Query().Get("secret")
			if secret != hh.secret {
				return HTTPError(401, "Missing or invalid authorization")
			}

			cid := r.URL.Query().Get("cid")
			if cid == "" {
				return HTTPError(400, "Missing cid parameter")
			}

			s := &httpFeedStream{
				r:   r,
				w:   w.(responseWriter),
				cid: cid,
				hh:  hh,
			}
			return s.Wait(ctx)
		}
	case "POST":
		{
			secret := r.Header.Get("x-secret")
			if secret != hh.secret {
				return HTTPError(401, "Missing or invalid authorization error")
			}

			cid := r.Header.Get("x-cid")

			path := strings.TrimLeft(r.URL.Path, "/")
			pathTokens := strings.Split(path, "/")

			if len(pathTokens) == 0 {
				return HTTPError(404, "Not found")
			}

			switch pathTokens[0] {
			case "call":
				if len(pathTokens) < 2 {
					return HTTPError(404, "Method missing")
				}
				method := pathTokens[1]

				s := &httpCallStream{
					r:      r,
					w:      w,
					cid:    cid,
					hh:     hh,
					jrh:    hh.jrh,
					method: method,
				}
				return s.Wait(ctx)
			case "cancel":
				cs, err := hh.assertCallStream(r)
				if err != nil {
					return err
				}

				cs.cancel()
				w.WriteHeader(204)
				return nil
			case "reply":
				cs, err := hh.assertCallStream(r)
				if err != nil {
					return err
				}

				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					return err
				}
				cs.readCh <- body
				w.WriteHeader(204)
				return nil
			default:
				return HTTPError(404, "Not found")
			}
		}
	default:
		log.Printf("Called with invalid method: %s", r.Method)
		return HTTPError(400, "Expected GET or POST")
	}
}

func (hh *httpHandler) assertCallStream(r *http.Request) (*httpCallStream, error) {
	cid := r.Header.Get("x-cid")
	if cid == "" {
		return nil, HTTPError(400, "Missing cid")
	}

	cs, ok := hh.getCallStream(cid)
	if !ok {
		return nil, HTTPError(404, fmt.Sprintf("No in-flight request with cid '%s'", cid))
	}

	return cs, nil
}
