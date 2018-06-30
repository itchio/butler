package hades

import (
	"github.com/itchio/wharf/state"
)

type Context struct {
	ScopeMap *ScopeMap
	Consumer *state.Consumer
	Error    error
	Log      bool
}

func NewContext(consumer *state.Consumer, models ...interface{}) (*Context, error) {
	if consumer == nil {
		consumer = &state.Consumer{}
	}
	c := &Context{
		Consumer: consumer,
		ScopeMap: NewScopeMap(),
	}

	for _, m := range models {
		err := c.ScopeMap.Add(c, m)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *Context) TableName(model interface{}) string {
	return c.NewScope(model).TableName()
}

func (c *Context) NewScope(value interface{}) *Scope {
	return &Scope{
		Value: value,
		ctx:   c,
	}
}

func (c *Context) AddError(err error) {
	c.Error = err
}
