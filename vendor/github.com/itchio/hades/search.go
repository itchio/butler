package hades

import (
	"fmt"
	"strings"
)

type Search struct {
	orders []string
	offset *int64
	limit  *int64
}

func (s Search) OrderBy(order string) Search {
	s.orders = append(s.orders, order)
	return s
}

func (s Search) Limit(limit int64) Search {
	s.limit = &limit
	return s
}

func (s Search) Offset(offset int64) Search {
	s.offset = &offset
	return s
}

func (s Search) Apply(sql string) string {
	if len(s.orders) > 0 {
		sql = fmt.Sprintf("%s ORDER BY %s", sql, strings.Join(s.orders, ", "))
	}

	if s.limit != nil || s.offset != nil {
		var limit int64 = -1
		if s.limit != nil {
			limit = *s.limit
		}

		// offset must appear without limit,
		// and a negative limit means no limit.
		// see https://www.sqlite.org/lang_select.html#limitoffset
		sql = fmt.Sprintf("%s LIMIT %d", sql, limit)

		if s.offset != nil {
			sql = fmt.Sprintf("%s OFFSET %d", sql, *s.offset)
		}
	}
	return sql
}

func (s Search) String() string {
	return s.Apply("")
}
