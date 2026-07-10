package search

import (
	"strings"

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

// relevanceSearch matches titles containing the query, ranked before the
// limit truncates: exact matches first, then prefix matches, then other
// substring matches. Within a tier, shorter titles rank higher (the query
// makes up more of the title) and id breaks ties so the order is
// deterministic.
func relevanceSearch(query string, limit int64) (builder.Cond, hades.Search) {
	q := strings.ToLower(query)
	cond := builder.Like{"lower(title)", "%" + q + "%"}
	search := hades.Search{}.OrderBy(
		"(lower(title) = ?) desc, (lower(title) like ?) desc, length(title) asc, id asc",
		q, q+"%",
	).Limit(limit)
	return cond, search
}

func searchLocalGames(conn *sqlite.Conn, query string) []*itchio.Game {
	var games []*itchio.Game
	cond, search := relevanceSearch(query, searchLocalGamesLimit)
	models.MustSelect(conn, &games, cond, search)
	return games
}

func searchLocalBundles(conn *sqlite.Conn, profileID int64, query string) []*itchio.Bundle {
	var bundles []*itchio.Bundle
	cond, search := relevanceSearch(query, searchLocalBundlesLimit)
	models.MustSelect(conn, &bundles,
		builder.And(cond, builder.Expr(
			"exists (select 1 from bundle_keys where bundle_keys.bundle_id = bundles.id and bundle_keys.owner_id = ?)",
			profileID,
		)),
		search,
	)
	return bundles
}

func searchLocalCollections(conn *sqlite.Conn, profileID int64, query string) []*itchio.Collection {
	var collections []*itchio.Collection
	cond, search := relevanceSearch(query, searchLocalCollectionsLimit)
	models.MustSelect(conn, &collections,
		builder.And(cond, builder.Expr(
			"exists (select 1 from profile_collections where profile_collections.collection_id = collections.id and profile_collections.profile_id = ?)",
			profileID,
		)),
		search,
	)
	return collections
}
