package option

import (
	"errors"
	"net/http"
	"time"

	"github.com/itchio/httpkit/timeout"
	"github.com/itchio/wharf/state"
)

type EOSSettings struct {
	HTTPClient     *http.Client
	Consumer       *state.Consumer
	MaxTries       int
	ForceHTFSCheck bool
	HTFSDumpStats  bool
}

var defaultConsumer *state.Consumer

func init() {
	defaultHTTPClient = timeout.NewClient(time.Second*time.Duration(20), time.Second*time.Duration(10))
	setupHTTPClient(defaultHTTPClient)
}

func SetDefaultConsumer(consumer *state.Consumer) {
	defaultConsumer = consumer
}

var defaultHTTPClient *http.Client

func SetDefaultHTTPClient(c *http.Client) {
	setupHTTPClient(c)
	defaultHTTPClient = c
}

func DefaultSettings() *EOSSettings {
	return &EOSSettings{
		HTTPClient: defaultHTTPClient,
		Consumer:   defaultConsumer,
		MaxTries:   2,
	}
}

func setupHTTPClient(c *http.Client) {
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
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
}

//////////////////////////////////////

type Option interface {
	Apply(*EOSSettings)
}

//

type httpClientOption struct {
	client *http.Client
}

func (o *httpClientOption) Apply(settings *EOSSettings) {
	settings.HTTPClient = o.client
}

func WithHTTPClient(client *http.Client) Option {
	return &httpClientOption{client}
}

//

type consumerOption struct {
	consumer *state.Consumer
}

func (o *consumerOption) Apply(settings *EOSSettings) {
	settings.Consumer = o.consumer
}
func WithConsumer(consumer *state.Consumer) Option {
	return &consumerOption{consumer}
}

//

type maxTriesOption struct {
	maxTries int
}

func (o *maxTriesOption) Apply(settings *EOSSettings) {
	settings.MaxTries = o.maxTries
}

func WithMaxTries(maxTries int) Option {
	return &maxTriesOption{maxTries}
}

//

type htfsCheckOption struct{}

func (o *htfsCheckOption) Apply(settings *EOSSettings) {
	settings.ForceHTFSCheck = true
}

func WithHTFSCheck() Option {
	return &htfsCheckOption{}
}

//

type htfsDumpStatsOption struct{}

func (o *htfsDumpStatsOption) Apply(settings *EOSSettings) {
	settings.HTFSDumpStats = true
}

func WithHTFSDumpStats() Option {
	return &htfsDumpStatsOption{}
}
