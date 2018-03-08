package fetch

import (
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
)

func Register(router *buse.Router) {
	messages.FetchGame.Register(router, FetchGame)
	messages.FetchCollection.Register(router, FetchCollection)
	messages.FetchProfileCollections.Register(router, FetchProfileCollections)
	messages.FetchProfileGames.Register(router, FetchProfileGames)
	messages.FetchProfileOwnedKeys.Register(router, FetchProfileOwnedKeys)
	messages.FetchCommons.Register(router, FetchCommons)
	messages.FetchCave.Register(router, FetchCave)
	messages.FetchCaves.Register(router, FetchCaves)
	messages.FetchCavesByGameID.Register(router, FetchCavesByGameID)
	messages.FetchCavesByInstallLocationID.Register(router, FetchCavesByInstallLocationID)
}
