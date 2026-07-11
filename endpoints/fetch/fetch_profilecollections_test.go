package fetch

import (
	"testing"
	"time"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/stretchr/testify/require"
)

func TestExpireChangedCollectionGames(t *testing.T) {
	conn, err := sqlite.OpenConn("file::memory:?mode=memory", 0)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, models.HadesContext().AutoMigrate(conn))

	updatedAt := time.Now().UTC().Add(-time.Hour)
	previous := map[int64]*itchio.Collection{
		1: {ID: 1, UpdatedAt: &updatedAt, GamesCount: 3},
		2: {ID: 2, UpdatedAt: &updatedAt, GamesCount: 3},
	}

	unchangedTarget := models.FetchTargetForCollectionGames(1)
	changedTarget := models.FetchTargetForCollectionGames(2)
	newTarget := models.FetchTargetForCollectionGames(3)
	unchangedTarget.MustMarkFresh(conn)
	changedTarget.MustMarkFresh(conn)
	newTarget.MustMarkFresh(conn)

	changedAt := updatedAt.Add(time.Minute)
	expireChangedCollectionGames(conn, previous, []*itchio.Collection{
		{ID: 1, UpdatedAt: &updatedAt, GamesCount: 3},
		{ID: 2, UpdatedAt: &changedAt, GamesCount: 3},
		{ID: 3, UpdatedAt: &updatedAt, GamesCount: 1},
	})

	require.False(t, unchangedTarget.MustIsStale(conn))
	require.True(t, changedTarget.MustIsStale(conn))
	require.True(t, newTarget.MustIsStale(conn))
}

func TestCollectionChangedWhenGamesCountChanges(t *testing.T) {
	updatedAt := time.Now().UTC()
	previous := &itchio.Collection{UpdatedAt: &updatedAt, GamesCount: 3}
	current := &itchio.Collection{UpdatedAt: &updatedAt, GamesCount: 4}
	require.True(t, collectionChanged(previous, current))
}
