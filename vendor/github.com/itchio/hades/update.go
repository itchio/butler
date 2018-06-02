package hades

import (
	"reflect"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/pkg/errors"
)

type WhereCond interface {
	Cond() builder.Cond
}

type whereImpl struct {
	cond builder.Cond
}

func (wt whereImpl) Cond() builder.Cond {
	return wt.cond
}

func Where(cond builder.Cond) WhereCond {
	return whereImpl{cond: cond}
}

func (c *Context) Update(conn *sqlite.Conn, model interface{}, where WhereCond, updates ...builder.Eq) error {
	modelType := reflect.TypeOf(model)
	scope := c.ScopeMap.ByType(modelType)
	if scope == nil {
		return errors.Errorf("%v is not a know model type", modelType)
	}

	tableName := scope.TableName()
	b := builder.Update(updates...).Where(where.Cond()).Into(tableName)
	return c.Exec(conn, b, nil)
}
