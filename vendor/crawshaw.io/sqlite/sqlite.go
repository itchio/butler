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

package sqlite

// #cgo CFLAGS: -DSQLITE_THREADSAFE=2
// #cgo CFLAGS: -DSQLITE_DEFAULT_WAL_SYNCHRONOUS=1
// #cgo CFLAGS: -DSQLITE_ENABLE_UNLOCK_NOTIFY
// #cgo CFLAGS: -DSQLITE_ENABLE_FTS5
// #cgo CFLAGS: -DSQLITE_ENABLE_RTREE
// #cgo CFLAGS: -DSQLITE_LIKE_DOESNT_MATCH_BLOBS
// #cgo CFLAGS: -DSQLITE_OMIT_DEPRECATED
// #cgo CFLAGS: -DSQLITE_ENABLE_JSON1
// #cgo CFLAGS: -DSQLITE_ENABLE_SESSION
// #cgo CFLAGS: -DSQLITE_ENABLE_PREUPDATE_HOOK
// #cgo CFLAGS: -DSQLITE_USE_ALLOCA
// #cgo CFLAGS: -DSQLITE_ENABLE_COLUMN_METADATA
// #cgo windows LDFLAGS: -Wl,-Bstatic -lwinpthread -Wl,-Bdynamic
// #cgo linux LDFLAGS: -ldl -lm
// #cgo linux CFLAGS: -std=c99
// #cgo openbsd LDFLAGS: -lm
// #cgo openbsd CFLAGS: -std=c99
//
// #include <blocking_step.h>
// #include <sqlite3.h>
// #include <stdlib.h>
// #include <string.h>
//
// // Use a helper function here to avoid the cgo pointer detection
// // logic treating SQLITE_TRANSIENT as a Go pointer.
// static int transient_bind_text(sqlite3_stmt* stmt, int col, char* p, int n) {
//	return sqlite3_bind_text(stmt, col, p, n, SQLITE_TRANSIENT);
// }
//
// extern void log_fn(void* pArg, int code, char* msg);
// static void enable_logging() {
//	sqlite3_config(SQLITE_CONFIG_LOG, log_fn, NULL);
// }
import "C"
import (
	"bytes"
	"runtime"
	"sync"
	"unsafe"
)

// Conn is an open connection to an SQLite3 database.
//
// A Conn can only be used by goroutine at a time.
type Conn struct {
	conn   *C.sqlite3
	stmts  map[string]*Stmt
	closed bool
	count  int // shared variable to help the race detector find Conn misuse

	cancelCh   chan struct{}
	doneCh     <-chan struct{}
	unlockNote *C.unlock_note
	file       string
	line       int
}

// OpenFlags are flags used when opening a Conn.
//
// https://www.sqlite.org/c3ref/c_open_autoproxy.html
type OpenFlags int

const (
	SQLITE_OPEN_READONLY       = OpenFlags(C.SQLITE_OPEN_READONLY)
	SQLITE_OPEN_READWRITE      = OpenFlags(C.SQLITE_OPEN_READWRITE)
	SQLITE_OPEN_CREATE         = OpenFlags(C.SQLITE_OPEN_CREATE)
	SQLITE_OPEN_URI            = OpenFlags(C.SQLITE_OPEN_URI)
	SQLITE_OPEN_MEMORY         = OpenFlags(C.SQLITE_OPEN_MEMORY)
	SQLITE_OPEN_MAIN_DB        = OpenFlags(C.SQLITE_OPEN_MAIN_DB)
	SQLITE_OPEN_TEMP_DB        = OpenFlags(C.SQLITE_OPEN_TEMP_DB)
	SQLITE_OPEN_TRANSIENT_DB   = OpenFlags(C.SQLITE_OPEN_TRANSIENT_DB)
	SQLITE_OPEN_MAIN_JOURNAL   = OpenFlags(C.SQLITE_OPEN_MAIN_JOURNAL)
	SQLITE_OPEN_TEMP_JOURNAL   = OpenFlags(C.SQLITE_OPEN_TEMP_JOURNAL)
	SQLITE_OPEN_SUBJOURNAL     = OpenFlags(C.SQLITE_OPEN_SUBJOURNAL)
	SQLITE_OPEN_MASTER_JOURNAL = OpenFlags(C.SQLITE_OPEN_MASTER_JOURNAL)
	SQLITE_OPEN_NOMUTEX        = OpenFlags(C.SQLITE_OPEN_NOMUTEX)
	SQLITE_OPEN_FULLMUTEX      = OpenFlags(C.SQLITE_OPEN_FULLMUTEX)
	SQLITE_OPEN_SHAREDCACHE    = OpenFlags(C.SQLITE_OPEN_SHAREDCACHE)
	SQLITE_OPEN_PRIVATECACHE   = OpenFlags(C.SQLITE_OPEN_PRIVATECACHE)
	SQLITE_OPEN_WAL            = OpenFlags(C.SQLITE_OPEN_WAL)
)

// OpenConn opens a single SQLite database connection.
// A flags value of 0 defaults to:
//
//	SQLITE_OPEN_READWRITE
//	SQLITE_OPEN_CREATE
//	SQLITE_OPEN_SHAREDCACHE
//	SQLITE_OPEN_WAL
//	SQLITE_OPEN_URI
//	SQLITE_OPEN_NOMUTEX
//
// https://www.sqlite.org/c3ref/open.html
func OpenConn(path string, flags OpenFlags) (*Conn, error) {
	return openConn(path, flags)
}

func openConn(path string, flags OpenFlags) (*Conn, error) {
	sqliteInit.Do(sqliteInitFn)
	if flags == 0 {
		flags = SQLITE_OPEN_READWRITE | SQLITE_OPEN_CREATE | SQLITE_OPEN_SHAREDCACHE | SQLITE_OPEN_WAL | SQLITE_OPEN_URI | SQLITE_OPEN_NOMUTEX
	}
	conn := &Conn{
		stmts: make(map[string]*Stmt),
		// A pointer to unlockNote is retained by C,
		// so we allocate it on the C heap.
		unlockNote: C.unlock_note_alloc(),
	}

	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	res := C.sqlite3_open_v2(cpath, &conn.conn, C.int(flags), nil)
	if res != 0 {
		extres := C.sqlite3_extended_errcode(conn.conn)
		if extres != 0 {
			res = extres
		}
		C.sqlite3_close_v2(conn.conn)
		return nil, reserr("OpenConn", path, "", res)
	}
	C.sqlite3_extended_result_codes(conn.conn, 1)

	// TODO: only if Debug ?
	_, file, line, _ := runtime.Caller(2) // caller of OpenConn or Open
	runtime.SetFinalizer(conn, func(conn *Conn) {
		if !conn.closed {
			var buf [20]byte
			panic(file + ":" + string(itoa(buf[:], int64(line))) + ": *sqlite.Conn garbage collected, call Close method")
		}
	})

	if flags&SQLITE_OPEN_WAL > 0 {
		stmt, _, err := conn.PrepareTransient("PRAGMA journal_mode=wal;")
		if err != nil {
			conn.Close()
			return nil, err
		}
		defer stmt.Finalize()
		if _, err := stmt.Step(); err != nil {
			conn.Close()
			return nil, err
		}
	}

	return conn, nil
}

// Close closes the database connection using sqlite3_close_v2.
// https://www.sqlite.org/c3ref/close.html
func (conn *Conn) Close() error {
	conn.cancelInterrupt()
	conn.closed = true
	res := C.sqlite3_close_v2(conn.conn)
	C.unlock_note_free(conn.unlockNote)
	conn.unlockNote = nil
	return reserr("Conn.Close", "", "", res)
}

// SetInterrupt assigns a channel to control connection execution lifetime.
//
// When doneCh is closed, the connection uses sqlite3_interrupt to
// stop long-running queries and cancels any *Stmt.Step calls that
// are blocked waiting for the database write lock.
//
// Subsequent uses of the connection will return SQLITE_INTERRUPT
// errors until doneCh is reset with a subsequent call to SetInterrupt.
//
// Typically, doneCh is provided by the Done method on a context.Context.
// For example, a timeout can be associated with a connection session:
//
//	ctx := context.WithTimeout(context.Background(), 100*time.Millisecond)
//	conn.SetInterrupt(ctx.Done())
func (conn *Conn) SetInterrupt(doneCh <-chan struct{}) {
	if conn.closed {
		panic("sqlite.Conn is closed")
	}
	conn.cancelInterrupt()
	conn.doneCh = doneCh
	if doneCh == nil {
		return
	}
	cancelCh := make(chan struct{})
	conn.cancelCh = cancelCh
	go func() {
		select {
		case <-doneCh:
			C.sqlite3_interrupt(conn.conn)
			C.unlock_note_fire(conn.unlockNote)
			<-cancelCh
			cancelCh <- struct{}{}
		case <-cancelCh:
			cancelCh <- struct{}{}
		}
	}()
}

func (conn *Conn) interrupted(loc, query string) error {
	select {
	case <-conn.doneCh:
		return reserr(loc, query, "", C.SQLITE_INTERRUPT)
	default:
		return nil
	}
}

func (conn *Conn) cancelInterrupt() {
	if conn.cancelCh != nil {
		conn.cancelCh <- struct{}{}
		<-conn.cancelCh
		conn.cancelCh = nil
	}
}

// Prep returns a persistent SQL statement.
//
// Any error in preparation will panic.
//
// Persistent prepared statements are cached by the query
// string in a Conn. If Finalize is not called, then subsequent
// calls to Prepare will return the same statement.
//
// https://www.sqlite.org/c3ref/prepare.html
func (conn *Conn) Prep(query string) *Stmt {
	stmt, err := conn.Prepare(query)
	if err != nil {
		if ErrCode(err) == SQLITE_INTERRUPT {
			return &Stmt{
				conn:         conn,
				query:        query,
				bindNames:    make(map[string]int),
				colNames:     make(map[string]int),
				prepInterupt: true,
			}
		}
		panic(err)
	}
	return stmt
}

// Prepare prepares a persistent SQL statement.
//
// Persistent prepared statements are cached by the query
// string in a Conn. If Finalize is not called, then subsequent
// calls to Prepare will return the same statement.
//
// If the query has any unprocessed trailing bytes, Prepare
// returns an error.
//
// https://www.sqlite.org/c3ref/prepare.html
func (conn *Conn) Prepare(query string) (*Stmt, error) {
	if stmt := conn.stmts[query]; stmt != nil {
		stmt.Reset()
		stmt.ClearBindings()
		return stmt, nil
	}
	stmt, trailingBytes, err := conn.prepare(query, C.SQLITE_PREPARE_PERSISTENT)
	if err != nil {
		return nil, err
	}
	if trailingBytes != 0 {
		stmt.Finalize()
		return nil, reserr("Conn.Prepare", query, "statement has trailing bytes", C.SQLITE_ERROR)
	}
	conn.stmts[query] = stmt
	return stmt, nil
}

// PrepareTransient prepares an SQL statement that is not cached by
// the Conn. Subsequent calls with the same query will create new Stmts.
//
// The number of trailing bytes not consumed from query is returned.
//
// To run a sequence of queries once as part of a script,
// the sqliteutil package provides an ExecScript function built on this.
//
// https://www.sqlite.org/c3ref/prepare.html
func (conn *Conn) PrepareTransient(query string) (stmt *Stmt, trailingBytes int, err error) {
	return conn.prepare(query, 0)

}

func (conn *Conn) prepare(query string, flags C.uint) (*Stmt, int, error) {
	conn.count++
	if err := conn.interrupted("Conn.Prepare", query); err != nil {
		return nil, 0, err
	}

	stmt := &Stmt{
		conn:      conn,
		query:     query,
		bindNames: make(map[string]int),
		colNames:  make(map[string]int),
	}
	cquery := C.CString(query)
	defer C.free(unsafe.Pointer(cquery))
	var ctrailing *C.char
	res := C.sqlite3_prepare_v3(conn.conn, cquery, -1, flags, &stmt.stmt, &ctrailing)
	if err := conn.extreserr("Conn.Prepare", query, res); err != nil {
		return nil, 0, err
	}
	trailingBytes := int(C.strlen(ctrailing))

	// TODO: only if Debug ?
	runtime.SetFinalizer(stmt, func(stmt *Stmt) {
		if stmt.conn != nil && !stmt.conn.closed {
			panic("open *sqlite.Stmt \"" + query + "\" garbage collected, call Finalize")
		}
	})

	for i, count := 1, stmt.BindParamCount(); i <= count; i++ {
		cname := C.sqlite3_bind_parameter_name(stmt.stmt, C.int(i))
		if cname != nil {
			stmt.bindNames[C.GoString(cname)] = i
		}
	}

	for i, count := 0, int(C.sqlite3_column_count(stmt.stmt)); i < count; i++ {
		cname := C.sqlite3_column_name(stmt.stmt, C.int(i))
		if cname != nil {
			stmt.colNames[C.GoString(cname)] = i
		}
	}

	return stmt, trailingBytes, nil
}

// Changes reports the number of rows affected by the most recent statement.
//
// https://www.sqlite.org/c3ref/changes.html
func (conn *Conn) Changes() int {
	conn.count++
	return int(C.sqlite3_changes(conn.conn))
}

// LastInsertRowID reports the rowid of the most recently successful INSERT.
//
// https://www.sqlite.org/c3ref/last_insert_rowid.html
func (conn *Conn) LastInsertRowID() int64 {
	conn.count++
	return int64(C.sqlite3_last_insert_rowid(conn.conn))
}

// extreserr asks SQLite for a string explaining the error.
// Only called for errors that are probably program bugs.
func (conn *Conn) extreserr(loc, query string, res C.int) error {
	msg := ""
	switch res {
	case C.SQLITE_OK, C.SQLITE_ROW, C.SQLITE_DONE:
		return nil
	case C.SQLITE_BUSY:
	default:
		msg = C.GoString(C.sqlite3_errmsg(conn.conn))
	}
	return reserr(loc, query, msg, res)
}

func (conn *Conn) reserr(loc, query string, res C.int) error {
	switch res {
	case C.SQLITE_OK, C.SQLITE_ROW, C.SQLITE_DONE:
		return nil
	}
	// TODO
	/*extres := C.sqlite3_extended_errcode(conn.conn)
	if extres != 0 {
		res = extres
	}*/
	return reserr(loc, query, "", res)
}

func reserr(loc, query, msg string, res C.int) error {
	if res != 0 {
		return Error{
			Code:  ErrorCode(res),
			Loc:   loc,
			Query: query,
			Msg:   msg,
		}
	}
	return nil
}

// Stmt is an SQLite3 prepared statement.
//
// A Stmt is attached to a particular Conn
// (and that Conn can only be used by a single goroutine).
//
// When a Stmt is no longer needed it should be cleaned up
// by calling the Finalize method.
type Stmt struct {
	conn         *Conn
	stmt         *C.sqlite3_stmt
	query        string
	bindNames    map[string]int
	colNames     map[string]int
	bindErr      error
	prepInterupt bool // set if Prep was interrupted
}

func (stmt *Stmt) interrupted(loc string) error {
	if stmt.prepInterupt {
		return reserr(loc, stmt.query, "", C.SQLITE_INTERRUPT)
	}
	return stmt.conn.interrupted(loc, stmt.query)
}

// Finalize deletes a prepared statement.
//
// Be sure to always call Finalize when done with
// a statement created using PrepareTransient.
//
// Do not call Finalize on a prepared statement that
// you intend to prepare again in the future.
//
// https://www.sqlite.org/c3ref/finalize.html
func (stmt *Stmt) Finalize() error {
	stmt.conn.count++
	if ptr := stmt.conn.stmts[stmt.query]; ptr == stmt {
		delete(stmt.conn.stmts, stmt.query)
	}
	res := C.sqlite3_finalize(stmt.stmt)
	stmt.conn = nil
	return stmt.conn.reserr("Stmt.Finalize", stmt.query, res)
}

// Reset resets a prepared statement so it can be executed again.
//
// Note that any parameter values bound to the statement are retained.
// To clear bound values, call ClearBindings.
//
// https://www.sqlite.org/c3ref/reset.html
func (stmt *Stmt) Reset() error {
	stmt.conn.count++
	if err := stmt.interrupted("Stmt.Reset"); err != nil {
		return err
	}
	res := C.sqlite3_reset(stmt.stmt)
	return stmt.conn.reserr("Stmt.Reset", stmt.query, res)
}

// ClearBindings clears all bound parameter values on a statement.
//
// https://www.sqlite.org/c3ref/clear_bindings.html
func (stmt *Stmt) ClearBindings() error {
	stmt.conn.count++
	if err := stmt.interrupted("Stmt.ClearBindings"); err != nil {
		return err
	}
	res := C.sqlite3_clear_bindings(stmt.stmt)
	return stmt.conn.reserr("Stmt.ClearBindings", stmt.query, res)
}

// Step moves through the statement cursor using sqlite3_step.
//
// If a row of data is available, rowReturned is reported as true.
// If the statement has reached the end of the available data then
// rowReturned is false.
//
// https://www.sqlite.org/c3ref/step.html
//
// As the sqlite package enables shared cache mode by default
// and multiple writers are common in multi-threaded programs,
// this Step method uses sqlite3_unlock_notify to handle any
// SQLITE_LOCKED errors.
//
// Without the shared cache, SQLite will block for
// several seconds while trying to acquire the write lock.
// With the shared cache, it returns SQLITE_LOCKED immediately
// if the write lock is held by another connection in this process.
// Dealing with this correctly makes for an unpleasant programming
// experience, so this package does it automatically by blocking
// Step until the write lock is relinquished.
//
// This means Step can block for a very long time.
// Use SetInterrupt to control how long Step will block.
//
// For far more details, see:
//
//	http://www.sqlite.org/unlock_notify.html
func (stmt *Stmt) Step() (rowReturned bool, err error) {
	if stmt.bindErr != nil {
		err = stmt.bindErr
		stmt.bindErr = nil
		return false, err
	}

	for {
		stmt.conn.count++
		if err := stmt.interrupted("Stmt.Step"); err != nil {
			return false, err
		}
		switch res := C.sqlite3_step(stmt.stmt); uint8(res) { // reduce to non-extended error code
		case C.SQLITE_LOCKED:
			if res := C.wait_for_unlock_notify(stmt.conn.conn, stmt.conn.unlockNote); res != C.SQLITE_OK {
				return false, stmt.conn.reserr("Stmt.Step(Wait)", stmt.query, res)
			}
			C.sqlite3_reset(stmt.stmt)
			// loop
		case C.SQLITE_ROW:
			return true, nil
		case C.SQLITE_DONE:
			return false, nil
		case C.SQLITE_BUSY, C.SQLITE_INTERRUPT, C.SQLITE_CONSTRAINT:
			// TODO: embed some of these errors into the stmt for zero-alloc errors?
			return false, stmt.conn.reserr("Stmt.Step", stmt.query, res)
		default:
			return false, stmt.conn.extreserr("Stmt.Step", stmt.query, res)
		}
	}
}

func (stmt *Stmt) handleBindErr(loc string, res C.int) {
	if stmt.bindErr == nil {
		stmt.bindErr = stmt.conn.reserr(loc, stmt.query, res)
	}
}

func (stmt *Stmt) findBindName(loc string, param string) int {
	pos := stmt.bindNames[param]
	if pos == 0 && stmt.bindErr == nil {
		stmt.bindErr = reserr(loc, stmt.query, "unknown parameter: "+param, C.SQLITE_ERROR)
	}
	return pos
}

// BindParamCount reports the number of parameters in stmt.
// https://www.sqlite.org/c3ref/bind_parameter_count.html
func (stmt *Stmt) BindParamCount() int {
	return int(C.sqlite3_bind_parameter_count(stmt.stmt))
}

// BindInt64 binds value to a numbered stmt parameter.
// https://www.sqlite.org/c3ref/bind_blob.html
func (stmt *Stmt) BindInt64(param int, value int64) {
	res := C.sqlite3_bind_int64(stmt.stmt, C.int(param), C.sqlite3_int64(value))
	stmt.handleBindErr("BindInt64", res)
}

// BindBool binds value (as an integer 0 or 1) to a numbered stmt parameter.
func (stmt *Stmt) BindBool(param int, value bool) {
	v := 0
	if value {
		v = 1
	}
	res := C.sqlite3_bind_int64(stmt.stmt, C.int(param), C.sqlite3_int64(v))
	stmt.handleBindErr("BindBool", res)
}

// BindBytes binds value to a numbered stmt parameter.
//
// In-memory copies of value are made using this interface.
// For large blobs, consider using the streaming Blob object.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (stmt *Stmt) BindBytes(param int, value []byte) {
	var v *C.char
	if len(value) != 0 {
		v = (*C.char)(unsafe.Pointer(&value[0]))
	}
	res := C.transient_bind_text(stmt.stmt, C.int(param), v, C.int(len(value)))
	runtime.KeepAlive(value)
	stmt.handleBindErr("BindBytes", res)
}

var emptyCstr = C.CString("")

// BindText binds value to a numbered stmt parameter.
// https://www.sqlite.org/c3ref/bind_blob.html
func (stmt *Stmt) BindText(param int, value string) {
	var v *C.char
	var free *[0]byte
	if len(value) == 0 {
		v = emptyCstr
	} else {
		v = C.CString(value)
		free = (*[0]byte)(C.free)
	}
	res := C.sqlite3_bind_text(stmt.stmt, C.int(param), v, C.int(len(value)), free)
	stmt.handleBindErr("BindText", res)
}

// BindFloat binds value to a numbered stmt parameter.
// Parameter indicies start at 1.
// https://www.sqlite.org/c3ref/bind_blob.html
func (stmt *Stmt) BindFloat(param int, value float64) {
	res := C.sqlite3_bind_double(stmt.stmt, C.int(param), C.double(value))
	stmt.handleBindErr("BindFloat", res)
}

// BindNull binds an SQL NULL value to a numbered stmt parameter.
// Parameter indicies start at 1.
// https://www.sqlite.org/c3ref/bind_blob.html
func (stmt *Stmt) BindNull(param int) {
	res := C.sqlite3_bind_null(stmt.stmt, C.int(param))
	stmt.handleBindErr("BindNull", res)
}

// BindNull binds a blob of zeros of length len to a numbered stmt parameter.
// Parameter indicies start at 1.
// https://www.sqlite.org/c3ref/bind_blob.html
func (stmt *Stmt) BindZeroBlob(param int, len int64) {
	res := C.sqlite3_bind_zeroblob64(stmt.stmt, C.int(param), C.sqlite3_uint64(len))
	stmt.handleBindErr("BindZeroBlob", res)
}

// SetInt64 binds an int64 to a parameter using a column name.
func (stmt *Stmt) SetInt64(param string, value int64) {
	stmt.BindInt64(stmt.findBindName("SetInt64", param), value)
}

// SetBool binds a value (as a 0 or 1) to a parameter using a column name.
func (stmt *Stmt) SetBool(param string, value bool) {
	stmt.BindBool(stmt.findBindName("SetBool", param), value)
}

// SetBytes binds bytes to a parameter using a column name.
// An invalid parameter name will cause the call to Step to return an error.
func (stmt *Stmt) SetBytes(param string, value []byte) {
	stmt.BindBytes(stmt.findBindName("SetBytes", param), value)
}

// SetText binds text to a parameter using a column name.
// An invalid parameter name will cause the call to Step to return an error.
func (stmt *Stmt) SetText(param string, value string) {
	stmt.BindText(stmt.findBindName("SetText", param), value)
}

// SetFloat binds a float64 to a parameter using a column name.
// An invalid parameter name will cause the call to Step to return an error.
func (stmt *Stmt) SetFloat(param string, value float64) {
	stmt.BindFloat(stmt.findBindName("SetFloat", param), value)
}

// SetNull binds a null to a parameter using a column name.
// An invalid parameter name will cause the call to Step to return an error.
func (stmt *Stmt) SetNull(param string) {
	stmt.BindNull(stmt.findBindName("SetNull", param))
}

// SetZeroBlob binds a zero blob of length len to a parameter using a column name.
// An invalid parameter name will cause the call to Step to return an error.
func (stmt *Stmt) SetZeroBlob(param string, len int64) {
	stmt.BindZeroBlob(stmt.findBindName("SetZeroBlob", param), len)
}

// ColumnInt returns a query result value as an int.
//
// Note: this method calls sqlite3_column_int64 and then converts the
// resulting 64-bits to an int.
//
// Column indicies start at 0.
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnInt(col int) int {
	return int(stmt.ColumnInt64(col))
}

// ColumnInt32 returns a query result value as an int32.
//
// Column indicies start at 0.
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnInt32(col int) int32 {
	return int32(C.sqlite3_column_int(stmt.stmt, C.int(col)))
}

// ColumnInt64 returns a query result value as an int64.
//
// Column indicies start at 0.
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnInt64(col int) int64 {
	return int64(C.sqlite3_column_int64(stmt.stmt, C.int(col)))
}

// ColumnBytes reads a query result into buf.
// It reports the number of bytes read.
//
// Column indicies start at 0.
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnBytes(col int, buf []byte) int {
	return copy(buf, stmt.columnBytes(col))
}

// ColumnReader creates a byte reader for a query result column.
//
// The reader directly references C-managed memory that stops
// being valid as soon as the statement row resets.
func (stmt *Stmt) ColumnReader(col int) *bytes.Reader {
	// Load the C memory directly into the Reader.
	// There is no exported method that lets it escape.
	return bytes.NewReader(stmt.columnBytes(col))
}

func (stmt *Stmt) columnBytes(col int) []byte {
	p := C.sqlite3_column_blob(stmt.stmt, C.int(col))
	if p == nil {
		return nil
	}
	n := stmt.ColumnLen(col)
	return (*[1 << 28]byte)(unsafe.Pointer(p))[:n:n]
}

// ColumnType are codes for each of the SQLite fundamental datatypes:
//
//   64-bit signed integer
//   64-bit IEEE floating point number
//   string
//   BLOB
//   NULL
//
// https://www.sqlite.org/c3ref/c_blob.html
type ColumnType int

const (
	SQLITE_INTEGER = ColumnType(C.SQLITE_INTEGER)
	SQLITE_FLOAT   = ColumnType(C.SQLITE_FLOAT)
	SQLITE_TEXT    = ColumnType(C.SQLITE3_TEXT)
	SQLITE_BLOB    = ColumnType(C.SQLITE_BLOB)
	SQLITE_NULL    = ColumnType(C.SQLITE_NULL)
)

func (t ColumnType) String() string {
	switch t {
	case SQLITE_INTEGER:
		return "SQLITE_INTEGER"
	case SQLITE_FLOAT:
		return "SQLITE_FLOAT"
	case SQLITE_TEXT:
		return "SQLITE_TEXT"
	case SQLITE_BLOB:
		return "SQLITE_BLOB"
	case SQLITE_NULL:
		return "SQLITE_NULL"
	default:
		return "<unknown sqlite datatype>"
	}
}

// ColumnType returns the datatype code for the initial data
// type of the result column. The returned value is one of:
//
//   SQLITE_INTEGER
//   SQLITE_FLOAT
//   SQLITE_TEXT
//   SQLITE_BLOB
//   SQLITE_NULL
//
// Column indicies start at 0.
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnType(col int) ColumnType {
	return ColumnType(C.sqlite3_column_type(stmt.stmt, C.int(col)))
}

// ColumnText returns a query result as a string.
//
// Column indicies start at 0.
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnText(col int) string {
	n := stmt.ColumnLen(col)
	return C.GoStringN((*C.char)(unsafe.Pointer(C.sqlite3_column_text(stmt.stmt, C.int(col)))), C.int(n))
}

// ColumnFloat returns a query result as a float64.
//
// Column indicies start at 0.
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnFloat(col int) float64 {
	return float64(C.sqlite3_column_double(stmt.stmt, C.int(col)))
}

// ColumnLen returns the number of bytes in a query result.
//
// Column indicies start at 0.
// https://www.sqlite.org/c3ref/column_blob.html
func (stmt *Stmt) ColumnLen(col int) int {
	return int(C.sqlite3_column_bytes(stmt.stmt, C.int(col)))
}

func (stmt *Stmt) ColumnDatabaseName(col int) string {
	return C.GoString((*C.char)(unsafe.Pointer(C.sqlite3_column_database_name(stmt.stmt, C.int(col)))))
}

func (stmt *Stmt) ColumnTableName(col int) string {
	return C.GoString((*C.char)(unsafe.Pointer(C.sqlite3_column_table_name(stmt.stmt, C.int(col)))))
}

// GetInt64 returns a query result value for colName as an int64.
func (stmt *Stmt) GetInt64(colName string) int64 {
	col, found := stmt.colNames[colName]
	if !found {
		return 0
	}
	return stmt.ColumnInt64(col)
}

// GetBytes reads a query result for colName into buf.
// It reports the number of bytes read.
func (stmt *Stmt) GetBytes(colName string, buf []byte) int {
	col, found := stmt.colNames[colName]
	if !found {
		return 0
	}
	return stmt.ColumnBytes(col, buf)
}

// GetReader creates a byte reader for colName.
//
// The reader directly references C-managed memory that stops
// being valid as soon as the statement row resets.
func (stmt *Stmt) GetReader(colName string) *bytes.Reader {
	col, found := stmt.colNames[colName]
	if !found {
		return bytes.NewReader(nil)
	}
	return stmt.ColumnReader(col)
}

// GetText returns a query result value for colName as a string.
func (stmt *Stmt) GetText(colName string) string {
	col, found := stmt.colNames[colName]
	if !found {
		return ""
	}
	return stmt.ColumnText(col)
}

// GetFloat returns a query result value for colName as a float64.
func (stmt *Stmt) GetFloat(colName string) float64 {
	col, found := stmt.colNames[colName]
	if !found {
		return 0
	}
	return stmt.ColumnFloat(col)
}

// GetLen returns the number of bytes in a query result for colName.
func (stmt *Stmt) GetLen(colName string) int {
	col, found := stmt.colNames[colName]
	if !found {
		return 0
	}
	return stmt.ColumnLen(col)
}

var (
	sqliteInit sync.Once
)

func sqliteInitFn() {
	if Logger != nil {
		C.enable_logging()
	}
}

//export log_fn
func log_fn(_ unsafe.Pointer, code C.int, msg *C.char) {
	var msgBytes []byte
	if msg != nil {
		str := C.GoString(msg) // TODO: do not copy msg.
		msgBytes = []byte(str)
	}
	Logger(ErrorCode(code), msgBytes)
}

// Logger is written to by SQLite.
// The Logger must be set before any connection is opened.
// The msg slice is only valid for the duration of the call.
var Logger func(code ErrorCode, msg []byte)
