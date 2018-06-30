// Copyright (c) 2018 David Crawshaw <david@zentus.com>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

// Package sqliteutil provides utilities for working with SQLite.
package sqliteutil // import "crawshaw.io/sqlite/sqliteutil"

import (
	"fmt"
	"reflect"
	"strings"

	"crawshaw.io/sqlite"
)

// Exec executes an SQLite query.
//
// For each result row, the resultFn is called.
// Result values can be read by resultFn using stmt.Column* methods.
// If resultFn returns an error then iteration ceases and Exec returns
// the error value.
//
// Any args provided to Exec are bound to numbered parameters of the
// query using the Stmt Bind* methods. Basic reflection on args is used
// to map:
//
//	integers to BindInt64
//	floats   to BindFloat
//	[]byte   to BindBytes
//	string   to BindText
//	bool     to BindBool
//
// All other kinds are printed using fmt.Sprintf("%v", v) and passed
// to BindText.
//
// Exec is implemented using the Stmt prepare mechanism which allows
// better interactions with Go's type system and avoids pitfalls of
// passing a Go closure to cgo.
//
// As Exec is implemented using Conn.Prepare, subsequent calls to Exec
// with the same statement will reuse the cached statement object.
//
// Typical use:
//
//	conn := dbpool.Get()
//	defer dbpool.Put(conn)
//
//	if err := sqliteutil.Exec(conn, "INSERT INTO t (a, b, c, d) VALUES (?, ?, ?, ?);", nil, "a1", 1, 42, 1); err != nil {
//		// handle err
//	}
//
//	var a []string
//	var b []int64
//	fn := func(stmt *sqlite.Stmt) error {
//		a = append(a, stmt.ColumnText(0))
//		b = append(b, stmt.ColumnInt64(1))
//		return nil
//	}
//	err := sqlutil.Exec(conn, "SELECT a, b FROM t WHERE c = ? AND d = ?;", fn, 42, 1)
//	if err != nil {
//		// handle err
//	}
func Exec(conn *sqlite.Conn, query string, resultFn func(stmt *sqlite.Stmt) error, args ...interface{}) error {
	stmt, err := conn.Prepare(query)
	if err != nil {
		return annotateErr(err)
	}
	err = exec(stmt, resultFn, args)
	resetErr := stmt.Reset()
	if err == nil {
		err = resetErr
	}
	return err
}

// ExecTransient executes an SQLite query without caching the
// underlying query.
// The interface is exactly the same as Exec.
//
// It is the spiritual equivalent of sqlite3_exec.
func ExecTransient(conn *sqlite.Conn, query string, resultFn func(stmt *sqlite.Stmt) error, args ...interface{}) (err error) {
	var stmt *sqlite.Stmt
	var trailingBytes int
	stmt, trailingBytes, err = conn.PrepareTransient(query)
	if err != nil {
		return annotateErr(err)
	}
	defer func() {
		ferr := stmt.Finalize()
		if err == nil {
			err = ferr
		}
	}()
	if trailingBytes != 0 {
		return fmt.Errorf("sqliteutil.Exec: query %q has trailing bytes", query)
	}
	return exec(stmt, resultFn, args)
}

func exec(stmt *sqlite.Stmt, resultFn func(stmt *sqlite.Stmt) error, args []interface{}) (err error) {
	for i, arg := range args {
		i++ // parameters are 1-indexed
		v := reflect.ValueOf(arg)
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			stmt.BindInt64(i, v.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

			stmt.BindInt64(i, int64(v.Uint()))
		case reflect.Float32, reflect.Float64:
			stmt.BindFloat(i, v.Float())
		case reflect.String:
			stmt.BindText(i, v.String())
		case reflect.Bool:
			stmt.BindBool(i, v.Bool())
		case reflect.Invalid:
			stmt.BindNull(i)
		default:
			if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
				stmt.BindBytes(i, v.Bytes())
			} else {
				stmt.BindText(i, fmt.Sprintf("%v", arg))
			}
		}
	}
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return annotateErr(err)
		}
		if !hasRow {
			break
		}
		if resultFn != nil {
			if err := resultFn(stmt); err != nil {
				if err, isError := err.(sqlite.Error); isError {
					if err.Loc == "" {
						err.Loc = "Exec"
					} else {
						err.Loc = "Exec: " + err.Loc
					}
				}
				// don't modify non-Error errors from resultFn.
				return err
			}
		}
	}
	return nil
}

func annotateErr(err error) error {
	if err, isError := err.(sqlite.Error); isError {
		if err.Loc == "" {
			err.Loc = "Exec"
		} else {
			err.Loc = "Exec: " + err.Loc
		}
		return err
	}
	return fmt.Errorf("sqlutil.Exec: %v", err)
}

// ExecScript executes a script of SQL statements.
//
// The script is wrapped in a SAVEPOINT transaction,
// which is rolled back on any error.
func ExecScript(conn *sqlite.Conn, queries string) (err error) {
	defer Save(conn)(&err)

	for {
		queries = strings.TrimSpace(queries)
		if queries == "" {
			break
		}
		var stmt *sqlite.Stmt
		var trailingBytes int
		stmt, trailingBytes, err = conn.PrepareTransient(queries)
		if err != nil {
			return err
		}
		usedBytes := len(queries) - trailingBytes
		queries = queries[usedBytes:]
		_, err := stmt.Step()
		stmt.Finalize()
		if err != nil {
			return err
		}
	}
	return nil
}
