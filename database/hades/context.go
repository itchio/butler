package hades

import (
	"reflect"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/database"
	"github.com/itchio/wharf/state"
	"github.com/jinzhu/gorm"
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

func NewContext(db *gorm.DB, consumer *state.Consumer) *Context {
	scopeMap := make(ScopeMap)
	for _, m := range database.Models {
		mtyp := reflect.TypeOf(m)
		scopeMap[mtyp] = db.NewScope(m)
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
		return errors.Wrap(err, 0)
	} else {
		tx.Commit()
	}

	return nil
}
