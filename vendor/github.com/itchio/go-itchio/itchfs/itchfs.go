package itchfs

import (
	"fmt"
	"net/http"
	"net/url"

	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/httpkit/httpfile"
)

type ItchFS struct {
	ItchServer string
}

func (ifs *ItchFS) Scheme() string {
	return "itchfs"
}

func needsRenewal(res *http.Response, body []byte) bool {
	if res.StatusCode == 400 {
		// XXX: could parse XML / make sure it's expired URL and not something else,
		// but 400 is a good enough indicator for GCS
		return true
	}
	if res.StatusCode == 403 {
		// 403 is a good indicator for Highwinds - additionally, we could parse the URL
		// and compare the expires timestamp ourselves
		return true
	}
	return false
}

func (ifs *ItchFS) MakeResource(u *url.URL) (httpfile.GetURLFunc, httpfile.NeedsRenewalFunc, error) {
	if u.Host != "" {
		return nil, nil, fmt.Errorf("invalid itchfs URL (must start with itchfs:///): %s", u.String())
	}

	vals := u.Query()

	apiKey := vals.Get("api_key")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("missing API key")
	}

	vals.Del("api_key")

	itchClient := itchio.ClientWithKey(apiKey)
	if ifs.ItchServer != "" {
		itchClient.SetServer(ifs.ItchServer)
	}

	source, err := ObtainSource(itchClient, u.Path, vals)
	if err != nil {
		return nil, nil, err
	}

	getURL, err := source.makeGetURL()
	if err != nil {
		return nil, nil, err
	}

	return getURL, needsRenewal, nil
}
