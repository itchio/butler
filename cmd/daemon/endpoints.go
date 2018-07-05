package daemon

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/endpoints/cleandownloads"
	"github.com/itchio/butler/endpoints/downloads"
	"github.com/itchio/butler/endpoints/fetch"
	"github.com/itchio/butler/endpoints/install"
	"github.com/itchio/butler/endpoints/launch"
	"github.com/itchio/butler/endpoints/meta"
	"github.com/itchio/butler/endpoints/profile"
	"github.com/itchio/butler/endpoints/search"
	"github.com/itchio/butler/endpoints/system"
	"github.com/itchio/butler/endpoints/tests"
	"github.com/itchio/butler/endpoints/update"
	"github.com/itchio/butler/endpoints/utilities"
	"github.com/itchio/butler/mansion"
)

var mainRouter *butlerd.Router

func getRouter(dbPool *sqlite.Pool, mansionContext *mansion.Context) *butlerd.Router {
	if mainRouter != nil {
		return mainRouter
	}

	mainRouter = butlerd.NewRouter(dbPool, mansionContext.NewClient)
	mainRouter.ButlerVersion = mansionContext.Version
	mainRouter.ButlerVersionString = mansionContext.VersionString

	meta.Register(mainRouter)
	utilities.Register(mainRouter)
	tests.Register(mainRouter)
	update.Register(mainRouter)
	install.Register(mainRouter)
	launch.Register(mainRouter)
	cleandownloads.Register(mainRouter)
	profile.Register(mainRouter)
	fetch.Register(mainRouter)
	downloads.Register(mainRouter)
	search.Register(mainRouter)
	system.Register(mainRouter)

	messages.EnsureAllRequests(mainRouter)

	return mainRouter
}
