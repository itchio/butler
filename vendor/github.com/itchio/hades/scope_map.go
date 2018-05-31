package hades

import (
	"reflect"

	"github.com/pkg/errors"
)

type ScopeMap struct {
	byType   map[reflect.Type]*Scope
	byDBName map[string]*Scope
}

func NewScopeMap() *ScopeMap {
	return &ScopeMap{
		byType:   make(map[reflect.Type]*Scope),
		byDBName: make(map[string]*Scope),
	}
}

func (sm *ScopeMap) Add(c *Context, m interface{}) error {
	val := reflect.ValueOf(m)

	if val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Type().Kind() == reflect.Interface {
		val = val.Elem()
	}

	reflectType := val.Type()

	// what should we do if it's not a struct?
	if reflectType.Kind() != reflect.Struct {
		return errors.Errorf("hades expects all models to be structs, but got %v instead", reflectType)
	}

	s := c.NewScope(m)
	sm.byType[reflect.PtrTo(reflectType)] = s
	sm.byDBName[s.TableName()] = s
	return nil
}

func (sm *ScopeMap) ByDBName(dbname string) *Scope {
	return sm.byDBName[dbname]
}

func (sm *ScopeMap) ByType(typ reflect.Type) *Scope {
	return sm.byType[typ]
}
