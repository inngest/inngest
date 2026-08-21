package devserver

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/dualwrite"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
)

// dualWriteEnvVar gates the entire DuckDB dual-write POC. The spec
// (docs/plans/006-duckdb-poc-subprocess-dual-write.md) scopes this feature to
// `inngest dev`, but pkg/devserver.start() also backs `inngest start`
// (production self-hosted), so the feature must never turn itself on
// implicitly in either. It is opt-in, off by default, for both.
const dualWriteEnvVar = "INNGEST_DUCKDB_DUALWRITE"

// dualWriteEnabled reports whether the DuckDB dual-write POC has been
// explicitly opted into via INNGEST_DUCKDB_DUALWRITE. Off by default; set to
// "1", "true", "yes", or any other truthy value to enable. Mirrors the
// truthiness convention of metrics.ExperimentalPromMetricsEnabled.
func dualWriteEnabled() bool {
	v := strings.TrimSpace(os.Getenv(dualWriteEnvVar))
	if v == "" || v == "0" || strings.EqualFold(v, "false") || strings.EqualFold(v, "no") {
		return false
	}
	return true
}

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
// the dual-write listener — or nil if any step fails, in which case the
// caller must proceed without dual-write. This is additive, best-effort
// wiring for the POC described in
// docs/plans/006-duckdb-poc-subprocess-dual-write.md: a missing binary,
// failed spawn, or failed migration is logged once and must never block or
// crash devserver startup. binaryPath is the resolved path to try (tests can
// point this at a nonexistent path to exercise the failure path without
// needing to fake PATH resolution).
//
// Dual-write is opt-in: unless INNGEST_DUCKDB_DUALWRITE is set truthy, this
// returns nil immediately without spawning a subprocess, running migrations,
// or registering anything (see dualWriteEnabled).
//
// stateDir is the raw --sqlite-dir option value ("" meaning "the default");
// it is resolved the same way pkg/db/sqlite resolves it, so the catalog lands
// in <state-dir>/duckdb/ rather than relative to the process's cwd.
func setupDualWrite(ctx context.Context, binaryPath, stateDir string) execution.SyncLifecycleListener {
	l := logger.StdlibLogger(ctx)

	if !dualWriteEnabled() {
		return nil
	}

	if _, err := exec.LookPath(binaryPath); err != nil {
		l.Warn("duckdb binary not found; dual-write disabled", "path", binaryPath, "error", err)
		return nil
	}

	resolvedStateDir, err := util.ResolveStateDir(stateDir)
	if err != nil {
		l.Warn("failed to resolve state directory; dual-write disabled", "dir", stateDir, "error", err)
		return nil
	}

	duckdbDir := filepath.Join(resolvedStateDir, "duckdb")
	if err := os.MkdirAll(duckdbDir, 0o755); err != nil {
		l.Warn("failed to create duckdb state directory; dual-write disabled", "path", duckdbDir, "error", err)
		return nil
	}

	dbFile := filepath.Join(duckdbDir, "catalog.duckdb")
	db, err := duckdb.Open(ctx, duckdb.Options{BinaryPath: binaryPath, DBFile: dbFile})
	if err != nil {
		l.Warn("failed to start duckdb subprocess; dual-write disabled", "error", err)
		return nil
	}

	if err := duckdb.Migrate(ctx, db); err != nil {
		l.Warn("failed to migrate duckdb staging schema; dual-write disabled", "error", err)
		_ = db.Close()
		return nil
	}

	return dualwrite.NewListener(db)
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
