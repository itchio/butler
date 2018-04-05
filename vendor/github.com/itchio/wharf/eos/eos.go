// eos stands for 'enhanced os', it mostly supplies 'eos.Open', which supports
// the 'itchfs://' scheme to access remote files
package eos

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/itchio/httpkit/httpfile"
	"github.com/itchio/httpkit/retrycontext"
	"github.com/itchio/wharf/eos/option"
	"github.com/pkg/errors"
)

var httpFileLogLevel = os.Getenv("HTTPFILE_DEBUG")
var httpFileCheck = os.Getenv("HTTPFILE_CHECK") == "1"

type File interface {
	io.Reader
	io.Closer
	io.ReaderAt
	io.Seeker

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
	f, err := realOpen(name, opts...)
	if err != nil {
		return nil, err
	}

	if hf, ok := f.(*httpfile.HTTPFile); ok && httpFileCheck {
		hf.ForbidBacktracking = true

		f2, err := realOpen(name, opts...)
		if err != nil {
			return nil, err
		}

		return &CheckingFile{
			Reference: f,
			Trainee:   f2,
		}, nil
	}

	return f, err
}

func realOpen(name string, opts ...option.Option) (File, error) {
	settings := option.DefaultSettings()

	for _, opt := range opts {
		opt.Apply(settings)
	}

	if name == "/dev/null" {
		return &emptyFile{}, nil
	}

	u, err := url.Parse(name)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	httpFileSettings := func() *httpfile.Settings {
		return &httpfile.Settings{
			Client: settings.HTTPClient,
			RetrySettings: &retrycontext.Settings{
				MaxTries: settings.MaxTries,
				Consumer: settings.Consumer,
			},
		}
	}

	switch u.Scheme {
	case "http", "https":
		res := &simpleHTTPResource{name}
		hf, err := httpfile.New(res.GetURL, res.NeedsRenewal, httpFileSettings())

		if err != nil {
			return nil, err
		}

		setupHttpFileDebug(hf)

		return hf, nil
	default:
		handler := handlers[u.Scheme]
		if handler == nil {
			return os.Open(name)
		}

		getURL, needsRenewal, err := handler.MakeResource(u)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		hf, err := httpfile.New(getURL, needsRenewal, httpFileSettings())

		if err != nil {
			return nil, err
		}

		setupHttpFileDebug(hf)

		return hf, nil
	}
}

func Redact(name string) string {
	u, err := url.Parse(name)
	if err != nil {
		return name
	}

	return u.Path
}

var hfSeed = 0

func setupHttpFileDebug(hf *httpfile.HTTPFile) {
	hfSeed += 1
	hfIndex := hfSeed

	if httpFileLogLevel != "" {
		hf.Log = func(msg string) {
			fmt.Fprintf(os.Stderr, "[hf%d] %s\n", hfIndex, msg)
		}
		numericLevel, err := strconv.ParseInt(httpFileLogLevel, 10, 64)
		if err == nil {
			hf.LogLevel = int(numericLevel)
		}
	}
}
