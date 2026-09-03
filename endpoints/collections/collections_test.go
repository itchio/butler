package collections

import (
	"testing"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
)

func testConn(t *testing.T) *sqlite.Conn {
	conn, err := sqlite.OpenConn("file::memory:?mode=memory", 0)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, models.HadesContext().AutoMigrate(conn))
	return conn
}

func TestSaveCollectionKeepsGamesAndMembershipOut(t *testing.T) {
	conn := testConn(t)
	yes := true

	models.MustSave(conn, &itchio.Collection{ID: 1, Title: "old"})
	models.MustSave(conn, &itchio.CollectionGame{CollectionID: 1, GameID: 10, Position: 0})

	saveCollection(conn, &itchio.Collection{
		ID:      1,
		Title:   "new",
		Private: true,
		Layout:  itchio.CollectionLayoutList,
		HasGame: &yes,
		CollectionGames: []*itchio.CollectionGame{
			{CollectionID: 1, GameID: 99},
		},
	})

	c := models.CollectionByID(conn, 1)
	require.NotNil(t, c)
	require.Equal(t, "new", c.Title)
	require.True(t, c.Private)
	require.Equal(t, itchio.CollectionLayoutList, c.Layout)
	require.Nil(t, c.HasGame)

	// existing membership rows are untouched, the payload's are ignored
	require.EqualValues(t, 1, models.MustCount(conn, &itchio.CollectionGame{}, builder.Eq{"collection_id": 1}))
	require.EqualValues(t, 1, models.MustCount(conn, &itchio.CollectionGame{}, builder.Eq{"collection_id": 1, "game_id": 10}))
}

func TestSaveCollectionGameWithAndWithoutGame(t *testing.T) {
	conn := testConn(t)

	saveCollectionGame(conn, &itchio.CollectionGame{
		CollectionID: 1,
		GameID:       10,
		Position:     3,
		Blurb:        "<p>hi</p>",
		Game:         &itchio.Game{ID: 10, Title: "Ten"},
	})
	saveCollectionGame(conn, &itchio.CollectionGame{
		CollectionID: 1,
		GameID:       11,
		Position:     4,
	})

	var items []*itchio.CollectionGame
	models.MustSelect(conn, &items, builder.Eq{"collection_id": 1}, hades.Search{}.OrderBy("position ASC"))
	models.MustPreload(conn, items, hades.Assoc("Game"))
	require.Len(t, items, 2)
	require.Equal(t, "<p>hi</p>", items[0].Blurb)
	require.NotNil(t, items[0].Game)
	require.Equal(t, "Ten", items[0].Game.Title)
	require.Nil(t, items[1].Game)
}

func TestForgetCollection(t *testing.T) {
	conn := testConn(t)

	models.MustSave(conn, &itchio.Collection{ID: 1, Title: "gone"})
	models.MustSave(conn, &itchio.Collection{ID: 2, Title: "stays"})
	models.MustSave(conn, &itchio.CollectionGame{CollectionID: 1, GameID: 10})
	models.MustSave(conn, &itchio.CollectionGame{CollectionID: 2, GameID: 10})
	models.MustSave(conn, &models.ProfileCollection{ProfileID: 7, CollectionID: 1})
	models.MustSave(conn, &models.ProfileCollection{ProfileID: 7, CollectionID: 2})
	models.FetchTargetForCollection(1).MustMarkFresh(conn)
	models.FetchTargetForCollectionGames(1).MustMarkFresh(conn)

	forgetCollection(conn, 1)

	require.Nil(t, models.CollectionByID(conn, 1))
	require.NotNil(t, models.CollectionByID(conn, 2))
	require.EqualValues(t, 0, models.MustCount(conn, &itchio.CollectionGame{}, builder.Eq{"collection_id": 1}))
	require.EqualValues(t, 1, models.MustCount(conn, &itchio.CollectionGame{}, builder.Eq{"collection_id": 2}))
	require.EqualValues(t, 0, models.MustCount(conn, &models.ProfileCollection{}, builder.Eq{"collection_id": 1}))
	require.EqualValues(t, 1, models.MustCount(conn, &models.ProfileCollection{}, builder.Eq{"collection_id": 2}))
	require.True(t, models.FetchTargetForCollection(1).MustIsStale(conn))
	require.True(t, models.FetchTargetForCollectionGames(1).MustIsStale(conn))
}

func TestExpireCollectionContents(t *testing.T) {
	conn := testConn(t)

	// profile 8 also lists collection 1 (as an admin), profile 9 does not
	models.MustSave(conn, &models.ProfileCollection{ProfileID: 8, CollectionID: 1})
	models.MustSave(conn, &models.ProfileCollection{ProfileID: 9, CollectionID: 2})
	models.FetchTargetForProfileCollections(8).MustMarkFresh(conn)
	models.FetchTargetForProfileCollections(9).MustMarkFresh(conn)

	models.FetchTargetForCollection(1).MustMarkFresh(conn)
	models.FetchTargetForCollectionGames(1).MustMarkFresh(conn)
	models.FetchTargetForProfileCollections(7).MustMarkFresh(conn)
	models.FetchTargetForCollectionGames(2).MustMarkFresh(conn)

	expireCollectionContents(conn, 7, 1)

	require.True(t, models.FetchTargetForCollection(1).MustIsStale(conn))
	require.True(t, models.FetchTargetForCollectionGames(1).MustIsStale(conn))
	require.True(t, models.FetchTargetForProfileCollections(7).MustIsStale(conn))
	require.False(t, models.FetchTargetForCollectionGames(2).MustIsStale(conn))
	require.True(t, models.FetchTargetForProfileCollections(8).MustIsStale(conn))
	require.False(t, models.FetchTargetForProfileCollections(9).MustIsStale(conn))
}
