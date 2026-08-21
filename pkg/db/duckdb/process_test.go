package duckdb

import (
	"context"
	"database/sql"
	"os/exec"
	"sync"
	"testing"
	"time"

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
// ErrDisabled rather than repeatedly retrying a respawn that cannot
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

	// Every subsequent call must fail fast with ErrDisabled instead
	// of attempting another (still-doomed) restart.
	_, err = db.ExecContext(t.Context(), "SELECT 1;")
	require.ErrorIs(t, err, ErrDisabled)
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

// TestRestartSurvivesTriggeringContextCancellation is a regression test for
// a bug introduced by the exec-triggered-restart fix above: spawnLocked must
// start the subprocess under the process's own long-lived procCtx, not the
// per-call ctx that happened to trigger the spawn. exec.CommandContext kills
// its process for the *entire* lifetime of the context passed to it, not
// just while starting — so if a restart is triggered from inside a
// short-lived, per-request context (exactly the shape Task 8's dualwrite
// package uses: one context per batch flush or hook call), spawning under
// that ctx would kill the freshly-restarted, perfectly healthy subprocess
// the instant that unrelated context ends.
//
// This crashes the subprocess, triggers a restart via a real p.exec call
// using a ctx that is cancelled immediately after that call returns, and
// then confirms the restarted subprocess is still alive and healthy well
// after that cancellation — proving its OS lifetime is governed by
// process's own procCtx, not the triggering call's ctx.
func TestRestartSurvivesTriggeringContextCancellation(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	p, err := startProcess(t.Context(), binPath, ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	// Simulate a crash out from under the session.
	require.NoError(t, p.cmd.Process.Kill())
	_, _ = p.cmd.Process.Wait()

	// Trigger the restart through a short-lived context, mirroring a single
	// dualwrite batch-flush/hook-call context.
	shortCtx, cancel := context.WithCancel(t.Context())
	_, err = p.exec(shortCtx, "SELECT 1;")
	require.NoError(t, err, "exec should transparently restart and succeed")

	// End the triggering context shortly after the call returns, as a real
	// per-request context would.
	cancel()

	// Give an incorrectly context-tied subprocess time to be killed by
	// ctx cancellation before we check on it.
	time.Sleep(200 * time.Millisecond)

	require.NoError(t, p.healthCheck(t.Context()),
		"restarted subprocess must survive the triggering call's context ending")
}

// TestExecSurfacesConstraintViolation is the regression test for the silent
// data-loss bug the final review found: the DuckDB CLI reports constraint /
// type / schema errors only on stderr and *still* emits the next statement's
// marker on stdout, so a driver that watched stdout alone reported success
// for statements DuckDB had actually rejected. The subprocess must stay
// healthy and usable afterwards — a rejected statement is not a crash.
func TestExecSurfacesConstraintViolation(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE t (id INTEGER NOT NULL);")
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), "INSERT INTO t VALUES (NULL);")
	require.Error(t, err, "a NOT NULL violation must not be reported as success")
	require.ErrorIs(t, err, errStatementFailed)
	require.Contains(t, err.Error(), "Constraint Error")

	// The row must genuinely not be there, and the session must still be
	// usable: a SQL error is not a transport failure, so no restart happened.
	var count int
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(*) AS c FROM t;").Scan(&count))
	require.Equal(t, 0, count)

	_, err = db.ExecContext(t.Context(), "INSERT INTO t VALUES (1);")
	require.NoError(t, err)
	require.NoError(t, db.QueryRowContext(t.Context(), "SELECT count(*) AS c FROM t;").Scan(&count))
	require.Equal(t, 1, count)
}

// TestExecSurfacesTypeAndSchemaErrors covers the other two shapes of SQL
// failure the dual-write path can hit — a type/conversion failure and schema
// drift (a column that doesn't exist) — since DuckDB reports each with a
// different "<Kind> Error:" prefix and isErrorDiagnostic must catch them all.
func TestExecSurfacesTypeAndSchemaErrors(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE t (id INTEGER);")
	require.NoError(t, err)

	// Conversion Error.
	_, err = db.ExecContext(t.Context(), "INSERT INTO t VALUES ('not-an-int');")
	require.ErrorIs(t, err, errStatementFailed)

	// Binder Error: schema drift, i.e. a column the migration never created.
	_, err = db.ExecContext(t.Context(), "INSERT INTO t (nonexistent_column) VALUES (1);")
	require.ErrorIs(t, err, errStatementFailed)

	// Catalog Error: a missing table.
	_, err = db.ExecContext(t.Context(), "INSERT INTO nonexistent_table VALUES (1);")
	require.ErrorIs(t, err, errStatementFailed)

	// Still healthy after three rejected statements.
	_, err = db.ExecContext(t.Context(), "INSERT INTO t VALUES (1);")
	require.NoError(t, err)
}

// TestExecStatementErrorDoesNotRestartSubprocess pins the classification
// itself: a SQL error must not be mistaken for a dead subprocess, because
// exec's restart-then-disable policy would otherwise burn its single restart
// attempt (and eventually disable dual-write entirely) on a bad row.
func TestExecStatementErrorDoesNotRestartSubprocess(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	c := &Connector{opts: Options{BinaryPath: binPath, DBFile: ":memory:"}}
	db := sql.OpenDB(c)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	_, err := db.ExecContext(t.Context(), "CREATE TABLE t (id INTEGER NOT NULL);")
	require.NoError(t, err)

	pidBefore := c.proc.cmd.Process.Pid

	_, err = db.ExecContext(t.Context(), "INSERT INTO t VALUES (NULL);")
	require.Error(t, err)

	require.Equal(t, pidBefore, c.proc.cmd.Process.Pid,
		"a rejected statement must not respawn the subprocess")
	c.proc.mu.Lock()
	disabled := c.proc.disabled
	c.proc.mu.Unlock()
	require.False(t, disabled)
}

// TestExecRespectsContextCancellation covers session.exec's previously
// ignored ctx parameter. Cancelling mid-statement must return promptly with
// the context error rather than blocking in a pipe read, and — because
// abandoning a statement leaves the session unable to frame the subprocess's
// remaining output — the driver must still be usable afterwards, via the
// respawn exec performs to resync.
func TestExecRespectsContextCancellation(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	db, err := Open(t.Context(), Options{BinaryPath: binPath, DBFile: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	start := time.Now()
	_, err = db.ExecContext(ctx, "SELECT 1;")
	require.Error(t, err)
	require.Less(t, time.Since(start), 5*time.Second, "a cancelled exec must not block on the read")

	// database/sql may reject a pre-cancelled ctx before ever reaching the
	// driver, so this only asserts the pool recovers — the desync/resync path
	// itself is covered directly below.
	_, err = db.ExecContext(t.Context(), "SELECT 1;")
	require.NoError(t, err)
}

// TestSessionExecCancelMidStatementDesyncsAndProcessResyncs drives the
// cancellation path at the layer that owns it, bypassing database/sql's own
// pre-flight ctx check. The statement is written to a live subprocess and then
// abandoned, which must (a) return the ctx error, (b) mark the session
// unusable rather than letting the next exec read this statement's output as
// its own, and (c) leave process.exec able to recover by respawning.
func TestSessionExecCancelMidStatementDesyncsAndProcessResyncs(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	p, err := startProcess(t.Context(), binPath, ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = p.sess.exec(ctx, "SELECT 42 AS answer;")
	require.ErrorIs(t, err, errSessionDesynced)
	require.ErrorIs(t, err, context.Canceled)

	// A desynced session refuses further work outright, so the abandoned
	// statement's queued output can never be misread as another statement's
	// result.
	_, err = p.sess.exec(t.Context(), "SELECT 1 AS ok;")
	require.ErrorIs(t, err, errSessionDesynced)

	// process.exec recognises the desync as recoverable-by-respawn and does
	// so, using a context detached from the (dead) caller ctx for the health
	// check.
	rows, err := p.exec(t.Context(), "SELECT 7 AS seven;")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, float64(7), rows[0]["seven"])
}

// TestExecFailedRestartWrapsErrDisabledImmediately pins Fix 5's contract: the
// *first* error a caller sees once the one-restart-then-disable policy fires
// already wraps ErrDisabled, so pkg/execution/dualwrite can stop flushing
// immediately instead of having to fail a second time to learn the state is
// terminal.
func TestExecFailedRestartWrapsErrDisabledImmediately(t *testing.T) {
	binPath := requireDuckDBBinary(t)

	c := &Connector{opts: Options{BinaryPath: binPath, DBFile: ":memory:"}}
	db := sql.OpenDB(c)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	_, err := db.ExecContext(t.Context(), "SELECT 1;")
	require.NoError(t, err)

	c.proc.mu.Lock()
	c.proc.binaryPath = "/nonexistent/path/to/duckdb"
	c.proc.mu.Unlock()

	require.NoError(t, c.proc.cmd.Process.Kill())
	_, _ = c.proc.cmd.Process.Wait()

	_, err = db.ExecContext(t.Context(), "SELECT 1;")
	require.ErrorIs(t, err, ErrDisabled, "the disabling error itself must be observable as ErrDisabled")
}
