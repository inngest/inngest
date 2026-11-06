package devserver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/icebergcatalog"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/inngest/inngest/pkg/duckdb/query"
	"github.com/inngest/inngest/pkg/duckdb/tracing"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
)

// init plugs insights' CEL-to-SQL conversion into the duckdb read path's
// CEL search (see query.CELEventFilters' doc comment). It lives here, not
// in an init() in pkg/duckdb/insights, since that would need insights to
// import query and query's own tests already import insights.
func init() {
	query.CELEventTableFilters = insights.CELEventTableFilters
	query.CELEventFilters = insights.CELEventFilters
	query.CELOutputFilters = insights.CELOutputFilters
	query.RenderWhereSQL = insights.RenderWhereSQL
}

// dualWriteQuackConns is the default *sql.DB connection cap setupDualWrite
// opens dual-write's own handle with (driver.Options.QuackConns),
// overridable w/ INNGEST_DUCKDB_WRITE_CONNS. Above 1, database/sql can run
// that many statements genuinely concurrently instead of serializing every
// one of them behind a single mutex -- see driver.Options.QuackConns. The
// read handle gets its own quack sessions either way (Connector.OpenShared),
// so even 1 doesn't make reads share dual-write's connection.
const dualWriteQuackConns = 16

// duckDBReadConns is the default cap on the separate read handle
// setupDualWrite returns for Insights, Sessions and query.Wrap
// (--duckdb-reads), overridable w/ INNGEST_DUCKDB_READ_CONNS. It's its own
// pool so slow user queries can wait on each other but never take a
// connection dual-write's batch flushes need. Sized for "a handful of
// simultaneously open Insights tabs," not production load.
const duckDBReadConns = 8

const (
	duckDBWriteConnsEnv = "INNGEST_DUCKDB_WRITE_CONNS"
	duckDBReadConnsEnv  = "INNGEST_DUCKDB_READ_CONNS"
)

// duckDBConnLimit reads a positive connection cap from env, falling back to
// def (w/ a warning) when it's unset, not an integer, or below 1.
func duckDBConnLimit(ctx context.Context, env string, def int) int {
	raw := os.Getenv(env)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		logger.StdlibLogger(ctx).Warn("ignoring invalid duckdb connection limit", "env", env, "value", raw, "default", def)
		return def
	}
	return n
}

// setupDualWrite starts the duckdb subprocess, runs migrations, and returns
// the dual-write listener devserver.go registers with the executor/runner,
// along with the *sql.DB handle dual-write writes through and a separate,
// smaller read handle on the same subprocess (duckDBReadConns, or
// INNGEST_DUCKDB_READ_CONNS) for
// Insights, Sessions and query.Wrap. Closing readDB leaves the subprocess
// up; closing db (via the listener) tears it down. All three return values
// are set together or not at all -- on any failure all are nil, in which case the caller must proceed
// without dual-write. This is additive, best-effort wiring for the POC: a
// missing binary, failed spawn, or failed migration is logged once and must
// never block or crash devserver startup. binaryPath is an explicit path to
// use (tests can point this at a nonexistent path to exercise the failure
// path without needing to fake PATH resolution); leave it empty to have
// driver.Open download/cache the pinned version under stateDir instead
// (production's default — see driver.Options.BinaryPath).
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
// in <state-dir>/duckdb/ rather than relative to the process's cwd. This
// only applies when persist is true — see below for the persist=false
// default.
//
// persist mirrors devserver's --persist flag (StartOpts.Persist), the same
// knob that already gates SQLite/Redis persistence -- ANDed by the caller
// with the EXPERIMENTAL_DUCKDB_PERSISTENT_DUALWRITE feature flag, so an
// on-disk Iceberg lake additionally requires that flag even when --persist
// is set (see devserver.go's start()). true starts an embedded Iceberg REST
// catalog (pkg/duckdb/icebergcatalog) on 127.0.0.1 and attaches it as the
// lake; it's closed w/ the listener. false skips the lake entirely and
// opens a plain in-memory duckdb catalog instead (no on-disk catalog/data
// files), matching how --persist=false already means
// "in-memory databases" for the rest of the dev server. It also changes the
// default (stateDir == "") location of the duckdb binary cache and extension
// directory (see driver.Options.StateDir): rather than the usual
// cwd-relative .inngest, they default to Go's os.TempDir(), since nothing
// under StateDir is worth keeping around once the in-memory catalog is gone
// anyway. An explicit stateDir is still honored even when persist is false.
//
// postgresURI, when non-empty and persist is true, keeps the Iceberg
// catalog's table pointers in that Postgres database instead of a local
// sqlite file — see icebergConfigFor. It is ignored when persist is false,
// matching devserver.go's own gating (opts.Persist && enableDuckDBPersistence).
func setupDualWrite(ctx context.Context, enabled, persist bool, binaryPath, stateDir, postgresURI string) (listener execution.SyncLifecycleListener, db, readDB *sql.DB) {
	l := logger.StdlibLogger(ctx)

	if !enabled {
		return nil, nil, nil
	}

	// An empty binaryPath means "resolve the pinned/downloaded binary" (see
	// driver.Options.BinaryPath) — only a caller-supplied path is checked
	// here, up front, so a bad explicit override still fails fast with a
	// clear log message instead of only surfacing once driver.Open tries to
	// spawn it.
	if binaryPath != "" {
		if _, err := exec.LookPath(binaryPath); err != nil {
			l.Warn("duckdb binary not found; dual-write disabled", "path", binaryPath, "error", err)
			return nil, nil, nil
		}
	}

	if stateDir == "" && !persist {
		stateDir = filepath.Join(os.TempDir(), "inngest-duckdb")
	}

	resolvedStateDir, err := util.ResolveStateDir(stateDir)
	if err != nil {
		l.Warn("failed to resolve state directory; dual-write disabled", "dir", stateDir, "error", err)
		return nil, nil, nil
	}

	duckdbDir := filepath.Join(resolvedStateDir, "duckdb")
	if err := os.MkdirAll(duckdbDir, 0o755); err != nil {
		l.Warn("failed to create duckdb state directory; dual-write disabled", "path", duckdbDir, "error", err)
		return nil, nil, nil
	}

	// A kernel-assigned port rather than quack's default (9494) or a
	// pre-probed free one: quack_serve binds with address reuse, so a second
	// `inngest dev --duckdb` on a fixed or raced port would "successfully"
	// share the first instance's port instead of failing to bind.
	quackAddr := driver.EphemeralQuackAddr

	// persist=false skips the lake entirely: a plain in-memory catalog with
	// no on-disk main/catalog/data files, matching --persist=false's existing
	// meaning for SQLite/Redis ("use in-memory databases").
	dbFile := ":memory:"
	var (
		lake    *driver.IcebergOptions
		catalog *icebergcatalog.Embedded
	)
	if persist {
		dbFile = filepath.Join(duckdbDir, "main.duckdb")
		catalog, err = icebergcatalog.Open(ctx, icebergConfigFor(duckdbDir, postgresURI))
		if err != nil {
			l.Warn("failed to start the embedded iceberg catalog; dual-write disabled", "error", err)
			return nil, nil, nil
		}
		lake = &driver.IcebergOptions{URI: catalog.URL, Warehouse: icebergcatalog.CatalogName, Token: catalog.Token, DataPath: catalog.Warehouse}
	}
	closeCatalog := func() {
		if catalog != nil {
			_ = catalog.Close()
		}
	}

	connector, db, err := driver.OpenConnector(ctx, driver.Options{
		BinaryPath: binaryPath,
		StateDir:   resolvedStateDir,
		DBFile:     dbFile,
		Iceberg:    lake,
		QuackAddr:  &quackAddr,
		QuackConns: duckDBConnLimit(ctx, duckDBWriteConnsEnv, dualWriteQuackConns),
		// Insights runs user-written SQL on this same handle, so sandbox the
		// subprocess (no file/env access beyond the catalog and warehouse)
		// as defense in depth behind insights' validator. The catalog's
		// own backing db (sqlite or postgres) is in this process, not
		// duckdb, so it doesn't matter which one it is.
		RestrictExternalAccess: true,
		// Also bound memory/threads/temp spill, so a heavy Insights query
		// can't take the whole machine (these are per-process, so they're
		// shared w/ dual-write's own writes).
		ResourceLimits: driver.DefaultResourceLimits(),
	})
	if err != nil {
		l.Warn("failed to start duckdb subprocess; dual-write disabled", "error", err)
		closeCatalog()
		return nil, nil, nil
	}

	mode := duckdb.ModeMemory
	if persist {
		mode = duckdb.ModeRawLake
	}
	if err := duckdb.Migrate(ctx, db, mode); err != nil {
		l.Warn("failed to migrate duckdb staging schema; dual-write disabled", "error", err)
		_ = db.Close()
		closeCatalog()
		return nil, nil, nil
	}

	readDB = connector.OpenShared(duckDBConnLimit(ctx, duckDBReadConnsEnv, duckDBReadConns))
	if catalog == nil {
		return tracing.NewListener(db), db, readDB
	}
	listener = tracing.NewListener(db, tracing.WithBatchInterval(lakeFlushInterval))
	return lakeListener{SyncLifecycleListener: listener, catalog: catalog}, db, readDB
}

// lakeFlushInterval is how often the persisted lake's batchers flush. Each
// flush is an Iceberg commit (a snapshot per statement, plus a metadata.json
// and manifest list), and there's no inlining like DuckLake had, so the
// in-memory mode's 200ms would make a pile of tiny files + snapshots.
//
// TODO(maintenance): the catalog expires old snapshots + metadata.json
// files (see icebergcatalog.Retention), but nothing removes orphan files or
// compacts the lake yet, so data + manifests still grow w/o bound.
const lakeFlushInterval = 3 * time.Second

// lakeListener is the dual-write listener for a persisted lake: Close stops
// the listener (closing duckdb) and then the embedded catalog duckdb was
// attached to.
type lakeListener struct {
	execution.SyncLifecycleListener
	catalog *icebergcatalog.Embedded
}

func (l lakeListener) Close(ctx context.Context) error {
	var err error
	if c, ok := l.SyncLifecycleListener.(tracing.Closer); ok {
		err = c.Close(ctx)
	}
	return errors.Join(err, l.catalog.Close())
}

// icebergConfigFor builds the embedded catalog config for a persisted
// (--persist) lake rooted at duckdbDir: data + metadata files under
// duckdbDir/warehouse, and the catalog's table pointers in
// duckdbDir/catalog.sqlite, or in postgresURI's database when set (so a dev
// server already configured w/ --postgres-uri doesn't also keep a local
// catalog file).
//
// This applies to `inngest start` too (which always persists), so a
// self-hosted `inngest start --duckdb --postgres-uri ...` w/
// EXPERIMENTAL_DUCKDB_PERSISTENT_DUALWRITE=true creates iceberg-go's catalog
// tables (`iceberg_tables`, `iceberg_namespace_properties`) in the production
// Postgres database, in its default schema alongside Inngest's own tables.
// That's accepted for this experimental POC.
//
// Kept as a pure function (no I/O) so this branching is unit-testable
// without starting anything.
func icebergConfigFor(duckdbDir, postgresURI string) icebergcatalog.Config {
	cfg := icebergcatalog.Config{Warehouse: filepath.Join(duckdbDir, "warehouse")}
	if postgresURI != "" {
		cfg.PostgresURI = postgresURI
	} else {
		cfg.SQLitePath = filepath.Join(duckdbDir, "catalog.sqlite")
	}
	return cfg
}

// stopDualWrite stops the background batcher goroutines started by
// setupDualWrite's listener and closes the underlying duckdb subprocess, if
// l is non-nil and implements the shutdown hook. NewListener's batchers
// otherwise run for the life of the process with no way to stop them. Errors
// are logged, not returned — dual-write shutdown must never block or fail
// `inngest dev`'s own shutdown.
func stopDualWrite(ctx context.Context, l execution.SyncLifecycleListener) {
	if l == nil {
		return
	}
	c, ok := l.(tracing.Closer)
	if !ok {
		return
	}
	if err := c.Close(ctx); err != nil {
		logger.StdlibLogger(ctx).Warn("error stopping dual-write listener", "error", err)
	}
}

// validateDuckDBPersistenceFlags rejects --persist combined with --duckdb
// unless enableDuckDBPersistence (EXPERIMENTAL_DUCKDB_PERSISTENT_DUALWRITE)
// is also set, instead of silently dual-writing against an in-memory catalog
// in that case. --persist without --duckdb, or --duckdb without --persist
// (the in-memory dual-write catalog, which stays available regardless of
// this flag -- only the on-disk lake, data that outlives the
// process, is gated, since that's the behavior with real operational
// impact: disk usage, accumulating state), are both still fine. Silently
// honoring --persist for everything else but not for dual-write was a
// confusing surprise; failing fast here means a caller finds out at
// startup, not by noticing dual-write data vanished after a restart.
func validateDuckDBPersistenceFlags(persist, enableDuckDB, enableDuckDBPersistence bool) error {
	if persist && enableDuckDB && !enableDuckDBPersistence {
		return fmt.Errorf("--persist with --duckdb also requires the EXPERIMENTAL_DUCKDB_PERSISTENT_DUALWRITE=true feature flag to write an on-disk Iceberg lake; unset --persist, drop --duckdb, or set that env var")
	}
	return nil
}
