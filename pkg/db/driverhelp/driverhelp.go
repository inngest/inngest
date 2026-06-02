// Package driverhelp defines the DialectHelpers interface for dialect-specific
// SQL behavior. It lives in its own subpackage so that pkg/db stays free of
// heavy dependencies like pkg/run and goqu.
package driverhelp

import (
	"time"

	sq "github.com/doug-martin/goqu/v9"
	sqexp "github.com/doug-martin/goqu/v9/exp"
	"github.com/inngest/inngest/pkg/run"
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
