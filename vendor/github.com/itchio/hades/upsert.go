package hades

import (
	"fmt"
	"reflect"
	"strings"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
)

// TODO: cache me
func (scope *Scope) ToSets() []string {
	var sets []string

	var processField func(sf *StructField)
	processField = func(sf *StructField) {
		if sf.IsSquashed {
			for _, nsf := range sf.SquashedFields {
				processField(nsf)
			}
		}

		if !sf.IsNormal {
			return
		}

		if sf.IsPrimaryKey {
			return
		}

		name := EscapeIdentifier(sf.DBName)
		sets = append(sets, fmt.Sprintf("%s=excluded.%s", name, name))
	}

	for _, sf := range scope.GetStructFields() {
		processField(sf)
	}

	return sets
}

func (c *Context) Upsert(conn *sqlite.Conn, scope *Scope, rec reflect.Value) error {
	eq := scope.ToEq(rec)

	b := builder.Insert(eq).Into(scope.TableName())

	sql, args, err := b.ToSQL()
	if err != nil {
		return err
	}

	sets := scope.ToSets()

	if len(sets) == 0 {
		sql = fmt.Sprintf("%s ON CONFLICT DO NOTHING",
			sql,
		)
	} else {
		var pfNames []string
		for _, pf := range scope.GetModelStruct().PrimaryFields {
			pfNames = append(pfNames, pf.DBName)
		}

		sql = fmt.Sprintf("%s ON CONFLICT(%s) DO UPDATE SET %s",
			sql,
			strings.Join(pfNames, ","),
			strings.Join(sets, ","),
		)
	}
	return c.ExecRaw(conn, sql, nil, args...)
}
