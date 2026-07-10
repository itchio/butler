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
func seedSearchLocal(t *testing.T, conn *sqlite.Conn) {
	models.MustSave(conn, &itchio.Game{ID: 1, Title: "Cozy Grove"})
	models.MustSave(conn, &itchio.Game{ID: 2, Title: "Celeste"})

	models.MustSave(conn, &itchio.Bundle{ID: 10, Title: "Cozy Bundle"})
	models.MustSave(conn, &itchio.Bundle{ID: 20, Title: "Cozy Rival Bundle"})
	models.MustSave(conn, &itchio.BundleKey{ID: 1000, BundleID: 10, OwnerID: 1})
	models.MustSave(conn, &itchio.BundleKey{ID: 2000, BundleID: 20, OwnerID: 2})

	models.MustSave(conn, &itchio.Collection{ID: 30, Title: "Cozy collection", UserID: 1})
	models.MustSave(conn, &itchio.Collection{ID: 40, Title: "Cozy foreign collection", UserID: 2})
	models.MustSave(conn, &models.ProfileCollection{CollectionID: 30, ProfileID: 1})
	models.MustSave(conn, &models.ProfileCollection{CollectionID: 40, ProfileID: 2})
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

	require.ElementsMatch(t, []int64{1}, gameIDs(searchLocalGames(conn, "cozy")))
	// case-insensitive
	require.ElementsMatch(t, []int64{1}, gameIDs(searchLocalGames(conn, "COZY")))
	require.ElementsMatch(t, []int64{2}, gameIDs(searchLocalGames(conn, "celeste")))
	require.Empty(t, searchLocalGames(conn, "nothing matches this"))
}

func Test_SearchLocalGamesLimit(t *testing.T) {
	conn := searchLocalTestConn(t)
	for i := int64(1); i <= 10; i++ {
		models.MustSave(conn, &itchio.Game{ID: i, Title: fmt.Sprintf("Cozy Game %d", i)})
	}

	require.Len(t, searchLocalGames(conn, "cozy"), searchLocalGamesLimit)
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
