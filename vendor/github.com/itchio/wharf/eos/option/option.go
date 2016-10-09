package option

import (
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
	return client
}

//////////////////////////////////////

type Option interface {
	Apply(*EOSSettings)
}
