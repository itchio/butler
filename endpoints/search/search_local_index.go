package search

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/database/titlesearch"
	"xorm.io/builder"
)

type indexEntry struct {
	id    int64
	title titlesearch.Title
}

// localIndex holds the titles of everything Search.Local can return for
// one profile, so that a search doesn't have to scan the database.
type localIndex struct {
	// models.LibraryGeneration when the index was built
	generation uint64

	games       []indexEntry
	bundles     []indexEntry
	collections []indexEntry
}

func buildLocalIndex(conn *sqlite.Conn, profileID int64) *localIndex {
	// taken before reading, so a write landing meanwhile forces a rebuild
	idx := &localIndex{generation: models.LibraryGeneration()}

	load := func(table string, cond builder.Cond) []indexEntry {
		var entries []indexEntry
		models.MustExec(conn,
			builder.Select("id", "title").From(table).Where(cond),
			func(stmt *sqlite.Stmt) error {
				entries = append(entries, indexEntry{
					id:    stmt.ColumnInt64(0),
					title: titlesearch.NewTitle(stmt.ColumnText(1)),
				})
				return nil
			},
		)
		return entries
	}

	idx.games = load("games", models.GameInProfileLibraryCond(profileID))
	idx.bundles = load("bundles", builder.Expr(
		"exists (select 1 from bundle_keys where bundle_keys.bundle_id = bundles.id and bundle_keys.owner_id = ?)",
		profileID,
	))
	idx.collections = load("collections", builder.Expr(
		"exists (select 1 from profile_collections where profile_collections.collection_id = collections.id and profile_collections.profile_id = ?)",
		profileID,
	))
	return idx
}

type localMatch struct {
	tier  titlesearch.Tier
	entry *indexEntry
}

// better orders matches by tier. Within a tier, shorter titles rank higher
// (the query makes up more of the title) and id breaks ties so the order
// is deterministic.
func (m localMatch) better(o localMatch) bool {
	if m.tier != o.tier {
		return m.tier < o.tier
	}
	if m.entry.title.Length != o.entry.title.Length {
		return m.entry.title.Length < o.entry.title.Length
	}
	return m.entry.id < o.entry.id
}

// rank returns the ids of the best matches for the query, best first.
func rank(q titlesearch.Query, entries []indexEntry, limit int) []int64 {
	if q.Empty() || limit == 0 {
		return nil
	}

	// kept sorted, and never longer than limit
	var best []localMatch
	for i := range entries {
		m := localMatch{tier: q.Tier(&entries[i].title), entry: &entries[i]}
		if m.tier == titlesearch.TierNone {
			continue
		}
		if len(best) == limit && !m.better(best[limit-1]) {
			continue
		}

		if len(best) < limit {
			best = append(best, m)
		} else {
			best[limit-1] = m
		}
		for j := len(best) - 1; j > 0 && best[j].better(best[j-1]); j-- {
			best[j], best[j-1] = best[j-1], best[j]
		}
	}

	var ids []int64
	for _, m := range best {
		ids = append(ids, m.entry.id)
	}
	return ids
}
