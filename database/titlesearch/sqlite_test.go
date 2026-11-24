package titlesearch

import (
	"testing"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
)

func Test_Cond(t *testing.T) {
	conn, err := sqlite.OpenConn("file::memory:?mode=memory", 0)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, Register(conn))
	require.NoError(t, sqlitex.ExecScript(conn, `
		create table games (id integer primary key, title text);
		insert into games values (1, 'X-Moon'), (2, 'Pokémon Tribute'), (3, null), (4, 'A Night in the Woods!');
	`))

	matching := func(query string) []int64 {
		sql, args, err := builder.Select("id").From("games").Where(Cond("games.title", query)).OrderBy("id").ToSQL()
		require.NoError(t, err)
		var ids []int64
		require.NoError(t, sqlitex.Exec(conn, sql, func(stmt *sqlite.Stmt) error {
			ids = append(ids, stmt.ColumnInt64(0))
			return nil
		}, args...))
		return ids
	}

	require.Equal(t, []int64{1}, matching("x moon"))
	require.Equal(t, []int64{2}, matching("pokemon"))
	// the query changes between statements on the same connection
	require.Equal(t, []int64{1}, matching("xmoon"))
	require.Empty(t, matching("celeste"))
	require.Equal(t, []int64{1, 2, 3, 4}, matching(" "))

	// whatever contains the query as typed still matches
	require.Equal(t, []int64{4}, matching("ight in"))
	require.Equal(t, []int64{4}, matching("!"))
	require.Equal(t, []int64{1}, matching("-"))
	// "and" is only skipped next to other words
	require.Empty(t, matching("and and"))
}
