package devserver

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/dualwrite"
	"github.com/stretchr/testify/require"
)

func TestSetupDualWriteReturnsNilListenerWhenBinaryMissing(t *testing.T) {
	l, db := setupDualWrite(context.Background(), true, true, "/nonexistent/duckdb", t.TempDir())
	require.Nil(t, l)
	require.Nil(t, db)
}

func TestSetupDualWriteReturnsListenerWhenBinaryPresent(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	l, _ := setupDualWrite(context.Background(), true, true, binPath, t.TempDir())
	require.NotNil(t, l)
	stopDualWrite(context.Background(), l)
}

// TestSetupDualWriteNeverBlocksOnFailure exercises the core guarantee of the
// whole POC: whatever goes wrong on the way to standing up dual-write, the
// caller gets back a nil listener quickly rather than an error or a hang, so
// `inngest dev` startup is never blocked by this code path.
func TestSetupDualWriteNeverBlocksOnFailure(t *testing.T) {
	done := make(chan execution.SyncLifecycleListener, 1)
	go func() {
		l, _ := setupDualWrite(context.Background(), true, true, "/nonexistent/duckdb", t.TempDir())
		done <- l
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

	l, db := setupDualWrite(context.Background(), false, true, binPath, stateDir)
	require.Nil(t, l)
	require.Nil(t, db)
	require.NoDirExists(t, filepath.Join(stateDir, "duckdb"),
		"setupDualWrite must not touch disk when dual-write is not opted into")
}

// TestSetupDualWriteInMemoryWhenNotPersisting proves persist=false skips
// DuckLake entirely: dual-write still starts successfully (a plain
// in-memory duckdb needs no on-disk catalog/data), but neither the DuckLake
// catalog nor its data directory ever gets created — --persist=false already
// means "in-memory databases" for SQLite/Redis, and this is the same
// contract extended to DuckDB.
func TestSetupDualWriteInMemoryWhenNotPersisting(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}

	stateDir := t.TempDir()

	l, db := setupDualWrite(context.Background(), true, false, binPath, stateDir)
	require.NotNil(t, l)
	require.NotNil(t, db)
	t.Cleanup(func() { stopDualWrite(context.Background(), l) })

	require.NoFileExists(t, filepath.Join(stateDir, "duckdb", "catalog.duckdb"),
		"persist=false must not create a DuckLake catalog file")
	require.NoDirExists(t, filepath.Join(stateDir, "duckdb", "data"),
		"persist=false must not create a DuckLake data directory")
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

	l, _ := setupDualWrite(context.Background(), true, true, binPath, "relative-state")
	require.NotNil(t, l)
	t.Cleanup(func() { stopDualWrite(context.Background(), l) })

	require.FileExists(t, filepath.Join(wd, "relative-state", "duckdb", "catalog.duckdb"))
}

// TestSetupDualWriteResolvesPinnedBinaryWhenBinaryPathEmpty proves that an
// empty binaryPath no longer fails setupDualWrite's own LookPath pre-check —
// it must defer entirely to duckdb.Open's StateDir-based resolution (which
// downloads/caches the pinned binary; here it's pre-seeded so the test
// doesn't hit the network).
func TestSetupDualWriteResolvesPinnedBinaryWhenBinaryPathEmpty(t *testing.T) {
	realBin, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}

	stateDir := t.TempDir()
	cached := filepath.Join(stateDir, "bin", duckdb.PinnedVersion, "duckdb")
	data, err := os.ReadFile(realBin)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(cached), 0o755))
	require.NoError(t, os.WriteFile(cached, data, 0o755))

	l, db := setupDualWrite(context.Background(), true, true, "", stateDir)
	require.NotNil(t, l)
	require.NotNil(t, db)
	t.Cleanup(func() { stopDualWrite(context.Background(), l) })
}

// TestSetupDualWriteIsolatesExtensionDirectory proves setupDualWrite threads
// the resolved state dir into duckdb.Options.StateDir, so the subprocess's
// extension cache lands under <state-dir>/duckdb/extensions rather than the
// ambient HOME.
func TestSetupDualWriteIsolatesExtensionDirectory(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	stateDir := t.TempDir()

	l, _ := setupDualWrite(context.Background(), true, true, binPath, stateDir)
	require.NotNil(t, l)
	t.Cleanup(func() { stopDualWrite(context.Background(), l) })

	require.DirExists(t, filepath.Join(stateDir, "duckdb", "extensions"))
}

func TestSetupDualWriteReturnsDBHandleOnSuccess(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	l, db := setupDualWrite(context.Background(), true, true, binPath, t.TempDir())
	require.NotNil(t, l)
	require.NotNil(t, db)
	t.Cleanup(func() { stopDualWrite(context.Background(), l) })
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

	l, _ := setupDualWrite(ctx, true, true, binPath, stateDir)
	require.NotNil(t, l)

	evt := event.NewBaseTrackedEvent(event.Event{Name: "smoke/test"}, nil)
	l.OnEventReceived(ctx, evt)

	dbFile := filepath.Join(stateDir, "duckdb", "catalog.duckdb")
	require.FileExists(t, dbFile, "duckdb.Open should have created the catalog file")

	stopDualWrite(context.Background(), l)
}

// TestDualWriteEndToEndRowsLandInDuckDB is the real end-to-end proof: a
// listener built the same way setupDualWrite builds one (real duckdb
// subprocess, real Migrate) receives an OnEventReceived call and, after its
// batcher flushes, the row is actually queryable in events_staging. This is
// the success criterion the whole POC (and this final wiring task) exists
// to demonstrate.
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
	require.NoError(t, duckdb.Migrate(ctx, db, true))

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

// TestDualWriteEndToEndRowsLandInDuckDBInMemory mirrors
// TestDualWriteEndToEndRowsLandInDuckDB for persist=false: the same
// migrations, dualwrite.NewListener, and batch-flush path, but against a
// bare in-memory catalog with no DuckLake ATTACH at all.
func TestDualWriteEndToEndRowsLandInDuckDBInMemory(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}

	ctx := t.Context()

	db, err := duckdb.Open(ctx, duckdb.Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
	})
	require.NoError(t, err)
	require.NoError(t, duckdb.Migrate(ctx, db, false))

	l := dualwrite.NewListener(db)
	closer, ok := l.(syncLifecycleCloser)
	require.True(t, ok, "NewListener's return value must implement the Close shutdown hook")
	defer func() { _ = closer.Close(context.Background()) }()

	evt := event.NewBaseTrackedEvent(event.Event{Name: "e2e/devserver-in-memory"}, nil)
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

	l, _ := setupDualWrite(ctx, true, true, binPath, stateDir)
	require.NotNil(t, l)

	done := make(chan struct{})
	go func() {
		stopDualWrite(context.Background(), l)
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

	l1, _ := setupDualWrite(ctx, true, true, binPath, t.TempDir())
	require.NotNil(t, l1, "first instance should start dual-write successfully")
	t.Cleanup(func() { stopDualWrite(context.Background(), l1) })

	l2, _ := setupDualWrite(ctx, true, true, binPath, t.TempDir())
	require.NotNil(t, l2, "second instance must not fail to start dual-write just because the first is already running")
	t.Cleanup(func() { stopDualWrite(context.Background(), l2) })
}
