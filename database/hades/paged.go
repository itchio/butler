package hades

import (
	"fmt"
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

const maxSqlVars = 900

type DBCB func(db *gorm.DB) *gorm.DB

// retrieve cached items in a []*SomeModel
// for some reason, reflect.New returns a &[]*SomeModel instead,
// I'm guessing slices can't be interfaces, but pointers to slices can?
func (c *Context) pagedByKeys(tx *gorm.DB, keyFieldName string, keys []interface{}, sliceType reflect.Type, cb DBCB) (reflect.Value, error) {
	consumer := c.Consumer

	// actually defaults to 999, but let's get some breathing room
	result := reflect.New(sliceType)
	resultVal := result.Elem()

	remainingItems := keys
	query := fmt.Sprintf("%s in (?)", keyFieldName)
	consumer.Debugf("%d %s to fetch by %s", len(keys), sliceType, keyFieldName)

	for len(remainingItems) > 0 {
		var pageSize int
		if len(remainingItems) > maxSqlVars {
			pageSize = maxSqlVars
		} else {
			pageSize = len(remainingItems)
		}

		consumer.Debugf("Fetching %d items", pageSize)
		pageAddr := reflect.New(sliceType)
		req := tx.Where(query, remainingItems[:pageSize])
		if cb != nil {
			req = cb(req)
		}

		err := req.Find(pageAddr.Interface()).Error
		if err != nil {
			return result, errors.Wrap(err, "performing page fetch")
		}

		appended := reflect.AppendSlice(resultVal, pageAddr.Elem())
		resultVal.Set(appended)
		remainingItems = remainingItems[pageSize:]
	}

	return result, nil
}
