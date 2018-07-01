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
		Must(err)
	}
	return hadesContext
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func Preload(conn *sqlite.Conn, record interface{}, opts ...hades.PreloadParam) error {
	return HadesContext().Preload(conn, record, opts...)
}

func MustPreload(conn *sqlite.Conn, record interface{}, opts ...hades.PreloadParam) {
	Must(Preload(conn, record, opts...))
}

func SelectOne(conn *sqlite.Conn, result interface{}, cond builder.Cond) (bool, error) {
	return HadesContext().SelectOne(conn, result, cond)
}

func MustSelectOne(conn *sqlite.Conn, result interface{}, cond builder.Cond) bool {
	ok, err := SelectOne(conn, result, cond)
	Must(err)
	return ok
}

func Select(conn *sqlite.Conn, result interface{}, cond builder.Cond, search hades.Search) error {
	return HadesContext().Select(conn, result, cond, search)
}

func MustSelect(conn *sqlite.Conn, result interface{}, cond builder.Cond, search hades.Search) {
	err := Select(conn, result, cond, search)
	Must(err)
}

func Save(conn *sqlite.Conn, record interface{}, opts ...hades.SaveParam) error {
	return HadesContext().Save(conn, record, opts...)
}

func MustSave(conn *sqlite.Conn, record interface{}, opts ...hades.SaveParam) {
	err := Save(conn, record, opts...)
	Must(err)
}

func Exec(conn *sqlite.Conn, b *builder.Builder, resultFn hades.ResultFn) error {
	return HadesContext().Exec(conn, b, resultFn)
}

func MustExec(conn *sqlite.Conn, b *builder.Builder, resultFn hades.ResultFn) {
	err := Exec(conn, b, resultFn)
	Must(err)
}

func ExecWithSearch(conn *sqlite.Conn, b *builder.Builder, search hades.Search, resultFn hades.ResultFn) error {
	query, args, err := b.ToSQL()
	if err != nil {
		return err
	}

	query = search.Apply(query)
	return ExecRaw(conn, query, resultFn, args...)
}

func MustExecWithSearch(conn *sqlite.Conn, b *builder.Builder, search hades.Search, resultFn hades.ResultFn) {
	err := ExecWithSearch(conn, b, search, resultFn)
	Must(err)
}

func ExecRaw(conn *sqlite.Conn, query string, resultFn hades.ResultFn, args ...interface{}) error {
	return HadesContext().ExecRaw(conn, query, resultFn, args...)
}

func MustExecRaw(conn *sqlite.Conn, query string, resultFn hades.ResultFn, args ...interface{}) {
	err := ExecRaw(conn, query, resultFn, args...)
	Must(err)
}

func Delete(conn *sqlite.Conn, model interface{}, cond builder.Cond) error {
	return HadesContext().Delete(conn, model, cond)
}

func MustDelete(conn *sqlite.Conn, model interface{}, cond builder.Cond) {
	err := Delete(conn, model, cond)
	Must(err)
}

func Update(conn *sqlite.Conn, model interface{}, where hades.WhereCond, updates ...builder.Eq) error {
	return HadesContext().Update(conn, model, where, updates...)
}

func MustUpdate(conn *sqlite.Conn, model interface{}, where hades.WhereCond, updates ...builder.Eq) {
	err := Update(conn, model, where, updates...)
	Must(err)
}

func Count(conn *sqlite.Conn, model interface{}, cond builder.Cond) (int64, error) {
	return HadesContext().Count(conn, model, cond)
}

func MustCount(conn *sqlite.Conn, model interface{}, cond builder.Cond) int64 {
	count, err := HadesContext().Count(conn, model, cond)
	Must(err)
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
