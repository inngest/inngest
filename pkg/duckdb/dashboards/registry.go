// Package dashboards is a query-key registry for backend-composed,
// dashboard-shaped queries against the DuckDB dual-write store (the
// `inngest.runs` table `pkg/cqrs/duckdbquery` already reads from). It
// mirrors the reference Cloud pkg/applogic/dashboards package's shape -- a
// map of named query keys to SQL builder functions, dispatched through one
// generic entrypoint -- adapted from that package's ClickHouse Insights
// runner to direct parameterized DuckDB SQL, since there's no ad-hoc-SQL
// pipeline to route a backend-composed (not user-written) query through
// here.
//
// This file holds the generic machinery every entry shares (the registry
// map, Scope, Result, Get). Domain-specific keys and SQL builders live in
// their own file -- see sessions.go for the Sessions entries, the first
// registrants.
package dashboards

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Scope carries the caller-provided parameters for one Get call.
// AccountID/EnvID/From/To/Limit/Cursor are read by every entry; SessionKey/
// SessionID/Search are only meaningful to the session-scoped entries
// (sessions.go) and ignored by any entry that doesn't need them.
type Scope struct {
	AccountID uuid.UUID
	EnvID     uuid.UUID
	From, To  time.Time
	Limit     int
	Cursor    string

	SessionKey string
	SessionID  string
	Search     string
}

// Result is one query's raw output: column names and every row's values,
// in the driver's own native decoded Go form (not stringified the way the
// reference's HTTP-transport InsightsResult is) -- Get runs in-process
// against the same *sql.DB pkg/cqrs/duckdbquery already reads from, so
// there's no transport boundary forcing a lossy string encoding here.
type Result struct {
	Columns []string
	Rows    [][]any
}

// registryEntry is one registered query: its SQL builder, returning the
// parameterized query text and its bound args.
type registryEntry struct {
	build func(scope Scope) (string, []any, error)
}

// registry maps every known query key to its definition. Domain-specific
// entries are added here from their own file (sessions.go's KeySession*
// constants below), following the reference registry's single-map
// convention rather than one map per domain.
var registry = map[string]registryEntry{
	KeySessionKeys: {build: buildSessionKeysSQL},
	KeySessions:    {build: buildSessionsSQL},
	KeySessionRuns: {build: buildSessionRunsSQL},
}

const (
	defaultLimit = 20
	maxLimit     = 100
)

// Get runs the registry query for one key. An unrecognized key returns
// (nil, nil) rather than erroring -- keys are plain strings nothing
// upstream validates ahead of time, matching the reference Get's own
// contract.
func Get(ctx context.Context, db *sql.DB, scope Scope, key string) (*Result, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}

	entry, ok := registry[key]
	if !ok {
		return nil, nil
	}

	query, args, err := entry.build(scope)
	if err != nil {
		return nil, fmt.Errorf("dashboard query (%s): %w", key, err)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("dashboard query (%s): %w", key, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("dashboard query (%s): reading columns: %w", key, err)
	}

	var out [][]any
	for rows.Next() {
		dest := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range dest {
			ptrs[i] = &dest[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("dashboard query (%s): scanning row: %w", key, err)
		}
		out = append(out, dest)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dashboard query (%s): reading rows: %w", key, err)
	}

	return &Result{Columns: cols, Rows: out}, nil
}

func validateScope(scope Scope) error {
	if scope.AccountID == uuid.Nil || scope.EnvID == uuid.Nil {
		return fmt.Errorf("dashboard query: accountID and envID are required")
	}
	if !scope.From.IsZero() && !scope.To.IsZero() && !scope.From.Before(scope.To) {
		return fmt.Errorf("dashboard query: invalid time range: from (%s) must be before to (%s)",
			scope.From.Format(time.RFC3339), scope.To.Format(time.RFC3339))
	}
	return nil
}

func resolveLimit(requested int) int {
	if requested <= 0 {
		return defaultLimit
	}
	if requested > maxLimit {
		return maxLimit
	}
	return requested
}

// columnIndex maps a Result's column names to their position, so callers
// that parse rows by name degrade gracefully (rather than reading the
// wrong position) if a builder's SELECT list is ever reordered.
func columnIndex(columns []string) map[string]int {
	m := make(map[string]int, len(columns))
	for i, c := range columns {
		m[c] = i
	}
	return m
}

func col(row []any, idx map[string]int, name string) any {
	if i, ok := idx[name]; ok && i < len(row) {
		return row[i]
	}
	return nil
}
