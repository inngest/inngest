package duckdb

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var MigrationsFS embed.FS

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
func Migrate(ctx context.Context, db *sql.DB) error {
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
