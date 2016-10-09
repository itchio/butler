// eos stands for 'enhanced os', it mostly supplies 'eos.Open', which supports
// the 'itchfs://' scheme to access remote files
package eos

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/go-errors/errors"
	"github.com/itchio/httpkit/httpfile"
	"github.com/itchio/wharf/eos/option"
)

type File interface {
	io.Reader
	io.Closer
	io.ReaderAt

	Stat() (os.FileInfo, error)
}

type Handler interface {
	Scheme() string
	MakeResource(u *url.URL) (httpfile.GetURLFunc, httpfile.NeedsRenewalFunc, error)
}

var handlers = make(map[string]Handler)

func RegisterHandler(h Handler) error {
	scheme := h.Scheme()

	if handlers[scheme] != nil {
		return fmt.Errorf("already have a handler for %s:", scheme)
	}

	handlers[h.Scheme()] = h
	return nil
}

func DeregisterHandler(h Handler) {
	delete(handlers, h.Scheme())
}

type simpleHTTPResource struct {
	url string
}

func (shr *simpleHTTPResource) GetURL() (string, error) {
	return shr.url, nil
}

func (shr *simpleHTTPResource) NeedsRenewal(res *http.Response, body []byte) bool {
	return false
}

func Open(name string, opts ...option.Option) (File, error) {
	settings := option.DefaultSettings()

	for _, opt := range opts {
		opt.Apply(settings)
	}

	if name == "/dev/null" {
		return &emptyFile{}, nil
	}

	u, err := url.Parse(name)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	switch u.Scheme {
	case "http", "https":
		res := &simpleHTTPResource{name}
		return httpfile.New(res.GetURL, res.NeedsRenewal, &httpfile.Settings{
			Client: settings.HTTPClient,
		})
	default:
		handler := handlers[u.Scheme]
		if handler == nil {
			return os.Open(name)
		}

		getURL, needsRenewal, err := handler.MakeResource(u)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		return httpfile.New(getURL, needsRenewal, &httpfile.Settings{
			Client: settings.HTTPClient,
		})
	}
}
