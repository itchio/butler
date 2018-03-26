package hades

import (
	"reflect"

	"github.com/itchio/wharf/state"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type ScopeMap map[reflect.Type]*gorm.Scope

func (sm ScopeMap) ByDBName(dbName string) *gorm.Scope {
	for _, s := range sm {
		if s.TableName() == dbName {
			return s
		}
	}
	return nil
}

type Context struct {
	Consumer *state.Consumer
	ScopeMap ScopeMap
	Stats    Stats
}

type Stats struct {
	Inserts int64
	Updates int64
	Deletes int64
	Current int64
}

func NewContext(db *gorm.DB, models []interface{}, consumer *state.Consumer) *Context {
	scopeMap := make(ScopeMap)
	for _, m := range models {
		mtyp := reflect.TypeOf(m)
		scopeMap[mtyp] = db.NewScope(m)
	}

	if consumer == nil {
		consumer = &state.Consumer{}
	}

	return &Context{
		Consumer: consumer,
		ScopeMap: scopeMap,
	}
}

type InTransactionFunc func(c *Context, tx *gorm.DB) error

func (c *Context) InTransaction(db *gorm.DB, itf InTransactionFunc) error {
	tx := db.Begin()

	err := itf(c, tx)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "in db transaction")
	} else {
		tx.Commit()
	}

	return nil
}
