package search

import (
	"fmt"
	"testing"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/stretchr/testify/require"
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
	seedSearchLocal(t, conn)

	// case-insensitive
	require.ElementsMatch(t, []int64{1}, gameIDs(searchLocalGames(conn, 1, "GROVE")))
	require.Empty(t, searchLocalGames(conn, 1, "nothing matches this"))
}

func Test_SearchLocalGamesScopedToProfile(t *testing.T) {
	conn := searchLocalTestConn(t)
	seedSearchLocal(t, conn)

	// owned via download key
	require.ElementsMatch(t, []int64{1}, gameIDs(searchLocalGames(conn, 1, "grove")))
	require.Empty(t, searchLocalGames(conn, 2, "grove"))
	require.Empty(t, searchLocalGames(conn, 1, "celeste"))

	// in one of the profile's collections
	require.ElementsMatch(t, []int64{3}, gameIDs(searchLocalGames(conn, 1, "shelf")))
	require.Empty(t, searchLocalGames(conn, 2, "shelf"))

	// part of an owned bundle
	require.ElementsMatch(t, []int64{4}, gameIDs(searchLocalGames(conn, 1, "pack")))
	require.Empty(t, searchLocalGames(conn, 2, "pack"))

	// on the profile's dashboard
	require.ElementsMatch(t, []int64{6}, gameIDs(searchLocalGames(conn, 1, "workshop")))
	require.Empty(t, searchLocalGames(conn, 2, "workshop"))

	// installs aren't profile-scoped, every profile sees them
	require.ElementsMatch(t, []int64{5}, gameIDs(searchLocalGames(conn, 1, "cabin")))
	require.ElementsMatch(t, []int64{5}, gameIDs(searchLocalGames(conn, 2, "cabin")))

	// cached from browsing only: no profile sees it
	require.Empty(t, searchLocalGames(conn, 1, "drifter"))
	require.Empty(t, searchLocalGames(conn, 2, "drifter"))
}

// saveOwnedGame saves a game along with a download key so it's in
// profile 1's library
func saveOwnedGame(conn *sqlite.Conn, id int64, title string) {
	models.MustSave(conn, &itchio.Game{ID: id, Title: title})
	models.MustSave(conn, &itchio.DownloadKey{ID: 100 + id, GameID: id, OwnerID: 1})
}

func Test_SearchLocalGamesRelevanceOrder(t *testing.T) {
	conn := searchLocalTestConn(t)
	saveOwnedGame(conn, 1, "A Very Cozy Adventure") // substring, longest
	saveOwnedGame(conn, 2, "So Cozy")               // substring
	saveOwnedGame(conn, 3, "Cozy Grove")            // prefix
	saveOwnedGame(conn, 4, "cozy")                  // exact (case-insensitive)
	saveOwnedGame(conn, 5, "Cozy Den")              // prefix, shorter than Cozy Grove

	// exact first, then prefixes by title length, then substrings by title
	// length; the longest substring match falls off the limit of 4
	require.Equal(t, []int64{4, 5, 3, 2}, gameIDs(searchLocalGames(conn, 1, "Cozy")))
}

func Test_SearchLocalGamesOrderIsDeterministic(t *testing.T) {
	conn := searchLocalTestConn(t)
	// same tier, same title length: id breaks the tie
	saveOwnedGame(conn, 9, "Cozy Web")
	saveOwnedGame(conn, 4, "Cozy Den")

	require.Equal(t, []int64{4, 9}, gameIDs(searchLocalGames(conn, 1, "cozy")))
}

func Test_SearchLocalGamesLimit(t *testing.T) {
	conn := searchLocalTestConn(t)
	for i := int64(1); i <= 10; i++ {
		saveOwnedGame(conn, i, fmt.Sprintf("Cozy Game %d", i))
	}

	require.Len(t, searchLocalGames(conn, 1, "cozy"), searchLocalGamesLimit)
}

func Test_SearchLocalBundlesScopedToProfile(t *testing.T) {
	conn := searchLocalTestConn(t)
	seedSearchLocal(t, conn)

	require.ElementsMatch(t, []int64{10}, bundleIDs(searchLocalBundles(conn, 1, "cozy")))
	require.ElementsMatch(t, []int64{20}, bundleIDs(searchLocalBundles(conn, 2, "cozy")))
	// profile without any bundle keys
	require.Empty(t, searchLocalBundles(conn, 3, "cozy"))
	// owned, but query doesn't match
	require.Empty(t, searchLocalBundles(conn, 1, "rival"))
}

func Test_SearchLocalCollectionsScopedToProfile(t *testing.T) {
	conn := searchLocalTestConn(t)
	seedSearchLocal(t, conn)

	require.ElementsMatch(t, []int64{30}, collectionIDs(searchLocalCollections(conn, 1, "cozy")))
	require.ElementsMatch(t, []int64{40}, collectionIDs(searchLocalCollections(conn, 2, "cozy")))
	require.Empty(t, searchLocalCollections(conn, 3, "cozy"))
	require.Empty(t, searchLocalCollections(conn, 1, "foreign"))
}
