// Package duckdbquery implements a DuckDB-backed cqrs.Manager decorator for
// the GQL runs/events/run-trace read paths, reading the tables
// pkg/execution/dualwrite writes into (inngest.runs, inngest.run_trace_spans,
// inngest.events).
//
// Manager embeds the real cqrs.Manager and overrides only the methods this
// package can answer from DuckDB; everything else — including
// LegacyGetSpanOutput, which the current DuckDB schema can't answer — falls
// through unchanged.
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

// FlatSpans reports a flat (non-dynamic, non-fragment-merged) span tree —
// see pkg/coreapi/graph/loaders' flatSpanSource, which selects the
// simplified GQL span converter based on this.
func (m *Manager) FlatSpans() bool { return true }
