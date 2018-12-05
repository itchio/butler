package itchio

import (
	"fmt"
	"net/url"
	"time"
)

// A Query represents an HTTP request made to the itch.io API,
// whether it's GET or POST.
type Query struct {
	Client *Client
	Path   string
	Values url.Values
}

// NewQuery creates a new query with a given formatted path,
// attached to a specific client (for http transport, retry logic,
// credentials)
func NewQuery(c *Client, format string, a ...interface{}) *Query {
	return &Query{
		Client: c,
		Path:   fmt.Sprintf(format, a...),
		Values: make(url.Values),
	}
}

// AddValues adds all parameters of values to this query
func (q *Query) AddValues(values url.Values) {
	for k, vv := range values {
		for _, v := range vv {
			q.Values.Add(k, v)
		}
	}
}

// AddBoolIfTrue adds the parameter key=true only if value is true.
func (q *Query) AddBoolIfTrue(key string, value bool) {
	if value {
		q.Values.Add(key, "")
	}
}

// AddString adds the parameter key=value even if the value is empty.
func (q *Query) AddString(key string, value string) {
	q.Values.Add(key, value)
}

// AddStringIfNonEmpty adds the parameter key=value only if the value is non-empty.
func (q *Query) AddStringIfNonEmpty(key string, value string) {
	if value != "" {
		q.AddString(key, value)
	}
}

// AddTimePtr adds param key=value only if value is not a nil pointer
func (q *Query) AddTimePtr(key string, value *time.Time) {
	if value != nil {
		q.AddTime(key, *value)
	}
}

// AddTime adds param key=value, with value formatted as RFC-3339 Nano
func (q *Query) AddTime(key string, value time.Time) {
	q.AddString(key, value.Format(time.RFC3339Nano))
}

// AddInt64 adds param key=value, even if value is 0
func (q *Query) AddInt64(key string, value int64) {
	q.Values.Add(key, fmt.Sprintf("%d", value))
}

// AddInt64IfNonZero adds param key=value, only if value is non-zero
func (q *Query) AddInt64IfNonZero(key string, value int64) {
	if value != 0 {
		q.AddInt64(key, value)
	}
}

// AddAPICredentials adds the api_key= parameter from the client's key
func (q *Query) AddAPICredentials() {
	q.Values.Add("api_key", q.Client.Key)
}

// AddGameCredentials adds the download_key_id, password, and secret
// parameters if they're non-empty in the passed GameCredentials.
func (q *Query) AddGameCredentials(gc GameCredentials) {
	q.AddInt64IfNonZero("download_key_id", gc.DownloadKeyID)
	q.AddStringIfNonEmpty("password", gc.Password)
	q.AddStringIfNonEmpty("secret", gc.Secret)
}

// URL returns the full path for this query, as if it was a GET
// request (all parameters are encoded into the request URL)
func (q *Query) URL() string {
	return q.Client.MakeValuesPath(q.Values, q.Path)
}

// Get performs this query as an HTTP GET request with the tied client.
// Params are URL-encoded and added to the path, see URL().
func (q *Query) Get(r interface{}) error {
	return q.Client.GetResponse(q.URL(), r)
}

// Post performs this query as an HTTP POST request with the tied client.
// Parameters are URL-encoded and passed as the body of the POST request.
func (q *Query) Post(r interface{}) error {
	url := q.Client.MakePath(q.Path)
	return q.Client.PostFormResponse(url, q.Values, r)
}
