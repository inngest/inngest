package db

import (
	"context"
	"database/sql"
)

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
	// Note: this is always the bare connection and ignores any transaction
	// scope -- callers running raw SQL inside a tx should use ExecContext.
	Conn() *sql.DB

	// ExecContext runs a raw parameterised SQL statement against the adapter's
	// current execution scope. The plain Adapter routes to its *sql.DB; a
	// TxAdapter routes to its active *sql.Tx so dynamically-built SQL (goqu
	// bulk inserts, etc.) participates in the transaction.
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)

	// Close releases any resources held by the adapter.
	Close() error
}

// TxAdapter extends Adapter with transaction commit/rollback.
type TxAdapter interface {
	Adapter

	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
