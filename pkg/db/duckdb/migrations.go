package duckdb

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var MigrationsFS embed.FS

// ducklakeOnlyEnvVar is the single env var every "-- +goose ENVSUB ON" block
// in migrations/000001_baseline.sql and migrations/000003_run_metadata.sql
// substitutes: each occurrence's own in-file default is a DuckLake-only "SET
// SORTED/PARTITIONED BY" statement, used as-is (unset var) when persist is
// true. When persist is false, Migrate sets this var to a harmless "SELECT
// 1" no-op — note the no-colon "${VAR-default}" form each occurrence uses,
// which (unlike "${VAR:-default}") only falls back to the default when the
// var is completely unset, so an explicitly-set value always wins.
//
// The override can't be a literal empty string: over the quack transport
// (see quack_session.go — production's setupDualWrite always enables it,
// alongside the plain jsonlines transport tests here use directly), DuckDB's
// prepare/fetch protocol rejects a statement that resolves to nothing with
// "Query did not return any columns", even though the same empty statement
// is silently accepted over jsonlines. "SELECT 1" is a real, valid statement
// on both transports.
const ducklakeOnlyEnvVar = "DUCKDB_DUCKLAKE_ONLY"

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
// persist mirrors devserver's --persist flag: true is today's DuckLake-backed
// mode (db was opened with Options.DuckLake set), and the migrations run
// completely unchanged — ducklakeOnlyEnvVar is left unset, so goose's
// substitution falls back to each statement's in-file default, i.e. the real
// DuckLake DDL. false is the in-memory mode (db was opened with
// Options.DBFile ":memory:" and Options.DuckLake nil): there is no DuckLake
// ATTACH to create the "inngest" catalog schema, so Migrate creates it
// directly, and every DuckLake-only SET SORTED/PARTITIONED BY statement is
// overridden to a no-op via ducklakeOnlyEnvVar, since a bare in-memory
// catalog rejects that syntax outright (confirmed: "Parser Error" against
// real duckdb).
func Migrate(ctx context.Context, db *sql.DB, persist bool) error {
	if !persist {
		if _, err := db.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS "+DuckLakeAlias+";"); err != nil {
			return fmt.Errorf("duckdb: creating in-memory schema %q: %w", DuckLakeAlias, err)
		}
		if err := os.Setenv(ducklakeOnlyEnvVar, "SELECT 1"); err != nil {
			return fmt.Errorf("duckdb: setting %s: %w", ducklakeOnlyEnvVar, err)
		}
		defer func() { _ = os.Unsetenv(ducklakeOnlyEnvVar) }()
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
