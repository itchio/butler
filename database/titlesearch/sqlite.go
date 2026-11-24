package titlesearch

import (
	"crawshaw.io/sqlite"
	"xorm.io/builder"
)

// Register makes title_matches(title, query) available to SQL run on the
// connection. Cond builds conditions with it, so every connection those
// run on has to be registered.
func Register(conn *sqlite.Conn) error {
	// a statement calls this for one query and many titles
	var lastQuery string
	var q Query

	return conn.CreateFunction("title_matches", true, 2, func(ctx sqlite.Context, values ...sqlite.Value) {
		if query := values[1].Text(); query != lastQuery {
			lastQuery = query
			q = NewQuery(query)
		}

		matches := q.Empty()
		if !matches {
			title := NewTitle(values[0].Text())
			matches = q.Tier(&title) != TierNone
		}
		if matches {
			ctx.ResultInt(1)
		} else {
			ctx.ResultInt(0)
		}
	}, nil, nil)
}

// Cond restricts a query to rows whose title column matches the query. A
// query with nothing to match on leaves every row in.
func Cond(column string, query string) builder.Cond {
	return builder.Expr("title_matches("+column+", ?)", query)
}
