package fetch

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
)

func TestCondForScannedPlatformsFilter(t *testing.T) {
	conn := cavesTestConn(t)

	models.MustSave(conn, &itchio.Game{ID: 1, Title: "linux only", ScannedPlatforms: []string{"linux-amd64"}})
	models.MustSave(conn, &itchio.Game{ID: 2, Title: "handheld rom", ScannedPlatforms: []string{"rom:gba", "linux-arm64-handheld"}})
	models.MustSave(conn, &itchio.Game{ID: 3, Title: "scanned, nothing found", ScannedPlatforms: []string{}})
	models.MustSave(conn, &itchio.Game{ID: 4, Title: "never scanned"})

	ids := func(platforms ...string) []int64 {
		var games []*itchio.Game
		cond := condForScannedPlatformsFilter(platforms)
		if cond == nil {
			cond = builder.NewCond()
		}
		require.NoError(t, models.HadesContext().Select(conn, &games, cond, hades.Search{}.OrderBy("games.id")))
		var out []int64
		for _, g := range games {
			out = append(out, g.ID)
		}
		return out
	}

	require.Nil(t, condForScannedPlatformsFilter(nil))
	require.Equal(t, []int64{1, 2, 3, 4}, ids())
	require.Equal(t, []int64{1}, ids("linux-amd64"))
	require.Equal(t, []int64{2}, ids("rom:gba", "rom:snes"))
	require.Equal(t, []int64{1, 2}, ids("linux-amd64", "linux-arm64-handheld"))
	require.Empty(t, ids("windows-amd64"))

	// the platforms round-trip through the JSON column
	var game itchio.Game
	found, err := models.HadesContext().SelectOne(conn, &game, builder.Eq{"id": 2})
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []string{"rom:gba", "linux-arm64-handheld"}, game.ScannedPlatforms)
}

func TestScannedPlatformsFilterValidation(t *testing.T) {
	require.NoError(t, butlerd.ProfileOwnedKeysFilters{ScannedPlatforms: []string{"linux-arm64"}}.Validate())
	require.Error(t, butlerd.ProfileOwnedKeysFilters{ScannedPlatforms: []string{""}}.Validate())
}
