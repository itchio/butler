package hades

import (
	"fmt"
	"reflect"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/pkg/errors"
)

func (c *Context) Select(conn *sqlite.Conn, result interface{}, cond builder.Cond, search *SearchParams) error {
	resultVal := reflect.ValueOf(result)
	originalType := resultVal.Type()

	if resultVal.Type().Kind() != reflect.Ptr {
		return errors.Errorf("Select expects results to be a *[]Model, but it got a %v", originalType)
	}
	resultVal = resultVal.Elem()

	if resultVal.Type().Kind() != reflect.Slice {
		return errors.Errorf("Select expects results to be a *[]Model, but it got a %v", originalType)
	}

	modelType := resultVal.Type().Elem()
	scope := c.ScopeMap.ByType(modelType)
	if scope == nil {
		return errors.Errorf("%v is not a model known to this hades context", modelType)
	}

	ms := scope.GetModelStruct()
	columns, fields := c.selectFields(ms)

	query, args, err := builder.Select(columns...).From(ms.TableName).Where(cond).ToSQL()
	if err != nil {
		return err
	}
	query = search.Apply(query)

	return c.ExecRaw(conn, query, func(stmt *sqlite.Stmt) error {
		el := reflect.New(ms.ModelType)
		err := c.Scan(stmt, fields, el.Elem())
		if err != nil {
			return err
		}
		resultVal.Set(reflect.Append(resultVal, el))
		return nil
	}, args...)
}

//

func (c *Context) SelectOne(conn *sqlite.Conn, result interface{}, cond builder.Cond) error {
	resultVal := reflect.ValueOf(result)
	originalType := resultVal.Type()
	modelType := originalType

	if resultVal.Type().Kind() != reflect.Ptr {
		return errors.Errorf("SelectOne expects results to be a *Model, but it got a %v", originalType)
	}
	resultVal = resultVal.Elem()

	scope := c.ScopeMap.ByType(modelType)
	if scope == nil {
		return errors.Errorf("%v is not a model known to this hades context", modelType)
	}

	ms := scope.GetModelStruct()
	columns, fields := c.selectFields(ms)

	query, args, err := builder.Select(columns...).From(ms.TableName).Where(cond).ToSQL()
	if err != nil {
		return err
	}
	query = Search().Limit(1).Apply(query)

	return c.ExecRaw(conn, query, func(stmt *sqlite.Stmt) error {
		return c.Scan(stmt, fields, resultVal)
	}, args...)
}

func (c *Context) selectFields(ms *ModelStruct) ([]string, []*StructField) {
	var columns []string
	var fields []*StructField

	var processField func(sf *StructField, nested bool)
	processField = func(sf *StructField, nested bool) {
		if sf.IsSquashed {
			fields = append(fields, sf)
			for _, nsf := range sf.SquashedFields {
				processField(nsf, true)
			}
		}

		if !sf.IsNormal {
			return
		}
		columns = append(columns, fmt.Sprintf(`%s.%s`, EscapeIdentifier(ms.TableName), EscapeIdentifier(sf.DBName)))
		if !nested {
			fields = append(fields, sf)
		}
	}

	for _, sf := range ms.StructFields {
		processField(sf, false)
	}

	return columns, fields
}
