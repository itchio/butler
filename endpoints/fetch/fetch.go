package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
)

func Register(router *buse.Router) {
	messages.FetchGame.Register(router, FetchGame)
	messages.FetchCollection.Register(router, FetchCollection)
	messages.FetchMyCollections.Register(router, FetchMyCollections)
}

func checkCredentials(credentials *buse.FetchCredentials) error {
	if credentials == nil {
		return errors.New("Credentials must be provided")
	}

	if credentials.SessionID == 0 {
		return errors.New("Credentials.SessionID cannot be zero")
	}

	return nil
}
