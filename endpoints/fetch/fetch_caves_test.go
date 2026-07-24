package fetch

import (
	"testing"
	"time"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func cavesTestConn(t *testing.T) *sqlite.Conn {
	conn, err := sqlite.OpenConn("file::memory:?mode=memory", 0)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, models.HadesContext().AutoMigrate(conn))
	return conn
}

func day(n int) *time.Time {
	d := time.Date(2026, 7, n, 0, 0, 0, 0, time.UTC)
	return &d
}

// Three installed games. Cave columns say game 100 is most played (legacy
// projection last written by some other user); user 1's interactions say
// game 300 is most played, game 200 never played, and nothing synced for 100.
func seedCaves(t *testing.T, conn *sqlite.Conn) {
	models.MustSave(conn, &itchio.Game{ID: 100, Title: "Alpha"})
	models.MustSave(conn, &itchio.Game{ID: 200, Title: "Beta"})
	models.MustSave(conn, &itchio.Game{ID: 300, Title: "Gamma"})

	models.MustSave(conn, &models.Cave{ID: "cave-100", GameID: 100, CustomInstallFolder: "/tmp/c100", SecondsRun: 9000, LastTouchedAt: day(15), InstalledAt: day(1)})
	models.MustSave(conn, &models.Cave{ID: "cave-200", GameID: 200, CustomInstallFolder: "/tmp/c200", SecondsRun: 500, LastTouchedAt: day(10), InstalledAt: day(2)})
	models.MustSave(conn, &models.Cave{ID: "cave-300", GameID: 300, CustomInstallFolder: "/tmp/c300", SecondsRun: 100, LastTouchedAt: day(5), InstalledAt: day(3)})

	require.NoError(t, models.SaveUserGameInteractionSummary(conn, 1, 300,
		&itchio.UserGameInteractionsSummary{SecondsRun: 7000, LastRunAt: day(20)}))
	require.NoError(t, models.SaveUserGameInteractionSummary(conn, 1, 200,
		&itchio.UserGameInteractionsSummary{SecondsRun: 0, LastRunAt: nil}))
}

func caveOrder(res *butlerd.FetchCavesResult) []string {
	var ids []string
	for _, c := range res.Items {
		ids = append(ids, c.ID)
	}
	return ids
}

func TestFetchCavesLegacySortUnchanged(t *testing.T) {
	conn := cavesTestConn(t)
	seedCaves(t, conn)

	res := &butlerd.FetchCavesResult{}
	fetchCavesWithConn(conn, butlerd.FetchCavesParams{SortBy: "playTime"}, res)
	require.Equal(t, []string{"cave-100", "cave-300", "cave-200"}, caveOrder(res))
	for _, c := range res.Items {
		require.Nil(t, c.Interaction)
	}
}

func TestFetchCavesProfileSortUsesInteractions(t *testing.T) {
	conn := cavesTestConn(t)
	seedCaves(t, conn)

	res := &butlerd.FetchCavesResult{}
	fetchCavesWithConn(conn, butlerd.FetchCavesParams{SortBy: "playTime", ProfileID: 1}, res)
	require.Equal(t, "cave-300", caveOrder(res)[0])

	byID := map[string]*butlerd.Cave{}
	for _, c := range res.Items {
		byID[c.ID] = c
	}
	require.NotNil(t, byID["cave-300"].Interaction)
	require.EqualValues(t, 7000, byID["cave-300"].Interaction.SecondsRun)
	require.Nil(t, byID["cave-100"].Interaction)
	require.NotNil(t, byID["cave-200"].Interaction)
	require.EqualValues(t, 0, byID["cave-200"].Interaction.SecondsRun)
}

func TestFetchCavesProfileLastTouchedSort(t *testing.T) {
	conn := cavesTestConn(t)
	seedCaves(t, conn)

	res := &butlerd.FetchCavesResult{}
	fetchCavesWithConn(conn, butlerd.FetchCavesParams{SortBy: "lastTouched", ProfileID: 1}, res)
	require.Equal(t, []string{"cave-300", "cave-200", "cave-100"}, caveOrder(res))
}

func TestFetchCavesProfileNeverPlayed(t *testing.T) {
	conn := cavesTestConn(t)
	seedCaves(t, conn)

	res := &butlerd.FetchCavesResult{}
	fetchCavesWithConn(conn, butlerd.FetchCavesParams{
		ProfileID: 1,
		Filters:   butlerd.CavesFilters{NeverPlayed: true},
	}, res)
	require.ElementsMatch(t, []string{"cave-100", "cave-200"}, caveOrder(res))
}

func TestRequireProfile(t *testing.T) {
	conn := cavesTestConn(t)
	models.MustSave(conn, &models.Profile{ID: 1, APIKey: "key1"})

	profile, err := requireProfile(conn, 1)
	require.NoError(t, err)
	require.Equal(t, "key1", profile.APIKey)

	_, err = requireProfile(conn, 999)
	require.Equal(t, butlerd.CodeNoSuchProfile, errors.Cause(err))
}

func TestFetchCavesLegacyNeverPlayed(t *testing.T) {
	conn := cavesTestConn(t)
	seedCaves(t, conn)

	res := &butlerd.FetchCavesResult{}
	fetchCavesWithConn(conn, butlerd.FetchCavesParams{
		Filters: butlerd.CavesFilters{NeverPlayed: true},
	}, res)
	require.Empty(t, caveOrder(res))
}

func TestNeverPlayedCountsLocalPlay(t *testing.T) {
	conn := cavesTestConn(t)
	models.MustSave(conn, &itchio.Game{ID: 100, Title: "Alpha"})
	models.MustSave(conn, &models.Cave{ID: "cave-local", GameID: 100, CustomInstallFolder: "/tmp/cl", LocalSecondsRun: 60})
	models.MustSave(conn, &itchio.Game{ID: 200, Title: "Beta"})
	models.MustSave(conn, &models.Cave{ID: "cave-untouched", GameID: 200, CustomInstallFolder: "/tmp/cu"})

	res := &butlerd.FetchCavesResult{}
	fetchCavesWithConn(conn, butlerd.FetchCavesParams{
		Filters: butlerd.CavesFilters{NeverPlayed: true},
	}, res)
	require.Equal(t, []string{"cave-untouched"}, caveOrder(res))

	res = &butlerd.FetchCavesResult{}
	fetchCavesWithConn(conn, butlerd.FetchCavesParams{
		ProfileID: 1,
		Filters:   butlerd.CavesFilters{NeverPlayed: true},
	}, res)
	require.ElementsMatch(t, []string{"cave-local", "cave-untouched"}, caveOrder(res))
}

func TestNeverPlayedAfterCaveSchemaUpgrade(t *testing.T) {
	conn, err := sqlite.OpenConn("file::memory:?mode=memory", 0)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	models.MustExecRaw(conn, `
		create table caves (
			id text not null,
			game_id integer,
			last_touched_at datetime,
			seconds_run integer,
			custom_install_folder text,
			primary key (id)
		)
	`, nil)
	models.MustExecRaw(conn, `
		insert into caves (id, game_id, seconds_run, custom_install_folder)
		values ('legacy-cave', 100, 0, '/tmp/legacy-cave')
	`, nil)

	require.NoError(t, models.HadesContext().AutoMigrate(conn))
	models.MustSave(conn, &itchio.Game{ID: 100, Title: "Alpha"})

	res := &butlerd.FetchCavesResult{}
	fetchCavesWithConn(conn, butlerd.FetchCavesParams{
		Filters: butlerd.CavesFilters{NeverPlayed: true},
	}, res)
	require.Equal(t, []string{"legacy-cave"}, caveOrder(res))
}
