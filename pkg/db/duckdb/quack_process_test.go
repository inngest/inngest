package duckdb

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

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

// TestQuackCancelledQueryDoesNotRestartSubprocess pins the fix for a real
// bug: cancelling a caller's ctx mid-query over quack used to be
// indistinguishable, from process.exec/query's point of view, from the
// subprocess having crashed -- runWithRestartLocked treated both the same
// way and paid for a full respawn every time, disrupting every other
// caller sharing this same connection (e.g. dual-write's own writes,
// serialized through the identical mutex) even though nothing was
// actually wrong with the subprocess. Unlike jsonlines (whose single
// shared stdin/stdout stream really is left desynced by an abandoned
// statement -- see TestSessionExecCancelMidStatementDesyncsAndProcessResyncs),
// quack's HTTP transport is a self-contained request per statement: an
// aborted request doesn't corrupt anything for the next one, verified
// empirically against the real quack extension (a fresh query on the same
// connection succeeds in ~1ms after an abort). So a cancelled quack query
// must now surface its ctx error without ever touching the subprocess.
//
// It also asserts the query is genuinely interrupted server-side, not just
// abandoned: quackSession.query's watchForCancel sends a real CancelRequest
// (encodeQuackCancelRequest), which quack_server.cpp turns into a DuckDB
// Connection::Interrupt() -- confirmed here by watching the subprocess's own
// CPU time via ps plateau almost immediately after cancel, rather than
// continuing to climb for the ~8s the query would otherwise take to finish
// on its own.
func TestQuackCancelledQueryDoesNotRestartSubprocess(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	addr := freeLocalAddr(t)
	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	_, _, err = p.exec(t.Context(), "SET threads=1;")
	require.NoError(t, err)

	pidBefore := p.cmd.Process.Pid

	// A genuinely slow, single-threaded query -- confirmed empirically to
	// take several seconds -- so cancelling it below lands mid-flight
	// rather than racing a query that already finished.
	slowSQL := "SELECT count(*) FROM (SELECT md5(a::VARCHAR || b::VARCHAR) FROM range(50000000) a, range(2000) b);"

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		_, _, _, err := p.query(ctx, slowSQL)
		done <- err
	}()

	time.Sleep(1500 * time.Millisecond)
	cpuAtCancel := processCPUSeconds(t, pidBefore)
	cancel()

	select {
	case err := <-done:
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(10 * time.Second):
		t.Fatal("cancelled query never returned")
	}

	p.mu.Lock()
	pidAfter := p.cmd.Process.Pid
	disabled := p.disabled
	p.mu.Unlock()
	require.Equal(t, pidBefore, pidAfter, "a cancelled query must not respawn the subprocess")
	require.False(t, disabled)

	// The connection (and process.exec's mutex-guarded path a write like
	// dual-write's would use) must still be immediately usable -- not
	// blocked behind the abandoned query.
	_, rows, err := p.exec(t.Context(), "SELECT 1 AS ok;")
	require.NoError(t, err)
	require.Len(t, rows, 1)

	// Give the CancelRequest watchForCancel fired a moment to reach the
	// server and for Interrupt() to actually take effect, then confirm CPU
	// usage plateaued rather than continuing to accumulate at ~1
	// core-second per wall-second (which single-threaded execution of
	// slowSQL would otherwise do for several more seconds).
	time.Sleep(2 * time.Second)
	delta := processCPUSeconds(t, pidAfter) - cpuAtCancel
	require.Less(t, delta, 1.5,
		"subprocess CPU time kept climbing after cancel -- the query does not appear to have been interrupted server-side (delta=%.2fs)", delta)
}

// processCPUSeconds shells out to ps to read pid's accumulated CPU time (the
// TIME column, HH:MM:SS or MM:SS) -- a crude but real signal of whether it's
// still actively burning CPU.
func processCPUSeconds(t *testing.T, pid int) float64 {
	t.Helper()
	out, err := exec.Command("ps", "-o", "time=", "-p", strconv.Itoa(pid)).Output()
	require.NoError(t, err)
	parts := strings.Split(strings.TrimSpace(string(out)), ":")
	var secs, mult float64 = 0, 1
	for i := len(parts) - 1; i >= 0; i-- {
		v, _ := strconv.ParseFloat(parts[i], 64)
		secs += v * mult
		mult *= 60
	}
	return secs
}
