package service

import (
	"github.com/go-errors/errors"

	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/database"
	"github.com/itchio/butler/endpoints/cleandownloads"
	"github.com/itchio/butler/endpoints/fetch"
	"github.com/itchio/butler/endpoints/install"
	"github.com/itchio/butler/endpoints/launch"
	"github.com/itchio/butler/endpoints/profile"
	"github.com/itchio/butler/endpoints/tests"
	"github.com/itchio/butler/endpoints/update"
	"github.com/itchio/butler/endpoints/utilities"
	"github.com/itchio/butler/mansion"
	"github.com/jinzhu/gorm"
)

var mainRouter *buse.Router

func getRouter(mansionContext *mansion.Context) *buse.Router {
	if mainRouter != nil {
		return mainRouter
	}

	getDB := func() (*gorm.DB, error) {
		dbPath := mansionContext.DBPath
		if dbPath == "" {
			return nil, errors.New("sqlite database path not set (use --dbpath)")
		}

		return database.OpenAndPrepare(dbPath)
	}

	mainRouter = buse.NewRouter(mansionContext, getDB)
	utilities.Register(mainRouter)
	tests.Register(mainRouter)
	update.Register(mainRouter)
	install.Register(mainRouter)
	launch.Register(mainRouter)
	cleandownloads.Register(mainRouter)
	profile.Register(mainRouter)
	fetch.Register(mainRouter)

	messages.EnsureAllRequests(mainRouter)

	return mainRouter
}
