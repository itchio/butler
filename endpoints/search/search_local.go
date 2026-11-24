package search

import (
	"sync"

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

// localSearch answers Search.Local from an in-memory index of one
// profile's library, rebuilt when the library has changed since it was
// built, or when another profile searches.
type localSearch struct {
	mu        sync.Mutex
	profileID int64
	idx       *localIndex
}

func (s *localSearch) SearchLocal(rc *butlerd.RequestContext, params butlerd.SearchLocalParams) (*butlerd.SearchLocalResult, error) {
	res := &butlerd.SearchLocalResult{}
	if params.Query == "" {
		return res, nil
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		res = s.search(conn, params.ProfileID, params.Query)
	})

	return res, nil
}

func (s *localSearch) index(conn *sqlite.Conn, profileID int64) *localIndex {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.idx == nil || s.profileID != profileID || s.idx.generation != models.LibraryGeneration() {
		s.idx = buildLocalIndex(conn, profileID)
		s.profileID = profileID
	}
	return s.idx
}

func (s *localSearch) search(conn *sqlite.Conn, profileID int64, query string) *butlerd.SearchLocalResult {
	idx := s.index(conn, profileID)
	q := newLocalQuery(query)

	res := &butlerd.SearchLocalResult{}
	res.Games = selectRanked(conn, q.rank(idx.games, searchLocalGamesLimit),
		func(g *itchio.Game) int64 { return g.ID })
	res.Bundles = selectRanked(conn, q.rank(idx.bundles, searchLocalBundlesLimit),
		func(b *itchio.Bundle) int64 { return b.ID })
	res.Collections = selectRanked(conn, q.rank(idx.collections, searchLocalCollectionsLimit),
		func(c *itchio.Collection) int64 { return c.ID })
	return res
}

// selectRanked fetches the records with the given ids, in that order.
func selectRanked[T any](conn *sqlite.Conn, ids []int64, id func(*T) int64) []*T {
	if len(ids) == 0 {
		return nil
	}

	var rows []*T
	models.MustSelect(conn, &rows, builder.Eq{"id": ids}, hades.Search{})

	byID := make(map[int64]*T, len(rows))
	for _, row := range rows {
		byID[id(row)] = row
	}

	ranked := make([]*T, 0, len(rows))
	for _, rowID := range ids {
		if row, ok := byID[rowID]; ok {
			ranked = append(ranked, row)
		}
	}
	return ranked
}
