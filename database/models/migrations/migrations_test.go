package migrations

import (
	"testing"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/headway/state"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func testConn(t *testing.T) *sqlite.Conn {
	conn, err := sqlite.OpenConn("file::memory:?mode=memory", 0)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, models.HadesContext().AutoMigrate(conn))
	return conn
}

// Regression test: the loop in Do used to return unconditionally after the
// first pending migration, so any migration after it silently never ran.
func Test_RunAppliesAllPendingMigrations(t *testing.T) {
	conn := testConn(t)

	var ran []int64
	table := map[int64]Migration{
		100: func(consumer *state.Consumer, conn *sqlite.Conn) error {
			ran = append(ran, 100)
			return nil
		},
		200: func(consumer *state.Consumer, conn *sqlite.Conn) error {
			ran = append(ran, 200)
			return nil
		},
	}

	err := run(&state.Consumer{}, conn, table)
	require.NoError(t, err)
	require.Equal(t, []int64{100, 200}, ran)
	require.EqualValues(t, 200, models.GetSchemaVersion(conn))
}

func Test_RunStopsAtFailingMigration(t *testing.T) {
	conn := testConn(t)

	var ran []int64
	table := map[int64]Migration{
		100: func(consumer *state.Consumer, conn *sqlite.Conn) error {
			ran = append(ran, 100)
			return nil
		},
		200: func(consumer *state.Consumer, conn *sqlite.Conn) error {
			return errors.New("boom")
		},
		300: func(consumer *state.Consumer, conn *sqlite.Conn) error {
			ran = append(ran, 300)
			return nil
		},
	}

	err := run(&state.Consumer{}, conn, table)
	require.Error(t, err)
	require.Equal(t, []int64{100}, ran)
	// schema version reflects the last successful migration only
	require.EqualValues(t, 100, models.GetSchemaVersion(conn))
}

func Test_RunSkipsAlreadyApplied(t *testing.T) {
	conn := testConn(t)
	models.SetSchemaVersion(conn, 100)

	var ran []int64
	table := map[int64]Migration{
		100: func(consumer *state.Consumer, conn *sqlite.Conn) error {
			ran = append(ran, 100)
			return nil
		},
		200: func(consumer *state.Consumer, conn *sqlite.Conn) error {
			ran = append(ran, 200)
			return nil
		},
	}

	err := run(&state.Consumer{}, conn, table)
	require.NoError(t, err)
	require.Equal(t, []int64{200}, ran)
	require.EqualValues(t, 200, models.GetSchemaVersion(conn))
}
