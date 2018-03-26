package operate

import (
	"github.com/pkg/errors"
	"github.com/itchio/butler/butlerd"
	itchio "github.com/itchio/go-itchio"
)

func ClientFromCredentials(credentials *butlerd.GameCredentials) (*itchio.Client, error) {
	if credentials == nil {
		return nil, errors.New("Missing credentials")
	}

	if credentials.APIKey == "" {
		return nil, errors.New("Missing API key in credentials")
	}

	client := itchio.ClientWithKey(credentials.APIKey)

	if credentials.Server != "" {
		client.SetServer(credentials.Server)
	}

	return client, nil
}
