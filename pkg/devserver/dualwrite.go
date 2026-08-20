package devserver

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/dualwrite"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
)

// syncLifecycleCloser is satisfied by dual-write listeners that own
// background goroutines/resources needing explicit shutdown — in practice,
// the value setupDualWrite returns always satisfies this (see
// dualwrite.Closer). Declared locally (rather than referencing
// dualwrite.Closer directly) so callers holding only an
// execution.SyncLifecycleListener can type-assert without every caller
// needing to import dualwrite.
type syncLifecycleCloser interface {
	Close(ctx context.Context) error
}

// setupDualWrite starts the duckdb subprocess, runs migrations, and returns
// the dual-write listener devserver.go registers with the executor/runner,
// along with the same *sql.DB handle dual-write writes through — needed so
// devserver.go can also wrap it as the GQL resolver's DuckDB-backed data
// source (duckdbquery.Wrap). Both return values are set together or not at
// all — on any failure both are nil, in which case the caller must proceed
// without dual-write. This is additive, best-effort wiring for the POC: a
// missing binary, failed spawn, or failed migration is logged once and must
// never block or crash devserver startup. binaryPath is the resolved path to
// try (tests can point this at a nonexistent path to exercise the failure
// path without needing to fake PATH resolution).
//
// Dual-write is opt-in: unless enabled is true, this returns nil immediately
// without spawning a subprocess, running migrations, or registering
// anything. The spec scopes this feature to `inngest dev`, but
// pkg/devserver.start() also backs `inngest start` (production
// self-hosted), so callers must never pass true implicitly in either — enabled
// should come from an explicit user opt-in (the `--duckdb` flag /
// INNGEST_DUCKDB env var, resolved by the caller).
//
// stateDir is the raw --sqlite-dir option value ("" meaning "the default");
// it is resolved the same way pkg/db/sqlite resolves it, so the catalog lands
// in <state-dir>/duckdb/ rather than relative to the process's cwd.
func setupDualWrite(ctx context.Context, enabled bool, binaryPath, stateDir string) (execution.SyncLifecycleListener, *sql.DB) {
	l := logger.StdlibLogger(ctx)

	if !enabled {
		return nil, nil
	}

	if _, err := exec.LookPath(binaryPath); err != nil {
		l.Warn("duckdb binary not found; dual-write disabled", "path", binaryPath, "error", err)
		return nil, nil
	}

	resolvedStateDir, err := util.ResolveStateDir(stateDir)
	if err != nil {
		l.Warn("failed to resolve state directory; dual-write disabled", "dir", stateDir, "error", err)
		return nil, nil
	}

	duckdbDir := filepath.Join(resolvedStateDir, "duckdb")
	if err := os.MkdirAll(duckdbDir, 0o755); err != nil {
		l.Warn("failed to create duckdb state directory; dual-write disabled", "path", duckdbDir, "error", err)
		return nil, nil
	}

	dbFile := filepath.Join(duckdbDir, "main.duckdb")
	catalogFile := filepath.Join(duckdbDir, "catalog.duckdb")
	dataDir := filepath.Join(duckdbDir, "data")
	quackAddr, err := freeLocalQuackAddr()
	if err != nil {
		l.Warn("failed to allocate a local port for the quack listener; dual-write disabled", "error", err)
		return nil, nil
	}
	db, err := duckdb.Open(ctx, duckdb.Options{
		BinaryPath: binaryPath,
		DBFile:     dbFile,
		DuckLake: &duckdb.DuckLakeOptions{
			CatalogPath: catalogFile,
			DataPath:    dataDir,
		},
		QuackAddr: &quackAddr,
	})
	if err != nil {
		l.Warn("failed to start duckdb subprocess; dual-write disabled", "error", err)
		return nil, nil
	}

	if err := duckdb.Migrate(ctx, db); err != nil {
		l.Warn("failed to migrate duckdb staging schema; dual-write disabled", "error", err)
		_ = db.Close()
		return nil, nil
	}

	return dualwrite.NewListener(db), db
}

// freeLocalQuackAddr resolves an ephemeral loopback port and returns it as a
// "127.0.0.1:<port>" address for duckdb.Options.QuackAddr. A fixed address
// (quack's own default port, 9494) would make a second `inngest dev
// --duckdb` on the same machine either fail to bind or — worse, observed
// empirically — silently hand its client the first instance's already-bound
// listener, causing an auth failure instead of a clear bind error. Closing
// the listener immediately before duckdb binds it is a small TOCTOU race,
// acceptable here since a launch racing another process for the same port in
// that window is exceedingly unlikely and, if it happens, surfaces as an
// ordinary failed-to-start warning rather than the cross-connection bug
// above.
func freeLocalQuackAddr() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := l.Addr().(*net.TCPAddr)
	if err := l.Close(); err != nil {
		return "", err
	}
	return fmt.Sprintf("127.0.0.1:%d", addr.Port), nil
}

// stopDualWrite stops the background batcher goroutines started by
// setupDualWrite's listener and closes the underlying duckdb subprocess, if
// l is non-nil and implements the shutdown hook. This addresses the
// goroutine-leak gap flagged by Task 8's review: NewListener's batchers
// otherwise run for the life of the process with no way to stop them. Errors
// are logged, not returned — dual-write shutdown must never block or fail
// `inngest dev`'s own shutdown.
func stopDualWrite(ctx context.Context, l execution.SyncLifecycleListener) {
	if l == nil {
		return
	}
	c, ok := l.(syncLifecycleCloser)
	if !ok {
		return
	}
	if err := c.Close(ctx); err != nil {
		logger.StdlibLogger(ctx).Warn("error stopping dual-write listener", "error", err)
	}
}
