package db

import (
	"context"
	"time"

	sq "github.com/doug-martin/goqu/v9"
	sqexp "github.com/doug-martin/goqu/v9/exp"
	"github.com/inngest/inngest/pkg/run"
)

// Dialect identifies a database backend.
type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectPostgres Dialect = "postgres"
	DialectMySQL    Dialect = "mysql"
)

// DialectHelpers provides dialect-specific SQL behavior for dynamic queries.
// These methods encapsulate the differences between SQL dialects that cannot
// be handled by sqlc code generation alone (e.g., goqu dialect strings,
// JSON unnesting, time parsing).
type DialectHelpers interface {
	// GoquDialect returns the goqu dialect identifier ("sqlite3", "postgres", "mysql").
	GoquDialect() string

	// CELConverter returns the dialect-specific CEL-to-SQL expression converter.
	CELConverter() run.ExprSQLConverter

	// EventIDsExpr returns the goqu expression for aggregating event IDs in span queries.
	EventIDsExpr() sqexp.Expression

	// BuildEventJoin adds the dialect-specific event join to a span query.
	BuildEventJoin(q *sq.SelectDataset) *sq.SelectDataset

	// ParseEventIDs parses a raw event IDs string from the database into a string slice.
	// SQLite stores plain JSON arrays; Postgres stores double-encoded JSON strings.
	ParseEventIDs(raw *string) []string

	// ParseTime parses a time string from the database.
	// SQLite uses Go's time.Time string format; Postgres uses RFC3339Nano.
	ParseTime(s string) (time.Time, error)
}

// Adapter bundles everything needed for a database backend.
// Each adapter (SQLite, Postgres, MySQL) implements this interface to provide
// the query layer and dialect-specific helpers.
type Adapter interface {
	// Dialect returns which database backend this adapter targets.
	Dialect() Dialect

	// Querier returns the query interface that produces domain row types.
	Q() Querier

	// Helpers returns dialect-specific SQL helpers for dynamic query building.
	Helpers() DialectHelpers

	// WithTx creates a new adapter scoped to a database transaction.
	WithTx(ctx context.Context) (TxAdapter, error)

	// Close releases any resources held by the adapter.
	Close() error
}

// TxAdapter extends Adapter with transaction commit/rollback.
type TxAdapter interface {
	Adapter

	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
