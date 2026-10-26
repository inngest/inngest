package duckdb

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"

	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var MigrationsFS embed.FS

// Mode is what kind of catalog Migrate runs against. Goose's ENVSUB can
// only substitute a variable or fall back to a default, never branch on a
// value, so each mode is the set of env vars (modeEnv) its "-- +goose ENVSUB
// ON" blocks substitute: an unset var keeps a statement's in-file default.
type Mode int

const (
	// ModeMemory is a bare in-memory catalog (devserver w/o --persist): no
	// DuckLake ATTACH, so Migrate creates the "inngest" schema itself and the
	// DuckLake-only SET SORTED/PARTITIONED BY statements are no-ops.
	ModeMemory Mode = iota
	// ModeDuckLake is a DuckLake catalog whose tables hold every emission
	// (devserver --persist): the migrations run unchanged.
	ModeDuckLake
	// ModeCloud is a DuckLake catalog whose runs are already one row per run
	// (cloud's lake, exported from final rows only): as ModeDuckLake, but
	// raw_runs doesn't collapse.
	ModeCloud
)

// modeEnv is the env vars each Mode sets while migrating.
//
// DUCKDB_DUCKLAKE_ONLY replaces each DuckLake-only statement. Every
// occurrence uses the no-colon "${VAR-default}" form, which (unlike
// "${VAR:-default}") only falls back to the default when the var is
// completely unset, so an explicitly-set value always wins. It can't be a
// literal empty string: over the quack transport (see
// pkg/duckdb/driver/internal/quack/session.go -- production's
// setupDualWrite always enables it, alongside the plain jsonlines transport
// tests here use directly), DuckDB's prepare/fetch protocol rejects a
// statement that resolves to nothing with "Query did not return any
// columns", even though the same empty statement is silently accepted over
// jsonlines. "SELECT 1" is a real, valid statement on both transports.
//
// DUCKDB_RUNS_COLLAPSE replaces raw_runs' QUALIFY, so empty drops it (the
// statement is still the rest of the macro, never empty).
var modeEnv = map[Mode]map[string]string{
	ModeMemory:   {"DUCKDB_DUCKLAKE_ONLY": "SELECT 1"},
	ModeDuckLake: {},
	ModeCloud:    {"DUCKDB_RUNS_COLLAPSE": ""},
}

// Migrate applies every staging-table migration to db. It's safe to call
// more than once (idempotent) and is run once at subprocess startup before
// dual-write is enabled.
//
// goose v3.27.0 ships no built-in DuckDB dialect (Postgres, SQLite, MySQL,
// and others, but not DuckDB). goose.DialectSQLite3 was tried first, since
// DuckDB is highly SQL/SQLite-compatible, but it doesn't work: the
// SQLite3 dialect's version-table DDL uses "INTEGER PRIMARY KEY
// AUTOINCREMENT", which DuckDB's parser rejects outright (confirmed against
// the real duckdb binary: "Parser Error: syntax error at or near
// AUTOINCREMENT"). So this uses goose.DialectCustom instead, with a
// hand-written database.Store (duckdbStore, in store.go) that generates
// DuckDB-compatible SQL for the goose_db_version bookkeeping table — and
// (see store.go's doc comment) also works around a second, independently
// discovered incompatibility in this driver's row-column ordering.
//
// mode says what db is (see Mode). ModeMemory's db was opened with
// Options.DBFile ":memory:" and Options.DuckLake nil, so there is no DuckLake
// ATTACH to create the "inngest" catalog schema and Migrate creates it
// directly; the others' db was opened with Options.DuckLake set.
func Migrate(ctx context.Context, db *sql.DB, mode Mode) error {
	env, ok := modeEnv[mode]
	if !ok {
		return fmt.Errorf("duckdb: unknown migration mode %d", mode)
	}
	if mode == ModeMemory {
		if _, err := db.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS "+driver.DuckLakeAlias+";"); err != nil {
			return fmt.Errorf("duckdb: creating in-memory schema %q: %w", driver.DuckLakeAlias, err)
		}
	}
	for k, v := range env {
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("duckdb: setting %s: %w", k, err)
		}
		defer func() { _ = os.Unsetenv(k) }()
	}

	migrationsFS, err := fs.Sub(MigrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("duckdb: reading embedded migrations: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectCustom, db, migrationsFS,
		goose.WithStore(newDuckdbStore(goose.DefaultTablename)),
		// This POC's driver.Conn doesn't implement transactions (see
		// conn.go's Begin, which always errors), so DDL/DML must run
		// directly against the connection rather than inside a goose
		// BeginTx/Commit wrapper.
		goose.WithIsolateDDL(true),
	)
	if err != nil {
		return fmt.Errorf("duckdb: creating goose provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("duckdb: running migrations: %w", err)
	}
	return nil
}
