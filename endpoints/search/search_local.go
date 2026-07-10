package search

import (
	"fmt"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"xorm.io/builder"
)

const (
	searchLocalGamesLimit       = 4
	searchLocalBundlesLimit     = 3
	searchLocalCollectionsLimit = 3
)

func SearchLocal(rc *butlerd.RequestContext, params butlerd.SearchLocalParams) (*butlerd.SearchLocalResult, error) {
	res := &butlerd.SearchLocalResult{}
	if params.Query == "" {
		return res, nil
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		res.Games = searchLocalGames(conn, params.Query)
		res.Bundles = searchLocalBundles(conn, params.ProfileID, params.Query)
		res.Collections = searchLocalCollections(conn, params.ProfileID, params.Query)
	})

	return res, nil
}

func likeQuery(query string) string {
	return fmt.Sprintf("%%%s%%", query)
}

func searchLocalGames(conn *sqlite.Conn, query string) []*itchio.Game {
	var games []*itchio.Game
	models.MustSelect(conn, &games,
		builder.Like{"lower(title)", likeQuery(query)},
		hades.Search{}.Limit(searchLocalGamesLimit),
	)
	return games
}

func searchLocalBundles(conn *sqlite.Conn, profileID int64, query string) []*itchio.Bundle {
	var bundles []*itchio.Bundle
	models.MustSelect(conn, &bundles,
		builder.And(
			builder.Like{"lower(title)", likeQuery(query)},
			builder.Expr(
				"exists (select 1 from bundle_keys where bundle_keys.bundle_id = bundles.id and bundle_keys.owner_id = ?)",
				profileID,
			),
		),
		hades.Search{}.Limit(searchLocalBundlesLimit),
	)
	return bundles
}

func searchLocalCollections(conn *sqlite.Conn, profileID int64, query string) []*itchio.Collection {
	var collections []*itchio.Collection
	models.MustSelect(conn, &collections,
		builder.And(
			builder.Like{"lower(title)", likeQuery(query)},
			builder.Expr(
				"exists (select 1 from profile_collections where profile_collections.collection_id = collections.id and profile_collections.profile_id = ?)",
				profileID,
			),
		),
		hades.Search{}.Limit(searchLocalCollectionsLimit),
	)
	return collections
}
