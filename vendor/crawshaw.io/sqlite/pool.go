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

// Pool is a pool of SQLite connections.
//
// It is safe for use by multiple goroutines concurrently.
//
// Typically, a goroutine that needs to use an SQLite *Conn
// Gets it from the pool and defers its return:
//
//	conn := dbpool.Get(nil)
//	defer dbpool.Put(conn)
//
// As Get may block, a context can be used to return if a task
// is cancelled. In this case the Conn returned will be nil:
//
//	conn := dbpool.Get(ctx.Done())
//	if conn == nil {
//		return context.Canceled
//	}
//	defer dbpool.Put(conn)
type Pool struct {
	// If checkReset, the Put method checks all of the connection's
	// prepared statements and ensures they were correctly cleaned up.
	// If they were not, Put will panic with details.
	//
	// TODO: export this? Is it enough of a performance concern?
	checkReset bool

	free   chan *Conn
	all    []*Conn
	closed chan struct{}
}

// Open opens a fixed-size pool of SQLite connections.
// A flags value of 0 defaults to:
//
//	SQLITE_OPEN_READWRITE
//	SQLITE_OPEN_CREATE
//	SQLITE_OPEN_SHAREDCACHE
//	SQLITE_OPEN_WAL
//	SQLITE_OPEN_URI
//	SQLITE_OPEN_NOMUTEX
//
// The pool is always created with the shared cache enabled.
func Open(uri string, flags OpenFlags, poolSize int) (*Pool, error) {
	if flags == 0 {
		flags = SQLITE_OPEN_READWRITE | SQLITE_OPEN_CREATE | SQLITE_OPEN_WAL | SQLITE_OPEN_URI | SQLITE_OPEN_NOMUTEX
	}
	flags |= SQLITE_OPEN_SHAREDCACHE
	if uri == ":memory:" {
		return nil, strerror{msg: `sqlite: ":memory:" does not work with multiple connections, use "file::memory:?mode=memory"`}
	}

	p := &Pool{
		checkReset: true,
		free:       make(chan *Conn, poolSize),
		closed:     make(chan struct{}),
	}

	for i := 0; i < poolSize; i++ {
		conn, err := openConn(uri, flags)
		if err != nil {
			p.Close()
			return nil, err
		}
		p.free <- conn
		p.all = append(p.all, conn)
	}

	return p, nil
}

// Get gets an SQLite connection from the pool.
//
// If no Conn is available, Get will block until one is,
// or until either the Pool is closed or doneCh is closed,
// in which case Get returns nil.
//
// The provided doneCh is used to control the execution
// lifetime of the connection. See Conn.SetInterrupt for
// details.
func (p *Pool) Get(doneCh <-chan struct{}) *Conn {
	select {
	case conn, ok := <-p.free:
		if !ok {
			return nil // pool is closed
		}
		conn.SetInterrupt(doneCh)
		return conn
	case <-doneCh:
		return nil
	case <-p.closed:
		return nil
	}
}

// Put puts an SQLite connection back into the Pool.
// A nil conn will cause Put to panic.
func (p *Pool) Put(conn *Conn) {
	if conn == nil {
		panic("attempted to Put a nil Conn into Pool")
	}
	if p.checkReset {
		for _, stmt := range conn.stmts {
			if stmt.lastHasRow {
				panic("connection returned to pool has active statement: \"" + stmt.query + "\"")
			}
		}
	}

	conn.SetInterrupt(nil)
	select {
	case p.free <- conn:
	default:
		panic("no space in Pool; Get/Put mismatch")
	}
}

// Close closes all the connections in the Pool.
func (p *Pool) Close() (err error) {
	close(p.closed)
	for _, conn := range p.all {
		err2 := conn.Close()
		if err == nil {
			err = err2
		}
	}
	close(p.free)
	for range p.free {
	}
	return err
}

type strerror struct {
	msg string
}

func (err strerror) Error() string { return err.msg }
