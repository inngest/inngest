// Package result holds what the driver's transports (internal/quack,
// internal/jsonlines) hand back to the driver: positional result rows, and
// the error classes the driver's process supervisor uses to decide between
// surfacing a failure, respawning the subprocess, or both.
package result

import "errors"

// ErrStatementFailed marks an error DuckDB reported for the statement itself:
// a constraint violation, a type/conversion failure, or schema drift. The
// jsonlines CLI writes those to stderr and *still* completes its canary round
// trip (verified empirically), so before this existed the driver reported
// success for statements DuckDB had actually rejected — silent data loss for
// the most likely real failure mode of the dual-write path. The subprocess is
// healthy when this is returned, so the driver must neither restart nor
// retry.
var ErrStatementFailed = errors.New("duckdb: statement failed")

// ErrSessionDesynced marks a jsonlines session that abandoned a statement
// mid-read (the only way that happens is ctx cancellation inside Exec/Query).
// The subprocess's remaining output for that statement is still queued, so
// the session can no longer correlate output with statements and must be
// replaced by a fresh subprocess.
var ErrSessionDesynced = errors.New("duckdb: session abandoned an in-flight statement")

// Columns is a result's column names in the query's own left-to-right
// order, plus an index from each name to its first position. Every row of
// one result shares a single *Columns.
//
// Values live positionally in Row.Vals rather than in a map keyed by name,
// so a query that repeats a column name (SELECT a.x, b.x, or SELECT * over
// a join) keeps every value; by-name lookup (Row.Get) resolves to the first
// column of that name, which is all the driver's own internal lookups need.
type Columns struct {
	Names []string
	index map[string]int
}

func NewColumns(names []string) *Columns {
	index := make(map[string]int, len(names))
	for i, name := range names {
		if _, ok := index[name]; !ok {
			index[name] = i
		}
	}
	return &Columns{Names: names, index: index}
}

// Row is one result row: Vals[i] is the value of Cols.Names[i].
type Row struct {
	Cols *Columns
	Vals []any
}

// Get returns the value of the first column named name, or nil if there is
// no such column.
func (r Row) Get(name string) any {
	if r.Cols == nil {
		return nil
	}
	i, ok := r.Cols.index[name]
	if !ok || i >= len(r.Vals) {
		return nil
	}
	return r.Vals[i]
}

// Names returns the row's column names, for diagnostics.
func (r Row) Names() []string {
	if r.Cols == nil {
		return nil
	}
	return r.Cols.Names
}
