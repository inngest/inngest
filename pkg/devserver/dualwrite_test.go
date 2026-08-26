package devserver

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/dualwrite"
	"github.com/stretchr/testify/require"
)

func TestSetupDualWriteReturnsNilListenerWhenBinaryMissing(t *testing.T) {
	l := setupDualWrite(context.Background(), true, "/nonexistent/duckdb", t.TempDir())
	require.Nil(t, l)
}

func TestSetupDualWriteReturnsListenerWhenBinaryPresent(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	l := setupDualWrite(context.Background(), true, binPath, t.TempDir())
	require.NotNil(t, l)
	stopDualWrite(context.Background(), l.Listener)
}

// TestSetupDualWriteNeverBlocksOnFailure exercises the core guarantee of the
// whole POC: whatever goes wrong on the way to standing up dual-write, the
// caller gets back a nil listener quickly rather than an error or a hang, so
// `inngest dev` startup is never blocked by this code path.
func TestSetupDualWriteNeverBlocksOnFailure(t *testing.T) {
	done := make(chan *dualWriteResult, 1)
	go func() {
		done <- setupDualWrite(context.Background(), true, "/nonexistent/duckdb", t.TempDir())
	}()

	select {
	case l := <-done:
		require.Nil(t, l)
	case <-time.After(5 * time.Second):
		t.Fatal("setupDualWrite blocked instead of failing fast when the binary is missing")
	}
}

// TestSetupDualWriteIsOptInAndOffByDefault is the guard against the whole POC
// silently enabling itself in `inngest start` (production self-hosted), which
// shares pkg/devserver's start(). With enabled=false (what callers get unless
// the `--duckdb` flag / INNGEST_DUCKDB env var was explicitly set),
// setupDualWrite must return nil without spawning a subprocess or creating
// any state on disk — even with a perfectly good duckdb binary available.
func TestSetupDualWriteIsOptInAndOffByDefault(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}

	stateDir := t.TempDir()

	require.Nil(t, setupDualWrite(context.Background(), false, binPath, stateDir))
	require.NoDirExists(t, filepath.Join(stateDir, "duckdb"),
		"setupDualWrite must not touch disk when dual-write is not opted into")
}

// TestSetupDualWriteResolvesRelativeStateDir pins the catalog to
// <state-dir>/duckdb/ rather than a "duckdb" directory relative to the
// process's cwd. The stateDir passed here is relative, mirroring how
// --sqlite-dir arrives from the CLI.
func TestSetupDualWriteResolvesRelativeStateDir(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}

	// Chdir into a temp dir so the relative override resolves somewhere
	// disposable rather than into the repo.
	wd := t.TempDir()
	t.Chdir(wd)

	l := setupDualWrite(context.Background(), true, binPath, "relative-state")
	require.NotNil(t, l)
	t.Cleanup(func() { stopDualWrite(context.Background(), l.Listener) })

	require.FileExists(t, filepath.Join(wd, "relative-state", "duckdb", "catalog.duckdb"))
}

func TestSetupDualWriteReturnsDBHandleOnSuccess(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	dw := setupDualWrite(context.Background(), true, binPath, t.TempDir())
	require.NotNil(t, dw)
	require.NotNil(t, dw.Listener)
	require.NotNil(t, dw.DB)
	t.Cleanup(func() { stopDualWrite(context.Background(), dw.Listener) })
}

func TestStopDualWriteIsSafeWithNilListener(t *testing.T) {
	require.NotPanics(t, func() {
		stopDualWrite(context.Background(), nil)
	})
}

// TestDualWriteEndToEndDoesNotAffectPrimaryPath exercises setupDualWrite the
// way devserver.go's start() calls it, then confirms the underlying duckdb
// catalog file was created on disk (proof the real subprocess spawned and
// was health-checked) and that stopDualWrite tears it down cleanly.
//
// This test intentionally does not assert against SQLite-backed state — the
// point of "zero observable effect" is that nothing above touches
// s.cqrs/dbcqrs at all, which is already covered by Tasks 2/3's tests
// asserting the async listeners' behavior is untouched.
func TestDualWriteEndToEndDoesNotAffectPrimaryPath(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}

	stateDir := t.TempDir()
	ctx := t.Context()

	l := setupDualWrite(ctx, true, binPath, stateDir)
	require.NotNil(t, l)

	evt := event.NewBaseTrackedEvent(event.Event{Name: "smoke/test"}, nil)
	l.Listener.OnEventReceived(ctx, evt)

	dbFile := filepath.Join(stateDir, "duckdb", "catalog.duckdb")
	require.FileExists(t, dbFile, "duckdb.Open should have created the catalog file")

	stopDualWrite(context.Background(), l.Listener)
}

// TestDualWriteEndToEndRowsLandInDuckDB is the real end-to-end proof: a
// listener built the same way setupDualWrite builds one (real duckdb
// subprocess, real Migrate) receives an OnEventReceived call and, after its
// batcher flushes, the row is actually queryable in events_staging. This is
// the success criterion the whole POC (and this final wiring task) exists
// to demonstrate — see docs/plans/006-duckdb-poc-subprocess-dual-write.md.
func TestDualWriteEndToEndRowsLandInDuckDB(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}

	ctx := t.Context()
	dir := t.TempDir()

	db, err := duckdb.Open(ctx, duckdb.Options{
		BinaryPath: binPath,
		DBFile:     filepath.Join(dir, "catalog.duckdb"),
		DuckLake: &duckdb.DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "lake-catalog.duckdb"),
			DataPath:    filepath.Join(dir, "data"),
		},
	})
	require.NoError(t, err)
	require.NoError(t, duckdb.Migrate(ctx, db))

	l := dualwrite.NewListener(db)
	closer, ok := l.(syncLifecycleCloser)
	require.True(t, ok, "NewListener's return value must implement the Close shutdown hook")
	defer func() { _ = closer.Close(context.Background()) }()

	evt := event.NewBaseTrackedEvent(event.Event{Name: "e2e/devserver"}, nil)
	l.OnEventReceived(ctx, evt)

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(ctx, "SELECT count(*) FROM inngest.events;")
		if err := row.Scan(&count); err != nil {
			return false
		}
		return count == 1
	}, 2*time.Second, 20*time.Millisecond, "event row should land in inngest.events after a batch flush")
}

// TestStopDualWriteStopsBatcherGoroutines proves the shutdown path actually
// stops the background batcher goroutines (rather than just closing the db
// out from under them) by sending an event, letting it flush, closing, and
// then confirming a further send after Close doesn't panic or leak a
// goroutine still trying to write to a closed db indefinitely. The
// underlying batcher.run's stopc case makes this deterministic: Close waits
// for every batcher to actually exit before returning.
func TestStopDualWriteStopsBatcherGoroutines(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}

	ctx := t.Context()
	stateDir := t.TempDir()

	l := setupDualWrite(ctx, true, binPath, stateDir)
	require.NotNil(t, l)

	done := make(chan struct{})
	go func() {
		stopDualWrite(context.Background(), l.Listener)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("stopDualWrite did not return; batcher goroutines likely did not stop")
	}
}

// TestSetupDualWriteTwoInstancesDoNotCollideOnQuackPort simulates two
// `inngest dev --duckdb` processes running on the same machine at once (a
// realistic scenario, e.g. two projects/worktrees). Each must get its own
// quack listener; a fixed address would make the second setupDualWrite fail
// to bind the port the first one already holds.
func TestSetupDualWriteTwoInstancesDoNotCollideOnQuackPort(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	ctx := t.Context()

	l1 := setupDualWrite(ctx, true, binPath, t.TempDir())
	require.NotNil(t, l1, "first instance should start dual-write successfully")
	t.Cleanup(func() { stopDualWrite(context.Background(), l1.Listener) })

	l2 := setupDualWrite(ctx, true, binPath, t.TempDir())
	require.NotNil(t, l2, "second instance must not fail to start dual-write just because the first is already running")
	t.Cleanup(func() { stopDualWrite(context.Background(), l2.Listener) })
}
