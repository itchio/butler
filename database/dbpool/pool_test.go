package dbpool

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestPool(t *testing.T, opts Options) *Pool {
	t.Helper()
	p, err := Open(filepath.Join(t.TempDir(), "test.db"), opts)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, p.Close()) })
	return p
}

func Test_OpensLazily(t *testing.T) {
	p := openTestPool(t, Options{MaxConns: 10, IdleTimeout: -1})

	open, inUse := p.Stats()
	assert.Equal(t, 1, open, "only the probe connection is opened up front")
	assert.Equal(t, 0, inUse)

	for i := 0; i < 5; i++ {
		conn := p.Get(context.Background())
		require.NotNil(t, conn)
		defer p.Put(conn)
	}
	open, inUse = p.Stats()
	assert.Equal(t, 5, open)
	assert.Equal(t, 5, inUse)
}

func Test_ConnectionsWork(t *testing.T) {
	p := openTestPool(t, Options{IdleTimeout: -1})

	conn := p.Get(context.Background())
	require.NotNil(t, conn)
	require.NoError(t, sqlitex.ExecScript(conn, `CREATE TABLE t (v TEXT); INSERT INTO t VALUES ('hello');`))
	p.Put(conn)

	// a fresh connection sees the committed write
	conn = p.Get(context.Background())
	require.NotNil(t, conn)
	defer p.Put(conn)
	var got string
	require.NoError(t, sqlitex.Exec(conn, `SELECT v FROM t`, func(stmt *sqlite.Stmt) error {
		got = stmt.ColumnText(0)
		return nil
	}))
	assert.Equal(t, "hello", got)
}

func Test_BlocksAtMaxAndHonorsContext(t *testing.T) {
	p := openTestPool(t, Options{MaxConns: 2, IdleTimeout: -1})

	a := p.Get(context.Background())
	b := p.Get(context.Background())
	require.NotNil(t, a)
	require.NotNil(t, b)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	assert.Nil(t, p.Get(ctx), "third Get must give up when the context expires")
	assert.GreaterOrEqual(t, time.Since(start), 50*time.Millisecond)

	// returning one unblocks a waiter
	done := make(chan struct{})
	go func() {
		defer close(done)
		c := p.Get(context.Background())
		assert.NotNil(t, c)
		p.Put(c)
	}()
	p.Put(a)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("waiter never got the returned connection")
	}
	p.Put(b)
}

func Test_ReapsIdleDownToMinIdle(t *testing.T) {
	p := openTestPool(t, Options{MaxConns: 8, MinIdle: 2, IdleTimeout: time.Hour})

	var conns []*sqlite.Conn
	for i := 0; i < 6; i++ {
		c := p.Get(context.Background())
		require.NotNil(t, c)
		conns = append(conns, c)
	}
	for _, c := range conns {
		p.Put(c)
	}
	open, _ := p.Stats()
	assert.Equal(t, 6, open)

	// nothing has been idle for an hour yet
	assert.Empty(t, p.takeExpired(time.Now()))

	expired := p.takeExpired(time.Now().Add(2 * time.Hour))
	assert.Len(t, expired, 4)
	for _, c := range expired {
		assert.NoError(t, c.Close())
	}
	open, _ = p.Stats()
	assert.Equal(t, 2, open, "MinIdle survivors")

	// survivors are still usable
	c := p.Get(context.Background())
	require.NotNil(t, c)
	require.NoError(t, sqlitex.Exec(c, `SELECT 1`, nil))
	p.Put(c)
}

func Test_ReaperRuns(t *testing.T) {
	p := openTestPool(t, Options{MaxConns: 4, MinIdle: 1, IdleTimeout: 20 * time.Millisecond})

	var conns []*sqlite.Conn
	for i := 0; i < 4; i++ {
		conns = append(conns, p.Get(context.Background()))
	}
	for _, c := range conns {
		p.Put(c)
	}

	assert.Eventually(t, func() bool {
		open, _ := p.Stats()
		return open == 1
	}, 5*time.Second, 5*time.Millisecond)
}

func Test_GetInterruptsOnContextCancel(t *testing.T) {
	p := openTestPool(t, Options{IdleTimeout: -1})

	ctx, cancel := context.WithCancel(context.Background())
	conn := p.Get(ctx)
	require.NotNil(t, conn)
	cancel()
	err := sqlitex.Exec(conn, `SELECT 1`, nil)
	assert.Error(t, err, "query on a cancelled context is interrupted")
	p.Put(conn)

	// the connection is usable again once re-acquired
	conn = p.Get(context.Background())
	require.NotNil(t, conn)
	assert.NoError(t, sqlitex.Exec(conn, `SELECT 1`, nil))
	p.Put(conn)
}

func Test_RejectsForeignConn(t *testing.T) {
	p := openTestPool(t, Options{IdleTimeout: -1})
	other, err := sqlite.OpenConn("file::memory:?mode=memory", 0)
	require.NoError(t, err)
	defer other.Close()
	assert.Panics(t, func() { p.Put(other) })
}

func Test_OpenFailsOnBadPath(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "missing", "dir", "test.db"), Options{})
	assert.Error(t, err)
}

// Many goroutines churning Get/Put on a pool smaller than the goroutine
// count. Under -race this catches a connection being handed to a new
// borrower before the previous one has finished with it.
func Test_ConcurrentChurn(t *testing.T) {
	p := openTestPool(t, Options{MaxConns: 4, MinIdle: 1, IdleTimeout: 10 * time.Millisecond})

	conn := p.Get(context.Background())
	require.NotNil(t, conn)
	require.NoError(t, sqlitex.ExecScript(conn, `CREATE TABLE t (v INTEGER);`))
	p.Put(conn)

	const workers = 16
	const iterations = 200
	errs := make(chan error, workers)
	for w := 0; w < workers; w++ {
		go func(w int) {
			for i := 0; i < iterations; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				c := p.Get(ctx)
				if c == nil {
					cancel()
					errs <- fmt.Errorf("worker %d: Get returned nil", w)
					return
				}
				err := sqlitex.Exec(c, `INSERT INTO t VALUES (?)`, nil, w)
				p.Put(c)
				cancel()
				if err != nil {
					errs <- fmt.Errorf("worker %d: %w", w, err)
					return
				}
			}
			errs <- nil
		}(w)
	}
	for w := 0; w < workers; w++ {
		select {
		case err := <-errs:
			require.NoError(t, err)
		case <-time.After(30 * time.Second):
			t.Fatal("churn deadlocked")
		}
	}

	c := p.Get(context.Background())
	require.NotNil(t, c)
	var n int
	require.NoError(t, sqlitex.Exec(c, `SELECT COUNT(*) FROM t`, func(stmt *sqlite.Stmt) error {
		n = stmt.ColumnInt(0)
		return nil
	}))
	p.Put(c)
	assert.Equal(t, workers*iterations, n)
}

// Close must interrupt a borrower that is mid-query so the connection comes
// back before the close timeout, rather than panicking.
func Test_CloseInterruptsActiveBorrower(t *testing.T) {
	p, err := Open(filepath.Join(t.TempDir(), "test.db"), Options{MaxConns: 2, IdleTimeout: -1})
	require.NoError(t, err)

	conn := p.Get(context.Background())
	require.NotNil(t, conn)
	require.NoError(t, sqlitex.ExecScript(conn, `CREATE TABLE t (v INTEGER); INSERT INTO t VALUES (1);`))
	p.Put(conn)

	started := make(chan struct{})
	returned := make(chan error, 1)
	go func() {
		c := p.Get(context.Background())
		if c == nil {
			returned <- fmt.Errorf("Get returned nil")
			return
		}
		close(started)
		// a query that runs until interrupted
		err := sqlitex.Exec(c, `WITH RECURSIVE r(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM r) SELECT COUNT(*) FROM r`, nil)
		p.Put(c)
		returned <- err
	}()
	<-started
	time.Sleep(20 * time.Millisecond)

	closeDone := make(chan error, 1)
	go func() { closeDone <- p.Close() }()

	select {
	case err := <-returned:
		assert.Error(t, err, "the long query should have been interrupted")
	case <-time.After(CloseTimeout):
		t.Fatal("borrower was never interrupted")
	}
	select {
	case err := <-closeDone:
		assert.NoError(t, err)
	case <-time.After(CloseTimeout):
		t.Fatal("Close did not finish")
	}
}

// Hammer Get while Close runs. Every Get either returns nil or a connection
// that is returned, Close must not panic, and no connection may leak past
// Close (the driver's finalizer panics on a leaked connection).
func Test_CloseRacesWithGet(t *testing.T) {
	for round := 0; round < 20; round++ {
		p, err := Open(filepath.Join(t.TempDir(), fmt.Sprintf("test%d.db", round)), Options{MaxConns: 8, IdleTimeout: -1})
		require.NoError(t, err)

		var wg sync.WaitGroup
		stop := make(chan struct{})
		for w := 0; w < 8; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-stop:
						return
					default:
					}
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					c := p.Get(ctx)
					if c != nil {
						_ = sqlitex.Exec(c, `SELECT 1`, nil)
						p.Put(c)
					}
					cancel()
				}
			}()
		}
		time.Sleep(time.Duration(round) * time.Millisecond)
		assert.NotPanics(t, func() { assert.NoError(t, p.Close()) })
		close(stop)
		wg.Wait()
	}
	runtime.GC()
}

// A burst of Gets on a fresh pool opens many connections at once. Concurrent
// opens on a new file can trip over each other's WAL setup, which must not
// surface as a spurious nil from Get.
func Test_SimultaneousFirstOpens(t *testing.T) {
	for round := 0; round < 10; round++ {
		p, err := Open(filepath.Join(t.TempDir(), fmt.Sprintf("fresh%d.db", round)), Options{MaxConns: 16, IdleTimeout: -1})
		require.NoError(t, err)

		const n = 16
		start := make(chan struct{})
		results := make(chan *sqlite.Conn, n)
		for i := 0; i < n; i++ {
			go func() {
				<-start
				results <- p.Get(context.Background())
			}()
		}
		close(start)
		var conns []*sqlite.Conn
		for i := 0; i < n; i++ {
			c := <-results
			if assert.NotNil(t, c, "round %d: Get returned nil with capacity available", round) {
				conns = append(conns, c)
			}
		}
		for _, c := range conns {
			p.Put(c)
		}
		require.NoError(t, p.Close())
		if t.Failed() {
			return
		}
	}
}

// A Get waiting for its turn to open a connection must give up when its
// context expires rather than opening after the caller has moved on.
func Test_OpenWaitHonorsContext(t *testing.T) {
	p := openTestPool(t, Options{MaxConns: 4, IdleTimeout: -1})

	// take the probe connection so the next Get has to open one
	held := p.Get(context.Background())
	require.NotNil(t, held)
	defer p.Put(held)

	// occupy the open semaphore as if an open were stuck
	p.openSem <- struct{}{}
	defer func() { <-p.openSem }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	assert.Nil(t, p.Get(ctx))
	assert.Less(t, time.Since(start), time.Second, "should return at the deadline, not block on the open")

	open, _ := p.Stats()
	assert.Equal(t, 1, open, "no connection was opened for the abandoned Get")
}
