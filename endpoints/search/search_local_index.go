package search

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/database/models"
	"golang.org/x/text/unicode/norm"
	"xorm.io/builder"
)

// searchWords folds a title or query down to what matching looks at:
// lowercase letters and digits without accents, each run of anything
// else being a word break. "X-Moon: Pokémon" gives x, moon, pokemon.
//
// Text with no letters or digits at all ("???", emoji) is kept as it is,
// split on spaces, so that it can still be searched for.
func searchWords(s string) []string {
	var words []string
	var word strings.Builder
	flush := func() {
		if word.Len() > 0 {
			words = append(words, word.String())
			word.Reset()
		}
	}
	// NFKD splits marks from their letter. Those are accents to drop in
	// these scripts, but part of the letter or word in most others
	// (Japanese voicing marks, Devanagari vowel signs).
	accented := false
	for _, r := range norm.NFKD.String(s) {
		switch {
		case unicode.IsMark(r):
			if !accented && word.Len() > 0 {
				word.WriteRune(r)
			}
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			accented = unicode.In(r, unicode.Latin, unicode.Greek, unicode.Cyrillic)
			word.WriteRune(unicode.ToLower(r))
		default:
			flush()
		}
	}
	flush()

	if len(words) == 0 {
		return strings.Fields(strings.ToLower(s))
	}
	return words
}

type indexEntry struct {
	id int64
	// the title as it is, lowercased
	title string
	// the title's words run together, so that "xmoon" finds "x-moon"
	key string
	// the title's words, each preceded by a space
	words       string
	titleLength int
}

func newIndexEntry(id int64, title string) indexEntry {
	words := searchWords(title)
	return indexEntry{
		id:          id,
		title:       strings.ToLower(title),
		key:         strings.Join(words, ""),
		words:       " " + strings.Join(words, " "),
		titleLength: utf8.RuneCountInString(title),
	}
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
				entries = append(entries, newIndexEntry(stmt.ColumnInt64(0), stmt.ColumnText(1)))
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

// Match tiers, best first.
const (
	// the query is the title as it is, apart from case
	tierIdentical = iota
	// the query is the whole title
	tierExact
	// the title starts with the query
	tierPrefix
	// every query word starts a word of the title, in any order
	tierWords
	// the query appears inside a word of the title
	tierSubstring
	tierNone
)

type localQuery struct {
	title string
	key   string
	// each word preceded by a space, to match against indexEntry.words
	words []string
}

func newLocalQuery(query string) localQuery {
	words := searchWords(query)
	q := localQuery{
		title: strings.ToLower(strings.TrimSpace(query)),
		key:   strings.Join(words, ""),
	}
	for _, word := range words {
		q.words = append(q.words, " "+word)
	}
	return q
}

func (q localQuery) tier(e *indexEntry) int {
	if strings.HasPrefix(e.key, q.key) {
		if len(e.key) == len(q.key) {
			if e.title == q.title {
				return tierIdentical
			}
			return tierExact
		}
		return tierPrefix
	}
	for _, word := range q.words {
		if !strings.Contains(e.words, word) {
			// words has no space inside a word, and key has none at all
			if strings.Contains(e.words, q.key) {
				return tierSubstring
			}
			return tierNone
		}
	}
	return tierWords
}

type localMatch struct {
	tier  int
	entry *indexEntry
}

// better orders matches by tier. Within a tier, shorter titles rank higher
// (the query makes up more of the title) and id breaks ties so the order
// is deterministic.
func (m localMatch) better(o localMatch) bool {
	if m.tier != o.tier {
		return m.tier < o.tier
	}
	if m.entry.titleLength != o.entry.titleLength {
		return m.entry.titleLength < o.entry.titleLength
	}
	return m.entry.id < o.entry.id
}

// rank returns the ids of the best matches for the query, best first.
func (q localQuery) rank(entries []indexEntry, limit int) []int64 {
	if q.key == "" {
		return nil
	}

	// kept sorted, and never longer than limit
	var best []localMatch
	for i := range entries {
		m := localMatch{tier: q.tier(&entries[i]), entry: &entries[i]}
		if m.tier == tierNone {
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
