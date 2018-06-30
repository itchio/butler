package hades

import (
	"reflect"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/pkg/errors"
)

const maxSqlVars = 900

type QueryFn func(query string) string

// retrieve cached items in a []*SomeModel
// for some reason, reflect.New returns a &[]*SomeModel instead,
// I'm guessing slices can't be interfaces, but pointers to slices can?
func (c *Context) fetchPagedByPK(conn *sqlite.Conn, PKDBName string, keys []interface{}, sliceType reflect.Type, search Search) (reflect.Value, error) {
	// actually defaults to 999, but let's get some breathing room
	result := reflect.New(sliceType)
	resultVal := result.Elem()

	remainingItems := keys

	for len(remainingItems) > 0 {
		var pageSize int
		if len(remainingItems) > maxSqlVars {
			pageSize = maxSqlVars
		} else {
			pageSize = len(remainingItems)
		}

		pageAddr := reflect.New(sliceType)
		cond := builder.In(EscapeIdentifier(PKDBName), remainingItems[:pageSize]...)

		err := c.Select(conn, pageAddr.Interface(), cond, search)
		if err != nil {
			return result, errors.WithMessage(err, "performing page fetch")
		}

		appended := reflect.AppendSlice(resultVal, pageAddr.Elem())
		resultVal.Set(appended)
		remainingItems = remainingItems[pageSize:]
	}

	return result, nil
}
