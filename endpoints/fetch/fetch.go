package fetch

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
)

func Register(router *butlerd.Router) {
	messages.FetchGame.Register(router, FetchGame)
	messages.FetchUser.Register(router, FetchUser)
	messages.FetchSale.Register(router, FetchSale)
	messages.FetchCollection.Register(router, FetchCollection)
	messages.FetchCollectionGames.Register(router, FetchCollectionGames)
	messages.FetchProfileCollections.Register(router, FetchProfileCollections)
	messages.FetchProfileGames.Register(router, FetchProfileGames)
	messages.FetchProfileOwnedKeys.Register(router, FetchProfileOwnedKeys)
	messages.FetchCommons.Register(router, FetchCommons)
	messages.FetchCave.Register(router, FetchCave)
	messages.FetchCaves.Register(router, FetchCaves)
	messages.FetchCavesByGameID.Register(router, FetchCavesByGameID)
	messages.FetchCavesByInstallLocationID.Register(router, FetchCavesByInstallLocationID)
	messages.FetchExpireAll.Register(router, FetchExpireAll)
}
