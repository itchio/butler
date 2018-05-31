package hades

import (
	"fmt"
	"strings"
)

type SearchParams struct {
	orders []string
	offset *int64
	limit  *int64
}

func Search() *SearchParams {
	return &SearchParams{}
}

func (s *SearchParams) OrderBy(order string) *SearchParams {
	s.orders = append(s.orders, order)
	return s
}

func (s *SearchParams) Limit(limit int64) *SearchParams {
	s.limit = &limit
	return s
}

func (s *SearchParams) Offset(offset int64) *SearchParams {
	s.offset = &offset
	return s
}

func (s *SearchParams) Apply(sql string) string {
	if s != nil {
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
	}
	return sql
}

func (s *SearchParams) String() string {
	return s.Apply("")
}
