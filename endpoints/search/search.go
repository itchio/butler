package search

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
)

func Register(router *butlerd.Router) {
	messages.SearchLocal.Register(router, SearchLocal)
	messages.SearchGames.Register(router, SearchGames)
	messages.SearchUsers.Register(router, SearchUsers)
}
