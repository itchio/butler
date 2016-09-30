package option

import (
	"net/http"
	"time"

	"github.com/itchio/wharf/timeout"
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
	client := timeout.NewClient(time.Second*time.Duration(5), time.Second*time.Duration(5))
	return client
}

//////////////////////////////////////

type Option interface {
	Apply(*EOSSettings)
}
