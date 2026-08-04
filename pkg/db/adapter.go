package db

import (
	"context"
	"database/sql"
)

// DBTX is the common query interface implemented by *sql.DB and *sql.Tx.
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

// Dialect identifies a database backend.
type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectPostgres Dialect = "postgres"
	DialectMySQL    Dialect = "mysql"
)

// Adapter bundles everything needed for a database backend.
// Each adapter (SQLite, Postgres, MySQL) implements this interface to provide
// the query layer and transaction support.
//
// Dialect-specific SQL helpers (CEL converters, goqu expressions, time parsing)
// live in the driverhelp subpackage to keep pkg/db free of heavy dependencies.
type Adapter interface {
	// Dialect returns which database backend this adapter targets.
	Dialect() Dialect

	// Q returns the query interface that produces domain row types.
	Q() Querier

	// WithTx creates a new adapter scoped to a database transaction.
	WithTx(ctx context.Context) (TxAdapter, error)

	// Conn returns the underlying *sql.DB for raw queries (e.g. goqu dynamic SQL).
	Conn() *sql.DB

	// RawDB returns the transaction-aware executor for raw queries.
	RawDB() DBTX

	// Close releases any resources held by the adapter.
	Close() error
}

// TxAdapter extends Adapter with transaction commit/rollback.
type TxAdapter interface {
	Adapter

	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
