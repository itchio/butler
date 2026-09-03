package models

import (
	"testing"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/stretchr/testify/require"
)

func Test_CollectionsTableHasNoHasGameColumn(t *testing.T) {
	conn, err := sqlite.OpenConn("file::memory:?mode=memory", 0)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, HadesContext().AutoMigrate(conn))

	var cols []string
	require.NoError(t, sqlitex.Exec(conn, "PRAGMA table_info(collections)", func(stmt *sqlite.Stmt) error {
		cols = append(cols, stmt.ColumnText(1))
		return nil
	}))
	require.Contains(t, cols, "layout")
	require.NotContains(t, cols, "has_game")
}
