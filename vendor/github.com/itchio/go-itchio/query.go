package itchio

import (
	"fmt"
	"net/url"
)

type Query struct {
	Client *Client
	Path   string
	Values url.Values
}

func NewQuery(c *Client, format string, a ...interface{}) *Query {
	return &Query{
		Client: c,
		Path:   fmt.Sprintf(format, a...),
		Values: make(url.Values),
	}
}

func (q *Query) AddValues(values url.Values) {
	for k, vv := range values {
		for _, v := range vv {
			q.Values.Add(k, v)
		}
	}
}

func (q *Query) AddBoolIfTrue(key string, value bool) {
	if value {
		q.Values.Add(key, "true")
	}
}

func (q *Query) AddString(key string, value string) {
	q.Values.Add(key, value)
}

func (q *Query) AddStringIfNonEmpty(key string, value string) {
	if value != "" {
		q.AddString(key, value)
	}
}

func (q *Query) AddInt64(key string, value int64) {
	q.Values.Add(key, fmt.Sprintf("%d", value))
}

func (q *Query) AddInt64IfNonZero(key string, value int64) {
	if value != 0 {
		q.AddInt64(key, value)
	}
}

func (q *Query) AddAPICredentials() {
	q.Values.Add("api_key", q.Client.Key)
}

func (q *Query) AddGameCredentials(gc GameCredentials) {
	q.AddInt64IfNonZero("download_key_id", gc.DownloadKeyID)
	q.AddStringIfNonEmpty("password", gc.Password)
	q.AddStringIfNonEmpty("secret", gc.Secret)
}

func (q *Query) URL() string {
	return q.Client.MakeValuesPath(q.Values, q.Path)
}

func (q *Query) Get(r interface{}) error {
	return q.Client.GetResponse(q.URL(), r)
}

func (q *Query) Post(r interface{}) error {
	url := q.Client.MakePath(q.Path)
	return q.Client.PostFormResponse(url, q.Values, r)
}
