package fetch

import (
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
)

func Register(router *buse.Router) {
	messages.FetchGame.Register(router, FetchGame)
	messages.FetchCollection.Register(router, FetchCollection)
	messages.FetchMyCollections.Register(router, FetchMyCollections)
	messages.FetchMyGames.Register(router, FetchMyGames)
	messages.FetchMyOwnedKeys.Register(router, FetchMyOwnedKeys)
}
