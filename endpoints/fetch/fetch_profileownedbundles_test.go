package fetch

import (
	"testing"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/stretchr/testify/require"
)

func TestExpireChangedBundleGames(t *testing.T) {
	conn, err := sqlite.OpenConn("file::memory:?mode=memory", 0)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, models.HadesContext().AutoMigrate(conn))

	previous := map[int64]*itchio.Bundle{
		1: {ID: 1, Version: 5, GamesCount: 10},
		2: {ID: 2, Version: 5, GamesCount: 10},
		3: {ID: 3, Version: 5, GamesCount: 10},
	}

	unchangedTarget := models.FetchTargetForBundleGames(1)
	versionBumpedTarget := models.FetchTargetForBundleGames(2)
	countChangedTarget := models.FetchTargetForBundleGames(3)
	newTarget := models.FetchTargetForBundleGames(4)
	nilBundleTarget := models.FetchTargetForBundleGames(5)
	unchangedTarget.MustMarkFresh(conn)
	versionBumpedTarget.MustMarkFresh(conn)
	countChangedTarget.MustMarkFresh(conn)
	newTarget.MustMarkFresh(conn)
	nilBundleTarget.MustMarkFresh(conn)

	expireChangedBundleGames(conn, previous, []*itchio.BundleKey{
		{BundleID: 1, Bundle: &itchio.Bundle{ID: 1, Version: 5, GamesCount: 10}},
		{BundleID: 2, Bundle: &itchio.Bundle{ID: 2, Version: 6, GamesCount: 10}},
		{BundleID: 3, Bundle: &itchio.Bundle{ID: 3, Version: 5, GamesCount: 11}},
		{BundleID: 4, Bundle: &itchio.Bundle{ID: 4, Version: 1, GamesCount: 2}},
		{BundleID: 5, Bundle: nil},
	})

	require.False(t, unchangedTarget.MustIsStale(conn))
	require.True(t, versionBumpedTarget.MustIsStale(conn))
	require.True(t, countChangedTarget.MustIsStale(conn))
	require.True(t, newTarget.MustIsStale(conn))
	require.False(t, nilBundleTarget.MustIsStale(conn))
}
