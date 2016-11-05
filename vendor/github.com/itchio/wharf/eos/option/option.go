package option

import (
	"errors"
	"net/http"
	"time"

	"github.com/itchio/httpkit/timeout"
)

type EOSSettings struct {
	HTTPClient *http.Client
}

func DefaultSettings() *EOSSettings {
	return &EOSSettings{
		HTTPClient: defaultHTTPClient(),
	}
}

func defaultHTTPClient() *http.Client {
	client := timeout.NewClient(time.Second*time.Duration(30), time.Second*time.Duration(15))
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}

		// forward initial request headers
		// see https://github.com/itchio/itch/issues/965
		ireq := via[0]
		for key, values := range ireq.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		return nil
	}
	return client
}

//////////////////////////////////////

type Option interface {
	Apply(*EOSSettings)
}
