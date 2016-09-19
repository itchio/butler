package netpool

import (
	"fmt"
	"io"
	"net/http"
)

type HttpSource struct {
	BaseURL string
}

var _ Source = (*HttpSource)(nil)

func (hs *HttpSource) Open(key string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/%s", hs.BaseURL, key)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Upstream returned HTTP %d", res.StatusCode)
	}

	return res.Body, nil
}
