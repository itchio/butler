package service

import (
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/endpoints/utilities"
	"github.com/itchio/butler/mansion"
)

var mainRouter *buse.Router

func getRouter(mansionContext *mansion.Context) *buse.Router {
	if mainRouter != nil {
		return mainRouter
	}

	mainRouter = &buse.Router{
		Handlers:       make(map[string]buse.RequestHandler),
		MansionContext: mansionContext,
	}

	utilities.Register(mainRouter)

	return mainRouter
}
