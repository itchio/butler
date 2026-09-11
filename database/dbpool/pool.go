// Package dbpool is a lazily-populated pool of SQLite connections.
//
// It has the same Get/Put/Close contract as crawshaw.io/sqlite/sqlitex.Pool,
// but connections are opened on demand and closed again when idle, so a
// generous concurrency cap costs nothing while the daemon is quiet.
package dbpool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"crawshaw.io/sqlite"
)

// Options tunes a Pool. The zero value is usable.
type Options struct {
	// Flags are passed to sqlite.OpenConn. 0 selects the driver defaults
	// (READWRITE | CREATE | WAL | URI | NOMUTEX).
	Flags sqlite.OpenFlags

	// MaxConns is the number of connections that may be checked out at once.
	// Get blocks when they are all in use. Defaults to 8.
	MaxConns int

	// MinIdle connections are kept open even when idle. Defaults to 1.
	MinIdle int

	// IdleTimeout is how long an unused connection above MinIdle survives
	// before being closed. Defaults to 1 minute; a negative value disables
	// reaping.
	IdleTimeout time.Duration

	// OnOpenError receives the error when an on-demand open fails and Get
	// returns nil. Get has no error return, to stay compatible with sqlitex.
	OnOpenError func(err error)
}

// CloseTimeout bounds how long Close waits for checked-out connections to
// be returned before panicking, mirroring sqlitex.PoolCloseTimeout.
var CloseTimeout = 5 * time.Second

type idleConn struct {
	conn     *sqlite.Conn
	idleFrom time.Time
}

// Pool is safe for concurrent use.
type Pool struct {
	uri  string
	opts Options

	// slots holds one token per connection that may still be checked out.
	// Acquiring a token is the only operation that blocks, so context and
	// close handling live there.
	slots  chan struct{}
	closed chan struct{}

	// lifetime is cancelled by Close. Every borrowed connection's interrupt
	// context descends from it, so Close reaches borrowers that were still
	// acquiring when it ran and had not yet been registered in inUse.
	lifetime context.Context
	endLife  context.CancelFunc

	// openSem serializes connection creation. The driver runs PRAGMA
	// journal_mode=wal inside every open, and concurrent first opens on a
	// fresh file collide during WAL recovery with SQLITE_BUSY. It is a
	// one-slot channel rather than a mutex so waiting on it can observe the
	// caller's context and pool shutdown, and it is separate from mu so Put
	// and the reaper never wait on a slow open.
	openSem chan struct{}

	mu       sync.Mutex
	idle     []idleConn // most recently used last
	inUse    map[*sqlite.Conn]context.CancelFunc
	reaperWG sync.WaitGroup
}

// Open validates that the database can be opened, then returns a pool that
// opens further connections as they are needed.
func Open(uri string, opts Options) (*Pool, error) {
	if uri == ":memory:" {
		return nil, fmt.Errorf(`dbpool: ":memory:" does not work with multiple connections, use "file::memory:?mode=memory"`)
	}
	if opts.MaxConns <= 0 {
		opts.MaxConns = 8
	}
	if opts.MinIdle <= 0 {
		opts.MinIdle = 1
	}
	if opts.MinIdle > opts.MaxConns {
		opts.MinIdle = opts.MaxConns
	}
	if opts.IdleTimeout == 0 {
		opts.IdleTimeout = time.Minute
	}

	lifetime, endLife := context.WithCancel(context.Background())
	p := &Pool{
		uri:      uri,
		opts:     opts,
		slots:    make(chan struct{}, opts.MaxConns),
		openSem:  make(chan struct{}, 1),
		closed:   make(chan struct{}),
		lifetime: lifetime,
		endLife:  endLife,
		inUse:    make(map[*sqlite.Conn]context.CancelFunc),
	}
	for i := 0; i < opts.MaxConns; i++ {
		p.slots <- struct{}{}
	}

	// Open one connection eagerly so that a bad path or a corrupt file is
	// reported here rather than from the first request.
	conn, err := p.open(context.Background())
	if err != nil {
		return nil, err
	}
	p.idle = append(p.idle, idleConn{conn: conn, idleFrom: time.Now()})

	if opts.IdleTimeout > 0 {
		p.reaperWG.Add(1)
		go p.reap()
	}
	return p, nil
}

// open creates a connection, or returns ctx.Err() if the context expires or
// the pool closes while waiting for a turn to open.
func (p *Pool) open(ctx context.Context) (*sqlite.Conn, error) {
	select {
	case p.openSem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.closed:
		return nil, context.Canceled
	}
	defer func() { <-p.openSem }()
	return sqlite.OpenConn(p.uri, p.opts.Flags)
}

// Get returns a connection, blocking while all MaxConns are checked out.
// It returns nil if the context expires or the pool is closed first.
// The context also interrupts long-running queries on the returned
// connection, see sqlite.Conn.SetInterrupt.
func (p *Pool) Get(ctx context.Context) *sqlite.Conn {
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-p.slots:
	case <-ctx.Done():
		return nil
	case <-p.closed:
		return nil
	}

	// select picks randomly among ready cases, so re-check closed now that
	// we hold a slot. Close drains every slot before it finishes, so a
	// borrower that gets past this point is guaranteed to be waited for.
	select {
	case <-p.closed:
		p.slots <- struct{}{}
		return nil
	default:
	}

	p.mu.Lock()
	var conn *sqlite.Conn
	if n := len(p.idle); n > 0 {
		conn = p.idle[n-1].conn
		p.idle = p.idle[:n-1]
	}
	p.mu.Unlock()

	if conn == nil {
		var err error
		conn, err = p.open(ctx)
		if err != nil {
			p.slots <- struct{}{}
			// a cancelled wait is the caller's doing, not a database problem
			if p.opts.OnOpenError != nil && ctx.Err() == nil {
				p.opts.OnOpenError(err)
			}
			return nil
		}
	}

	// The interrupt fires on the caller's context or on Close, whichever
	// comes first. If Close already ran, AfterFunc fires immediately and
	// the borrower gets an interrupted connection rather than a usable one.
	ctx, cancel := context.WithCancel(ctx)
	stopAfter := context.AfterFunc(p.lifetime, cancel)
	conn.SetInterrupt(ctx.Done())

	p.mu.Lock()
	p.inUse[conn] = func() {
		stopAfter()
		cancel()
	}
	p.mu.Unlock()
	return conn
}

// Put returns a connection obtained from Get. It panics if the connection
// did not come from this pool, or still has a statement mid-execution.
func (p *Pool) Put(conn *sqlite.Conn) {
	if conn == nil {
		panic("dbpool: Put of a nil Conn")
	}
	if query := conn.CheckReset(); query != "" {
		panic(fmt.Sprintf("dbpool: connection returned with active statement: %q", query))
	}

	p.mu.Lock()
	cancel, found := p.inUse[conn]
	delete(p.inUse, conn)
	p.mu.Unlock()
	if !found {
		panic("dbpool: Put of a connection not created by this pool")
	}

	// Finish with the connection while we still own it exclusively. Only
	// then publish it to idle: a Get that already holds a slot may take it
	// the instant it appears there.
	cancel()
	conn.SetInterrupt(nil)

	p.mu.Lock()
	p.idle = append(p.idle, idleConn{conn: conn, idleFrom: time.Now()})
	p.mu.Unlock()
	p.slots <- struct{}{}
}

func (p *Pool) reap() {
	defer p.reaperWG.Done()
	interval := p.opts.IdleTimeout / 2
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, conn := range p.takeExpired(time.Now()) {
				conn.Close()
			}
		case <-p.closed:
			return
		}
	}
}

// takeExpired removes and returns idle connections beyond MinIdle that have
// been unused for longer than IdleTimeout. Oldest first, so the survivors
// are the most recently used.
func (p *Pool) takeExpired(now time.Time) []*sqlite.Conn {
	p.mu.Lock()
	defer p.mu.Unlock()

	var expired []*sqlite.Conn
	for len(p.idle) > p.opts.MinIdle && now.Sub(p.idle[0].idleFrom) >= p.opts.IdleTimeout {
		expired = append(expired, p.idle[0].conn)
		p.idle = p.idle[1:]
	}
	return expired
}

// Stats reports how many connections are currently open and how many of
// those are checked out.
func (p *Pool) Stats() (open int, inUse int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.idle) + len(p.inUse), len(p.inUse)
}

// Close interrupts checked-out connections, waits for them to be returned,
// and closes everything. It panics if callers hold on to connections for
// longer than CloseTimeout, since that is a leak.
func (p *Pool) Close() (err error) {
	close(p.closed)
	p.reaperWG.Wait()

	// Interrupt every borrower, including any still inside Get.
	p.endLife()

	// Reclaiming every slot proves every connection has been Put back.
	timeout := time.After(CloseTimeout)
	for i := 0; i < p.opts.MaxConns; i++ {
		select {
		case <-p.slots:
		case <-timeout:
			panic("dbpool: not all connections returned to pool before timeout")
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	for _, ic := range p.idle {
		if cerr := ic.conn.Close(); err == nil {
			err = cerr
		}
	}
	p.idle = nil
	return err
}
