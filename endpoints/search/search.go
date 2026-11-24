package search

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
)

func Register(router *butlerd.Router) {
	local := &localSearch{}
	messages.SearchLocal.Register(router, local.SearchLocal)
	messages.SearchGames.Register(router, SearchGames)
	messages.SearchUsers.Register(router, SearchUsers)
}
