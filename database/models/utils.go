package models

import (
	"log"
	"os"
	"time"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/state"
)

var dbConsumer *state.Consumer
var logSql = os.Getenv("BUTLER_SQL_DEBUG") == "1"

func init() {
	dbConsumer = &state.Consumer{}
	if logSql {
		dbConsumer.OnMessage = func(lvl string, message string) {
			log.Printf("[hades] [%s] %s", lvl, message)
		}
	}
}

var hadesContext *hades.Context

func HadesContext() *hades.Context {
	if hadesContext == nil {
		var err error
		hadesContext, err = hades.NewContext(dbConsumer, AllModels...)
		hadesContext.Log = logSql
		if err != nil {
			panic(err)
		}
	}
	return hadesContext
}

func Preload(conn *sqlite.Conn, record interface{}, opts ...hades.PreloadParam) error {
	return HadesContext().Preload(conn, record, opts...)
}

func MustPreload(conn *sqlite.Conn, record interface{}, opts ...hades.PreloadParam) {
	err := Preload(conn, record, opts...)
	if err != nil {
		panic(err)
	}
}

func MustSelectOne(conn *sqlite.Conn, result interface{}, cond builder.Cond) bool {
	ok, err := HadesContext().SelectOne(conn, result, cond)
	if err != nil {
		panic(err)
	}
	return ok
}

func MustSelect(conn *sqlite.Conn, result interface{}, cond builder.Cond, search *hades.SearchParams) {
	err := HadesContext().Select(conn, result, cond, search)
	if err != nil {
		panic(err)
	}
}

func MustSave(conn *sqlite.Conn, record interface{}, opts ...hades.SaveParam) {
	err := HadesContext().Save(conn, record, opts...)
	if err != nil {
		panic(err)
	}
}

func MustExec(conn *sqlite.Conn, b *builder.Builder, resultFn hades.ResultFn) {
	err := HadesContext().Exec(conn, b, resultFn)
	if err != nil {
		panic(err)
	}
}

func MustExecWithSearch(conn *sqlite.Conn, b *builder.Builder, search *hades.SearchParams, resultFn hades.ResultFn) {
	query, args, err := b.ToSQL()
	if err != nil {
		panic(err)
	}

	query = search.Apply(query)
	MustExecRaw(conn, query, resultFn, args...)
}

func MustExecRaw(conn *sqlite.Conn, query string, resultFn hades.ResultFn, args ...interface{}) {
	err := HadesContext().ExecRaw(conn, query, resultFn, args...)
	if err != nil {
		panic(err)
	}
}

func MustDelete(conn *sqlite.Conn, model interface{}, cond builder.Cond) {
	err := HadesContext().Delete(conn, model, cond)
	if err != nil {
		panic(err)
	}
}

func MustUpdate(conn *sqlite.Conn, model interface{}, where hades.WhereCond, updates ...builder.Eq) {
	err := HadesContext().Update(conn, model, where, updates...)
	if err != nil {
		panic(err)
	}
}

func MustCount(conn *sqlite.Conn, model interface{}, cond builder.Cond) int64 {
	count, err := HadesContext().Count(conn, model, cond)
	if err != nil {
		panic(err)
	}
	return count
}

func ColumnTime(col int, stmt *sqlite.Stmt) *time.Time {
	if stmt.ColumnType(col) != sqlite.SQLITE_NULL {
		t, err := time.Parse(time.RFC3339Nano, stmt.ColumnText(col))
		if err == nil {
			return &t
		}
	}
	return nil
}
