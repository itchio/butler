package hades

import (
	"reflect"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/pkg/errors"
)

func (c *Context) Delete(conn *sqlite.Conn, model interface{}, cond builder.Cond) error {
	modelType := reflect.TypeOf(model)

	scope := c.ScopeMap.ByType(modelType)
	if scope == nil {
		return errors.Errorf("%v is not a model known to this hades context", modelType)
	}

	if cond == builder.NewCond() {
		return errors.Errorf("refusing to blindly delete all %v without an explicit builder.Expr(\"1\") clause", modelType)
	}

	b := builder.Delete(cond).From(scope.TableName())
	return c.Exec(conn, b, nil)
}
