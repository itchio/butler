package models

import (
	"testing"
	"time"

	"crawshaw.io/sqlite"
	itchio "github.com/itchio/go-itchio"
	"github.com/stretchr/testify/require"
)

func interactionTestConn(t *testing.T) *sqlite.Conn {
	conn, err := sqlite.OpenConn("file::memory:?mode=memory", 0)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, HadesContext().AutoMigrate(conn))
	return conn
}

func summary(seconds int64, lastRunAt *time.Time) *itchio.UserGameInteractionsSummary {
	return &itchio.UserGameInteractionsSummary{SecondsRun: seconds, LastRunAt: lastRunAt}
}

func TestUserGameInteractionUpsertIsPerUser(t *testing.T) {
	conn := interactionTestConn(t)

	require.NoError(t, SaveUserGameInteractionSummary(conn, 1, 100, summary(60, nil)))
	require.NoError(t, SaveUserGameInteractionSummary(conn, 2, 100, summary(999, nil)))
	require.NoError(t, SaveUserGameInteractionSummary(conn, 1, 100, summary(120, nil)))

	u1 := UserGameInteractionByUserAndGame(conn, 1, 100)
	require.NotNil(t, u1)
	require.EqualValues(t, 120, u1.SecondsRun)
	require.NotNil(t, u1.SyncedAt)

	u2 := UserGameInteractionByUserAndGame(conn, 2, 100)
	require.NotNil(t, u2)
	require.EqualValues(t, 999, u2.SecondsRun)
}

func TestMissingInteractionIsNil(t *testing.T) {
	conn := interactionTestConn(t)
	require.Nil(t, UserGameInteractionByUserAndGame(conn, 1, 100))
}

func TestSummaryMirrorsToAllCavesOfGame(t *testing.T) {
	conn := interactionTestConn(t)

	MustSave(conn, &Cave{ID: "cave-a", GameID: 100})
	MustSave(conn, &Cave{ID: "cave-b", GameID: 100})
	MustSave(conn, &Cave{ID: "cave-other", GameID: 200})

	lastRun := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	require.NoError(t, SaveUserGameInteractionSummary(conn, 1, 100, summary(300, &lastRun)))

	for _, id := range []string{"cave-a", "cave-b"} {
		c := CaveByID(conn, id)
		require.NotNil(t, c)
		require.EqualValues(t, 300, c.SecondsRun, "cave %s", id)
		require.NotNil(t, c.LastTouchedAt)
		require.True(t, lastRun.Equal(*c.LastTouchedAt), "cave %s", id)
	}
	other := CaveByID(conn, "cave-other")
	require.NotNil(t, other)
	require.EqualValues(t, 0, other.SecondsRun)
}

func TestSummaryNilLastRunLeavesCaveTimestamp(t *testing.T) {
	conn := interactionTestConn(t)

	existing := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	MustSave(conn, &Cave{ID: "cave-a", GameID: 100, LastTouchedAt: &existing})

	require.NoError(t, SaveUserGameInteractionSummary(conn, 1, 100, summary(42, nil)))

	c := CaveByID(conn, "cave-a")
	require.NotNil(t, c)
	require.EqualValues(t, 42, c.SecondsRun)
	require.NotNil(t, c.LastTouchedAt)
	require.True(t, existing.Equal(*c.LastTouchedAt))
}

func TestSaveSummaryRejectsZeroIdentity(t *testing.T) {
	conn := interactionTestConn(t)
	require.Error(t, SaveUserGameInteractionSummary(conn, 0, 100, summary(1, nil)))
	require.Error(t, SaveUserGameInteractionSummary(conn, 1, 0, summary(1, nil)))
	require.NoError(t, SaveUserGameInteractionSummary(conn, 1, 100, nil))
	require.Nil(t, UserGameInteractionByUserAndGame(conn, 1, 100))
}

func TestRecordLocalPlayTime(t *testing.T) {
	conn := interactionTestConn(t)

	MustSave(conn, &Cave{ID: "cave-a", GameID: 100})

	end1 := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	c := CaveByID(conn, "cave-a")
	c.RecordLocalPlayTime(90*time.Second, end1)
	c.Save(conn)

	end2 := end1.Add(time.Hour)
	c = CaveByID(conn, "cave-a")
	c.RecordLocalPlayTime(30*time.Second, end2)
	c.Save(conn)

	c = CaveByID(conn, "cave-a")
	require.EqualValues(t, 120, c.LocalSecondsRun)
	require.NotNil(t, c.LocalLastRunAt)
	require.True(t, end2.Equal(*c.LocalLastRunAt))
	require.EqualValues(t, 0, c.SecondsRun)
	require.Nil(t, c.LastTouchedAt)
}

func TestFreshDatabaseCreatesInteractionTable(t *testing.T) {
	conn := interactionTestConn(t)

	var cols []string
	MustExecRaw(conn,
		"select name from pragma_table_info('user_game_interactions')",
		func(stmt *sqlite.Stmt) error {
			cols = append(cols, stmt.ColumnText(0))
			return nil
		},
	)
	require.ElementsMatch(t, []string{"user_id", "game_id", "seconds_run", "last_run_at", "synced_at"}, cols)
}
