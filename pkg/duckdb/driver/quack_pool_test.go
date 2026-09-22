package driver

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// openPooledForTest opens a QuackConns > 1 handle and returns it with its
// connector, so tests can reach the supervised process.
func openPooledForTest(t *testing.T, conns int) (*Connector, *sql.DB) {
	t.Helper()
	binPath := RequireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	addr := EphemeralQuackAddr
	c, db, err := OpenConnector(t.Context(), Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		QuackAddr:  &addr,
		QuackConns: conns,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return c, db
}

func killSubprocess(t *testing.T, p *process) int {
	t.Helper()
	p.mu.Lock()
	cmd := p.cmd
	p.mu.Unlock()
	pid := cmd.Process.Pid
	require.NoError(t, cmd.Process.Kill())
	_, _ = cmd.Process.Wait()
	return pid
}

func currentPid(p *process) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cmd.Process.Pid
}

// TestPooledQuackConnsRecoverFromSubprocessRestart pins the fix for pooled
// connections snapshotting the listener URL/token: after the subprocess is
// killed, every pinned pooled connection must keep working against the
// respawned listener, and the crash must cause exactly one restart no
// matter how many connections observe it.
func TestPooledQuackConnsRecoverFromSubprocessRestart(t *testing.T) {
	c, db := openPooledForTest(t, 4)
	ctx := t.Context()

	conns := make([]*sql.Conn, 3)
	for i := range conns {
		conn, err := db.Conn(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })
		_, err = conn.ExecContext(ctx, "SELECT 1;")
		require.NoError(t, err)
		conns[i] = conn
	}

	c.proc.mu.Lock()
	genBefore := c.proc.quackGen
	c.proc.mu.Unlock()
	pidBefore := killSubprocess(t, c.proc)

	var wg sync.WaitGroup
	errs := make([]error, len(conns))
	for i, conn := range conns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var one int
			errs[i] = conn.QueryRowContext(ctx, "SELECT 1 AS one;").Scan(&one)
		}()
	}
	wg.Wait()
	for i, err := range errs {
		require.NoError(t, err, "pooled conn %d must recover after the subprocess restart", i)
	}

	c.proc.mu.Lock()
	genAfter, disabled := c.proc.quackGen, c.proc.disabled
	c.proc.mu.Unlock()
	require.False(t, disabled)
	require.NotEqual(t, pidBefore, currentPid(c.proc))
	require.Equal(t, genBefore+1, genAfter, "concurrent failures from one crash must restart the subprocess exactly once")
}

// TestPooledQuackConnRehandshakesAfterRestartElsewhere covers a pooled
// connection that was idle while some other connection restarted the
// subprocess: its first statement afterwards must go to the new listener
// without triggering a second restart.
func TestPooledQuackConnRehandshakesAfterRestartElsewhere(t *testing.T) {
	c, db := openPooledForTest(t, 4)
	ctx := t.Context()

	idle, err := db.Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = idle.Close() })
	_, err = idle.ExecContext(ctx, "SELECT 1;")
	require.NoError(t, err)

	c.proc.mu.Lock()
	genBefore := c.proc.quackGen
	c.proc.mu.Unlock()

	killSubprocess(t, c.proc)
	// The primary (restart-managed) path notices the crash and restarts.
	_, _, err = c.proc.exec(ctx, "SELECT 1;")
	require.NoError(t, err)

	var one int
	require.NoError(t, idle.QueryRowContext(ctx, "SELECT 1 AS one;").Scan(&one))
	require.Equal(t, 1, one)

	c.proc.mu.Lock()
	genAfter := c.proc.quackGen
	c.proc.mu.Unlock()
	require.Equal(t, genBefore+1, genAfter, "a stale pooled session must re-handshake, not restart the subprocess again")
}

// TestPooledQuackConnSurfacesStatementErrorsWithoutRestart pins that a
// statement DuckDB rejected is not mistaken for a transport failure.
func TestPooledQuackConnSurfacesStatementErrorsWithoutRestart(t *testing.T) {
	c, db := openPooledForTest(t, 4)
	ctx := t.Context()

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	pid := currentPid(c.proc)
	_, err = conn.ExecContext(ctx, "SELECT * FROM table_that_does_not_exist;")
	require.Error(t, err)
	require.Equal(t, pid, currentPid(c.proc))
}

// TestPooledQuackConnAfterCloseFailsFast pins that a pooled connection
// doesn't hang or dial a dead listener once the connector is closed.
func TestPooledQuackConnAfterCloseFailsFast(t *testing.T) {
	c, _ := openPooledForTest(t, 4)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	sess, err := c.proc.openQuackConn(ctx)
	require.NoError(t, err)
	require.NoError(t, c.Close())

	_, _, err = sess.exec(ctx, "SELECT 1;")
	require.ErrorIs(t, err, errQuackUnavailable)
}
