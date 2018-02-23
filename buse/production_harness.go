package buse

import (
	"errors"

	itchio "github.com/itchio/go-itchio"
)

// productionHarness

type productionHarness struct {
}

var _ Harness = (*productionHarness)(nil)

func NewProductionHarness() Harness {
	return &productionHarness{}
}

func (ph *productionHarness) ClientFromCredentials(credentials *GameCredentials) (*itchio.Client, error) {
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
