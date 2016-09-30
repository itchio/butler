package itchfs

import (
	"fmt"
	"net/http"
	"net/url"

	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/httpfile"
)

type ItchFS struct {
	ItchServer string
}

func (ifs *ItchFS) Scheme() string {
	return "itchfs"
}

func needsRenewal(req *http.Request) bool {
	// FIXME: stub
	return true
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

	itchClient := itchio.ClientWithKey(apiKey)
	if ifs.ItchServer != "" {
		itchClient.SetServer(ifs.ItchServer)
	}

	source, err := ObtainSource(itchClient, u.Path)
	if err != nil {
		return nil, nil, err
	}

	getURL, err := source.makeGetURL()
	if err != nil {
		return nil, nil, err
	}

	return getURL, needsRenewal, nil
}
