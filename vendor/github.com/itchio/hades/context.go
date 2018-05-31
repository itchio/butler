package hades

import (
	"github.com/itchio/wharf/state"
)

type Context struct {
	ScopeMap   *ScopeMap
	Consumer   *state.Consumer
	Stats      Stats
	Error      error
	Log        bool
	QueryCount int64
}

type Stats struct {
	Inserts int64
	Updates int64
	Deletes int64
	Current int64
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

func (c *Context) NewScope(value interface{}) *Scope {
	return &Scope{
		Value: value,
		ctx:   c,
	}
}

func (c *Context) AddError(err error) {
	c.Error = err
}
