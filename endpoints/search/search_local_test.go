package search

import (
	"fmt"
	"testing"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
)

func searchLocalTestConn(t *testing.T) *sqlite.Conn {
	conn, err := sqlite.OpenConn("file::memory:?mode=memory", 0)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, models.HadesContext().AutoMigrate(conn))
	return conn
}

// Seeds two profiles:
//   - profile 1 owns bundle 10 ("Cozy Bundle") and has collection 30
//     ("Cozy collection") in their collection list
//   - profile 2 owns bundle 20 ("Cozy Rival Bundle") and has collection 40
//     ("Cozy foreign collection")
//
// Games cover each way a game can be in a profile's library:
//   - game 1: owned by profile 1 (download key)
//   - game 2: owned by profile 2 (download key)
//   - game 3: in profile 1's collection 30
//   - game 4: in profile 1's owned bundle 10
//   - game 5: installed (caves have no profile)
//   - game 6: on profile 1's dashboard
//   - game 7: cached only, e.g. from browsing a game page
func seedSearchLocal(t *testing.T, conn *sqlite.Conn) {
	models.MustSave(conn, &itchio.Game{ID: 1, Title: "Cozy Grove"})
	models.MustSave(conn, &itchio.Game{ID: 2, Title: "Celeste"})
	models.MustSave(conn, &itchio.Game{ID: 3, Title: "Cozy Shelf"})
	models.MustSave(conn, &itchio.Game{ID: 4, Title: "Cozy Pack"})
	models.MustSave(conn, &itchio.Game{ID: 5, Title: "Cozy Cabin"})
	models.MustSave(conn, &itchio.Game{ID: 6, Title: "Cozy Workshop"})
	models.MustSave(conn, &itchio.Game{ID: 7, Title: "Cozy Drifter"})

	models.MustSave(conn, &itchio.DownloadKey{ID: 100, GameID: 1, OwnerID: 1})
	models.MustSave(conn, &itchio.DownloadKey{ID: 200, GameID: 2, OwnerID: 2})

	models.MustSave(conn, &itchio.Bundle{ID: 10, Title: "Cozy Bundle"})
	models.MustSave(conn, &itchio.Bundle{ID: 20, Title: "Cozy Rival Bundle"})
	models.MustSave(conn, &itchio.BundleKey{ID: 1000, BundleID: 10, OwnerID: 1})
	models.MustSave(conn, &itchio.BundleKey{ID: 2000, BundleID: 20, OwnerID: 2})
	models.MustSave(conn, &itchio.BundleGame{BundleID: 10, GameID: 4})

	models.MustSave(conn, &itchio.Collection{ID: 30, Title: "Cozy collection", UserID: 1})
	models.MustSave(conn, &itchio.Collection{ID: 40, Title: "Cozy foreign collection", UserID: 2})
	models.MustSave(conn, &models.ProfileCollection{CollectionID: 30, ProfileID: 1})
	models.MustSave(conn, &models.ProfileCollection{CollectionID: 40, ProfileID: 2})
	models.MustSave(conn, &itchio.CollectionGame{CollectionID: 30, GameID: 3})

	models.MustSave(conn, &models.Cave{ID: "cave-5", GameID: 5})

	models.MustSave(conn, &models.ProfileGame{GameID: 6, ProfileID: 1})
}

func localParams(profileID int64, query string) butlerd.SearchLocalParams {
	return butlerd.SearchLocalParams{ProfileID: profileID, Query: query}
}

func gameIDs(games []*itchio.Game) []int64 {
	var ids []int64
	for _, g := range games {
		ids = append(ids, g.ID)
	}
	return ids
}

func bundleIDs(bundles []*itchio.Bundle) []int64 {
	var ids []int64
	for _, b := range bundles {
		ids = append(ids, b.ID)
	}
	return ids
}

func collectionIDs(collections []*itchio.Collection) []int64 {
	var ids []int64
	for _, c := range collections {
		ids = append(ids, c.ID)
	}
	return ids
}

func Test_SearchLocalGames(t *testing.T) {
	conn := searchLocalTestConn(t)
	s := &localSearch{}
	seedSearchLocal(t, conn)

	// case-insensitive
	require.ElementsMatch(t, []int64{1}, gameIDs(s.search(conn, localParams(1, "GROVE")).Games))
	require.Empty(t, s.search(conn, localParams(1, "nothing matches this")).Games)
}

func Test_SearchLocalGamesScopedToProfile(t *testing.T) {
	conn := searchLocalTestConn(t)
	s := &localSearch{}
	seedSearchLocal(t, conn)

	// owned via download key
	require.ElementsMatch(t, []int64{1}, gameIDs(s.search(conn, localParams(1, "grove")).Games))
	require.Empty(t, s.search(conn, localParams(2, "grove")).Games)
	require.Empty(t, s.search(conn, localParams(1, "celeste")).Games)

	// in one of the profile's collections
	require.ElementsMatch(t, []int64{3}, gameIDs(s.search(conn, localParams(1, "shelf")).Games))
	require.Empty(t, s.search(conn, localParams(2, "shelf")).Games)

	// part of an owned bundle
	require.ElementsMatch(t, []int64{4}, gameIDs(s.search(conn, localParams(1, "pack")).Games))
	require.Empty(t, s.search(conn, localParams(2, "pack")).Games)

	// on the profile's dashboard
	require.ElementsMatch(t, []int64{6}, gameIDs(s.search(conn, localParams(1, "workshop")).Games))
	require.Empty(t, s.search(conn, localParams(2, "workshop")).Games)

	// installs aren't profile-scoped, every profile sees them
	require.ElementsMatch(t, []int64{5}, gameIDs(s.search(conn, localParams(1, "cabin")).Games))
	require.ElementsMatch(t, []int64{5}, gameIDs(s.search(conn, localParams(2, "cabin")).Games))

	// cached from browsing only: no profile sees it
	require.Empty(t, s.search(conn, localParams(1, "drifter")).Games)
	require.Empty(t, s.search(conn, localParams(2, "drifter")).Games)
}

// saveOwnedGame saves a game along with a download key so it's in
// profile 1's library
func saveOwnedGame(conn *sqlite.Conn, id int64, title string) {
	models.MustSave(conn, &itchio.Game{ID: id, Title: title})
	models.MustSave(conn, &itchio.DownloadKey{ID: 100 + id, GameID: id, OwnerID: 1})
}

func Test_SearchLocalGamesRelevanceOrder(t *testing.T) {
	conn := searchLocalTestConn(t)
	s := &localSearch{}
	saveOwnedGame(conn, 1, "A Very Cozy Adventure") // substring, longest
	saveOwnedGame(conn, 2, "So Cozy")               // substring
	saveOwnedGame(conn, 3, "Cozy Grove")            // prefix
	saveOwnedGame(conn, 4, "cozy")                  // exact (case-insensitive)
	saveOwnedGame(conn, 5, "Cozy Den")              // prefix, shorter than Cozy Grove

	// exact first, then prefixes by title length, then substrings by title
	// length; the longest substring match falls off the limit of 4
	require.Equal(t, []int64{4, 5, 3, 2}, gameIDs(s.search(conn, localParams(1, "Cozy")).Games))
}

func Test_SearchLocalGamesOrderIsDeterministic(t *testing.T) {
	conn := searchLocalTestConn(t)
	s := &localSearch{}
	// same tier, same title length: id breaks the tie
	saveOwnedGame(conn, 9, "Cozy Web")
	saveOwnedGame(conn, 4, "Cozy Den")

	require.Equal(t, []int64{4, 9}, gameIDs(s.search(conn, localParams(1, "cozy")).Games))
}

func Test_SearchLocalGamesLimit(t *testing.T) {
	conn := searchLocalTestConn(t)
	s := &localSearch{}
	for i := int64(1); i <= 10; i++ {
		saveOwnedGame(conn, i, fmt.Sprintf("Cozy Game %d", i))
	}

	require.Len(t, s.search(conn, localParams(1, "cozy")).Games, searchLocalGamesLimit)
}

func Test_SearchLocalTypesAndLimit(t *testing.T) {
	conn := searchLocalTestConn(t)
	s := &localSearch{}
	for i := int64(1); i <= 10; i++ {
		saveOwnedGame(conn, i, fmt.Sprintf("Cozy Game %d", i))
	}
	models.MustSave(conn, &itchio.Bundle{ID: 10, Title: "Cozy Bundle"})
	models.MustSave(conn, &itchio.BundleKey{ID: 1000, BundleID: 10, OwnerID: 1})

	params := localParams(1, "cozy")
	params.Types = []butlerd.SearchLocalKind{butlerd.SearchLocalKindGames}
	params.Limit = 8
	res := s.search(conn, params)
	require.Len(t, res.Games, 8)
	require.Empty(t, res.Bundles)

	// the limit applies to each type
	params.Types = nil
	params.Limit = 1
	res = s.search(conn, params)
	require.Len(t, res.Games, 1)
	require.Len(t, res.Bundles, 1)
}

func Test_SearchLocalBundlesScopedToProfile(t *testing.T) {
	conn := searchLocalTestConn(t)
	s := &localSearch{}
	seedSearchLocal(t, conn)

	require.ElementsMatch(t, []int64{10}, bundleIDs(s.search(conn, localParams(1, "cozy")).Bundles))
	require.ElementsMatch(t, []int64{20}, bundleIDs(s.search(conn, localParams(2, "cozy")).Bundles))
	// profile without any bundle keys
	require.Empty(t, s.search(conn, localParams(3, "cozy")).Bundles)
	// owned, but query doesn't match
	require.Empty(t, s.search(conn, localParams(1, "rival")).Bundles)
}

func Test_SearchLocalCollectionsScopedToProfile(t *testing.T) {
	conn := searchLocalTestConn(t)
	s := &localSearch{}
	seedSearchLocal(t, conn)

	require.ElementsMatch(t, []int64{30}, collectionIDs(s.search(conn, localParams(1, "cozy")).Collections))
	require.ElementsMatch(t, []int64{40}, collectionIDs(s.search(conn, localParams(2, "cozy")).Collections))
	require.Empty(t, s.search(conn, localParams(3, "cozy")).Collections)
	require.Empty(t, s.search(conn, localParams(1, "foreign")).Collections)
}

func Test_SearchLocalIgnoresPunctuationAndAccents(t *testing.T) {
	conn := searchLocalTestConn(t)
	s := &localSearch{}
	saveOwnedGame(conn, 1, "x-moon")
	saveOwnedGame(conn, 2, "Pokémon Tribute")

	for _, query := range []string{"x moon", "xmoon", "X_Moon", "moon x", "x-mo"} {
		require.Equal(t, []int64{1}, gameIDs(s.search(conn, localParams(1, query)).Games), query)
	}
	require.Equal(t, []int64{2}, gameIDs(s.search(conn, localParams(1, "pokemon")).Games))
	require.Equal(t, []int64{2}, gameIDs(s.search(conn, localParams(1, "TRIBUTE POKÉ")).Games))

}

func Test_SearchLocalMidTitleAndFoldedLetters(t *testing.T) {
	conn := searchLocalTestConn(t)
	s := &localSearch{}
	saveOwnedGame(conn, 1, "Super X-Moon")
	saveOwnedGame(conn, 2, "Please Don't Starve")
	saveOwnedGame(conn, 3, "Große Straße")
	saveOwnedGame(conn, 4, "King's Quest to Mars")
	saveOwnedGame(conn, 5, "Tiny Tomato Squad")

	require.Equal(t, []int64{1}, gameIDs(s.search(conn, localParams(1, "xmoon")).Games))
	require.Equal(t, []int64{1}, gameIDs(s.search(conn, localParams(1, "xmoon super")).Games))
	require.Equal(t, []int64{2}, gameIDs(s.search(conn, localParams(1, "starve dont")).Games))
	require.Equal(t, []int64{3}, gameIDs(s.search(conn, localParams(1, "strasse")).Games))

	// the s after an apostrophe doesn't start a word
	require.Equal(t, []int64{5}, gameIDs(s.search(conn, localParams(1, "squ")).Games))
	// a match that needs words run together ranks below one that doesn't
	require.Equal(t, []int64{5, 4}, gameIDs(s.search(conn, localParams(1, "tom")).Games))
}

func Test_SearchLocalSymbolTitles(t *testing.T) {
	conn := searchLocalTestConn(t)
	s := &localSearch{}
	saveOwnedGame(conn, 1, "???")
	saveOwnedGame(conn, 2, "C")
	saveOwnedGame(conn, 3, "C++")

	require.Equal(t, []int64{1}, gameIDs(s.search(conn, localParams(1, "??")).Games))
	// both fold to "c", the one typed out as it is comes first
	require.Equal(t, []int64{3, 2}, gameIDs(s.search(conn, localParams(1, "c++")).Games))
	require.Empty(t, s.search(conn, localParams(1, "-")).Games)
}

func Test_SearchLocalKeepsNonLatinMarks(t *testing.T) {
	conn := searchLocalTestConn(t)
	s := &localSearch{}
	saveOwnedGame(conn, 1, "パパ")
	saveOwnedGame(conn, 2, "ハハ")

	require.Equal(t, []int64{1}, gameIDs(s.search(conn, localParams(1, "パパ")).Games))
}

func Test_SearchLocalSubstringStaysInsideAWord(t *testing.T) {
	conn := searchLocalTestConn(t)
	s := &localSearch{}
	saveOwnedGame(conn, 1, "The Artful Escape")

	require.Equal(t, []int64{1}, gameIDs(s.search(conn, localParams(1, "scap")).Games))
	require.Empty(t, s.search(conn, localParams(1, "ear")).Games)
}

func Test_SearchLocalSeesLibraryChanges(t *testing.T) {
	conn := searchLocalTestConn(t)
	s := &localSearch{}
	saveOwnedGame(conn, 1, "Cozy Grove")
	require.Equal(t, []int64{1}, gameIDs(s.search(conn, localParams(1, "cozy")).Games))

	// cached, but not in the library until the key shows up
	models.MustSave(conn, &itchio.Game{ID: 2, Title: "Cozy Den"})
	require.Equal(t, []int64{1}, gameIDs(s.search(conn, localParams(1, "cozy")).Games))
	models.MustSave(conn, &itchio.DownloadKey{ID: 102, GameID: 2, OwnerID: 1})
	require.Equal(t, []int64{2, 1}, gameIDs(s.search(conn, localParams(1, "cozy")).Games))

	models.MustSave(conn, &itchio.Game{ID: 1, Title: "Snug Grove"})
	require.Equal(t, []int64{2}, gameIDs(s.search(conn, localParams(1, "cozy")).Games))

	models.MustDelete(conn, &itchio.DownloadKey{}, builder.Eq{"game_id": 2})
	require.Empty(t, s.search(conn, localParams(1, "cozy")).Games)
}
