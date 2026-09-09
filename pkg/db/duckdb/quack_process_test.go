package duckdb

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// freeLocalAddr resolves an ephemeral local port, closing the listener
// immediately so quack_serve can bind it. Small TOCTOU race, acceptable for
// tests (same tradeoff quack_session_test.go makes).
func freeLocalAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return fmt.Sprintf("127.0.0.1:%d", port)
}

func TestStartProcessWithQuackTransport(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	addr := freeLocalAddr(t)
	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	_, isQuack := p.sess.(*quackSession)
	require.True(t, isQuack, "process should have swapped its active session to the quack transport once bootstrapped")

	require.NoError(t, p.healthCheck(t.Context()))

	_, _, err = p.exec(t.Context(), "CREATE TABLE t (id INTEGER, name VARCHAR);")
	require.NoError(t, err)
	_, _, err = p.exec(t.Context(), "INSERT INTO t VALUES (1, 'a');")
	require.NoError(t, err)

	_, rows, err := p.exec(t.Context(), "SELECT id, name FROM t;")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(1), rows[0]["id"])
	require.Equal(t, "a", rows[0]["name"])
}

func TestQuackTransportSurvivesRestart(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	addr := freeLocalAddr(t)
	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	_, _, err = p.exec(t.Context(), "CREATE TABLE t (id INTEGER);")
	require.NoError(t, err)
	_, _, err = p.exec(t.Context(), "INSERT INTO t VALUES (1);")
	require.NoError(t, err)

	pidBefore := p.cmd.Process.Pid
	require.NoError(t, p.cmd.Process.Kill())
	_, _ = p.cmd.Process.Wait()

	// :memory: state (including the CREATE TABLE above) does not survive a
	// real crash — same as the jsonlines path — so the pre-crash table is
	// gone. What this proves is that the restart's fresh spawn+bootstrap
	// stands up a *new* quack listener and the process keeps working over
	// it, rather than hanging or falling back to some other transport.
	_, _, err = p.exec(t.Context(), "CREATE TABLE t (id INTEGER);")
	require.NoError(t, err, "restart must re-bootstrap the quack transport, not just the jsonlines control channel")
	_, _, err = p.exec(t.Context(), "INSERT INTO t VALUES (1);")
	require.NoError(t, err)
	_, rows, err := p.exec(t.Context(), "SELECT count(*) AS c FROM t;")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(1), rows[0]["c"])

	p.mu.Lock()
	disabled := p.disabled
	pidAfter := p.cmd.Process.Pid
	_, stillQuack := p.sess.(*quackSession)
	p.mu.Unlock()
	require.False(t, disabled)
	require.NotEqual(t, pidBefore, pidAfter)
	require.True(t, stillQuack, "the restarted process must still be using the quack transport, not have fallen back to jsonlines")
}

