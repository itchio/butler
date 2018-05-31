package hades

import (
	"fmt"

	"crawshaw.io/sqlite"
)

type PragmaTableInfoRow struct {
	ColumnID   int64
	Name       string
	Type       string
	NotNull    bool
	PrimaryKey bool
}

func (c *Context) PragmaTableInfo(conn *sqlite.Conn, tableName string) ([]PragmaTableInfoRow, error) {
	var res []PragmaTableInfoRow

	query := fmt.Sprintf("PRAGMA table_info(%s)", EscapeIdentifier(tableName))
	err := c.ExecRaw(conn, query, func(stmt *sqlite.Stmt) error {
		// results of pragma
		// 0 cid, 1 name, 2 type, 3 notnull, 4 dflt_value, 5 pk
		res = append(res, PragmaTableInfoRow{
			ColumnID:   stmt.ColumnInt64(0),
			Name:       stmt.ColumnText(1),
			Type:       stmt.ColumnText(2),
			NotNull:    stmt.ColumnInt(3) == 1,
			PrimaryKey: stmt.ColumnInt(5) == 1,
		})
		return nil
	})

	return res, err
}
