package pager

import (
	"encoding/base64"
	"encoding/json"
	"reflect"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

type PagedRequest interface {
	GetLimit() int64
	GetCursor() butlerd.Cursor
}

type Pager interface {
	Fetch(conn *sqlite.Conn, result interface{}, cond builder.Cond, search hades.Search) butlerd.Cursor
}

type pager struct {
	req PagedRequest
}

func New(req PagedRequest) Pager {
	return pager{req}
}

func (p pager) Fetch(conn *sqlite.Conn, result interface{}, cond builder.Cond, search hades.Search) butlerd.Cursor {
	cur := &CursorInfo{}
	cur.Decode(p.req.GetCursor())
	limit := p.req.GetLimit()
	if limit == 0 {
		limit = 5
	}

	search = search.Offset(cur.Offset).Limit(limit + 1)
	models.MustSelect(conn, result, cond, search)

	resVal := reflect.ValueOf(result)
	if resVal.Kind() != reflect.Ptr {
		panic(errors.Errorf("Expected _pointer_ to slice, had: %v", resVal.Type()))
	}
	sliceVal := resVal.Elem()
	if sliceVal.Kind() != reflect.Slice {
		panic(errors.Errorf("Expected pointer to _slice_, had: %v", resVal.Type()))
	}

	var nextCur *CursorInfo
	if int64(sliceVal.Len()) > limit {
		nextCur = &CursorInfo{
			Offset: cur.Offset + limit,
		}
		sliceVal.Set(sliceVal.Slice(0, sliceVal.Len()-1))
	}
	return nextCur.Encode()
}

// cursors!

type CursorInfo struct {
	Offset int64 `json:"offset"`
}

func (info *CursorInfo) Decode(c butlerd.Cursor) {
	if c == "" {
		// explicitly ignore error, invalid cursors mean no cursors
		return
	}

	bs, err := base64.StdEncoding.DecodeString(string(c))
	if err != nil {
		// explicitly ignore error, invalid cursors mean no cursors
		return
	}

	err = json.Unmarshal(bs, info)
	if err != nil {
		// explicitly ignore error, invalid cursors mean no cursors
		return
	}
}

func (info *CursorInfo) Encode() butlerd.Cursor {
	if info == nil {
		return ""
	}

	bs, err := json.Marshal(info)
	if err != nil {
		// explicitly ignore errors
		return butlerd.Cursor("")
	}

	cur := base64.StdEncoding.EncodeToString(bs)
	return butlerd.Cursor(cur)
}

func Ordering(defaultOrder string, reverse bool) string {
	if reverse {
		switch defaultOrder {
		case "ASC":
			return "DESC"
		case "DESC":
			return "ASC"
		default:
			panic(errors.Errorf("Unknown ordering '%s'", defaultOrder))
		}
	}
	return defaultOrder
}
