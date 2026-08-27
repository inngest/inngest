// Package duckdbquery implements a DuckDB-backed cqrs.Manager decorator for
// the GQL runs/events/run-trace read paths, reading the tables
// pkg/execution/dualwrite writes into (inngest.runs, inngest.run_trace_spans,
// inngest.events). See docs/plans/007-duckdb-gql-resolvers.md.
//
// Manager embeds the real cqrs.Manager (SQLite/Postgres-backed) and
// overrides only the methods this package can answer from DuckDB; every
// other method — including LegacyGetSpanOutput, which the current DuckDB
// schema fundamentally cannot answer — falls through to the embedded
// manager unchanged.
package duckdbquery

import (
	"database/sql"

	"github.com/inngest/inngest/pkg/cqrs"
)

type Manager struct {
	cqrs.Manager
	db *sql.DB
}

// Wrap returns a cqrs.Manager that reads runs/events/run-trace data from
// db (a DuckDB connection dual-write already writes through) and falls
// through to underlying for everything else.
func Wrap(underlying cqrs.Manager, db *sql.DB) cqrs.Manager {
	return &Manager{Manager: underlying, db: db}
}

// FlatSpans reports that spans this manager's GetSpansByRunID returns come
// from a flat (non-dynamic, non-fragment-merged) tree — see
// pkg/coreapi/graph/loaders' flatSpanSource marker interface, which uses
// this to select the simplified GQL span converter.
func (m *Manager) FlatSpans() bool { return true }
