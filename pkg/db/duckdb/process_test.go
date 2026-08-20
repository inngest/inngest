package duckdb

import (
	"database/sql"
	"os/exec"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func requireDuckDBBinary(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping subprocess test")
	}
	return path
}

func TestStartProcessHealthCheck(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	p, err := startProcess(t.Context(), binPath, ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	require.NoError(t, p.healthCheck(t.Context()))
}

func TestStartProcessExecAndQuery(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	p, err := startProcess(t.Context(), binPath, ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	_, err = p.sess.exec(t.Context(), "CREATE TABLE t (id INTEGER, name VARCHAR);")
	require.NoError(t, err)

	_, err = p.sess.exec(t.Context(), "INSERT INTO t VALUES (1, 'a');")
	require.NoError(t, err)

	rows, err := p.sess.exec(t.Context(), "SELECT id, name FROM t;")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, float64(1), rows[0]["id"])
	require.Equal(t, "a", rows[0]["name"])
}

func TestProcessRestartAfterCrash(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	p, err := startProcess(t.Context(), binPath, ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	require.NoError(t, p.healthCheck(t.Context()))

	// Simulate a crash: kill the underlying subprocess out from under the
	// session, bypassing the normal close() path.
	require.NoError(t, p.cmd.Process.Kill())
	_, _ = p.cmd.Process.Wait()

	// The stale session should now fail health checks (broken pipe / EOF).
	require.Error(t, p.healthCheck(t.Context()))

	require.NoError(t, p.restart(t.Context()))
	require.NoError(t, p.healthCheck(t.Context()))
}

func TestOpenAndSetMaxOpenConns(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE t2 (id INTEGER);")
	require.NoError(t, err)
}

// TestStartProcessFailsHealthCheckDuringSpawn covers review finding #1:
// startProcess (and therefore Open/Connector.Connect) must not hand back a
// process that hasn't proven it can round-trip a query. /bin/true is a real,
// startable executable that isn't duckdb: it exits immediately, so the
// health check's write/read against its (closed) stdin/stdout fails, and
// startProcess must surface that as an error rather than returning a
// "started but unusable" process.
func TestStartProcessFailsHealthCheckDuringSpawn(t *testing.T) {
	notDuckDB, err := exec.LookPath("true")
	if err != nil {
		t.Skip("no /bin/true-equivalent on PATH to simulate an unhealthy binary")
	}

	p, err := startProcess(t.Context(), notDuckDB, ":memory:")
	require.Error(t, err)
	require.Nil(t, p)
}

// TestOpenFailsWhenBinaryUnhealthy covers the same finding through the
// public Open API, confirming a caller never receives a *sql.DB backed by an
// unhealthy subprocess.
func TestOpenFailsWhenBinaryUnhealthy(t *testing.T) {
	notDuckDB, err := exec.LookPath("true")
	if err != nil {
		t.Skip("no /bin/true-equivalent on PATH to simulate an unhealthy binary")
	}

	db, err := Open(t.Context(), Options{BinaryPath: notDuckDB, DBFile: ":memory:"})
	require.Error(t, err)
	require.Nil(t, db)
}

// TestConnectorCloseTerminatesSubprocess covers review finding #2:
// *sql.DB.Close() must actually terminate the real duckdb subprocess, not
// just drop the Go-side reference to it. database/sql only calls
// Connector.Close (io.Closer) if the connector implements it, so this
// exercises that wiring end-to-end through the public API.
func TestConnectorCloseTerminatesSubprocess(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	c := &Connector{opts: Options{BinaryPath: binPath, DBFile: ":memory:"}}
	db := sql.OpenDB(c)
	db.SetMaxOpenConns(1)

	_, err := db.ExecContext(t.Context(), "SELECT 1;")
	require.NoError(t, err)
	require.NotNil(t, c.proc, "Connect should have spawned the subprocess")

	require.NoError(t, db.Close())

	// close() blocks (with a kill fallback) until cmd.Wait() returns, so by
	// the time db.Close() returns, ProcessState must be populated and report
	// the process as exited — not merely "we stopped watching it".
	require.NotNil(t, c.proc.cmd.ProcessState, "subprocess should have been waited on by db.Close()")
	require.True(t, c.proc.cmd.ProcessState.Exited(), "subprocess should have exited after db.Close()")
}

// TestExecTriggersRestartAfterCrash covers the success path of review
// finding #3: a session-level error (simulated here with a real kill, not a
// mock) detected inside conn.ExecContext -> process.exec must trigger one
// restart transparently and retry, all reachable through the public
// database/sql API rather than by calling the unexported restart() method
// directly.
func TestExecTriggersRestartAfterCrash(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	c := &Connector{opts: Options{BinaryPath: binPath, DBFile: ":memory:"}}
	db := sql.OpenDB(c)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	_, err := db.ExecContext(t.Context(), "CREATE TABLE t (id INTEGER);")
	require.NoError(t, err)
	require.NotNil(t, c.proc)

	// Simulate a crash out from under the session, bypassing close().
	require.NoError(t, c.proc.cmd.Process.Kill())
	_, _ = c.proc.cmd.Process.Wait()

	// This call goes through database/sql -> conn.ExecContext ->
	// process.exec, which must detect the dead session, restart, and retry —
	// transparently, from the caller's perspective.
	_, err = db.ExecContext(t.Context(), "SELECT 1;")
	require.NoError(t, err)

	c.proc.mu.Lock()
	disabled := c.proc.disabled
	c.proc.mu.Unlock()
	require.False(t, disabled, "a successful restart must not disable the process")
}

// TestExecPermanentlyDisablesAfterFailedRestart covers the failure path of
// review finding #3: if the one restart attempt also fails, the process
// must be permanently disabled so further calls fail fast with
// errProcessDisabled rather than repeatedly retrying a respawn that cannot
// succeed. The restart is made to fail deterministically by corrupting the
// binary path after the initial (successful) connect — simulating an
// environment where the duckdb binary becomes unavailable — then crashing
// the subprocess.
func TestExecPermanentlyDisablesAfterFailedRestart(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	c := &Connector{opts: Options{BinaryPath: binPath, DBFile: ":memory:"}}
	db := sql.OpenDB(c)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	_, err := db.ExecContext(t.Context(), "SELECT 1;")
	require.NoError(t, err)
	require.NotNil(t, c.proc)

	c.proc.mu.Lock()
	c.proc.binaryPath = "/nonexistent/path/to/duckdb"
	c.proc.mu.Unlock()

	require.NoError(t, c.proc.cmd.Process.Kill())
	_, _ = c.proc.cmd.Process.Wait()

	// First call after the crash: exec detects the dead session, attempts a
	// restart, and the restart fails because the binary path is now bogus.
	_, err = db.ExecContext(t.Context(), "SELECT 1;")
	require.Error(t, err)

	// Every subsequent call must fail fast with errProcessDisabled instead
	// of attempting another (still-doomed) restart.
	_, err = db.ExecContext(t.Context(), "SELECT 1;")
	require.ErrorIs(t, err, errProcessDisabled)
}

// TestProcessHealthCheckAndRestartAreRaceFree covers review finding #4
// directly: healthCheck only *reads* p.sess, so a healthCheck-vs-exec test
// with no induced crashes never actually writes p.sess/p.cmd concurrently
// and would pass even without any locking. This test instead runs
// healthCheck concurrently with real restart() calls — restart is the
// operation that mutates p.sess/p.cmd (via close+spawn) — against a real
// subprocess, so `go test -race` exercises an actual concurrent read/write
// pair on those fields and would fail if either method touched them without
// holding mu.
func TestProcessHealthCheckAndRestartAreRaceFree(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	p, err := startProcess(t.Context(), binPath, ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	const restarts = 5
	var wg sync.WaitGroup
	wg.Add(2)

	stop := make(chan struct{})
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = p.healthCheck(t.Context())
			}
		}
	}()

	go func() {
		defer wg.Done()
		defer close(stop)
		for i := 0; i < restarts; i++ {
			require.NoError(t, p.restart(t.Context()))
		}
	}()

	wg.Wait()
	require.NoError(t, p.healthCheck(t.Context()))
}
