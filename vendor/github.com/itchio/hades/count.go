package hades

import (
	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
)

func (c *Context) Count(conn *sqlite.Conn, model interface{}, cond builder.Cond) (int64, error) {
	ms := c.NewScope(model).GetModelStruct()

	query, args, err := builder.Select("count(*)").From(ms.TableName).Where(cond).ToSQL()
	if err != nil {
		return 0, err
	}

	var result int64

	err = c.ExecRaw(conn, query, func(stmt *sqlite.Stmt) error {
		result = stmt.ColumnInt64(0)
		return nil
	}, args...)

	if err != nil {
		return 0, err
	}
	return result, nil
}
