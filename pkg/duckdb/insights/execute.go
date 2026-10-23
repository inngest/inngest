package insights

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// Execute's resource limits. The connection pool behind Execute's *sql.DB is
// shared with dual-write (devserver's dualWriteQuackConns, 16), so Insights
// takes only a fraction of it; LIMIT (defaultInsightsLimit) already bounds
// rows, and maxResultBytes bounds their size (e.g. one repeat('a', 2e9)
// cell). Vars rather than consts only so tests can shrink them.
const maxConcurrentExecutes = 4

var (
	executeTimeout       = 30 * time.Second
	maxResultBytes int64 = 32 << 20
	executeSlots         = make(chan struct{}, maxConcurrentExecutes)
)

// limitError rewrites err into a user-facing message when it was caused by
// one of Execute's own limits rather than by the query itself.
func limitError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, driver.ErrResultTooLarge):
		return fmt.Errorf("query result exceeds the %d MiB limit; select fewer columns or rows: %w", maxResultBytes>>20, err)
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return fmt.Errorf("query exceeded the %s time limit: %w", executeTimeout, err)
	}
	return err
}

// ExecutionError wraps a failure that only surfaced once a transpiled query
// actually ran against DuckDB -- as opposed to *ValidationError, which
// rejects a query before it ever reaches the database. A query that parses
// and validates is still not guaranteed to run: this package's static
// checks (tables.go's column allowlist, typecheck.go's inferType, ...) are
// deliberately conservative, so a gap in one of them (a stale allowlist
// entry a macro doesn't actually project, a runtime type conversion this
// package can't see coming, ...) can still let a bad query reach here.
// Start/End are the whole query's own span (TranspileResult.Start/End),
// not a precise position within it: DuckDB's own error text sometimes
// carries a "LINE N: ..." position, but that position is against tr.SQL --
// the rewritten SQL Execute actually runs (macro calls substituted in,
// LIMIT appended, ...) -- not the user's original query text, and this
// package has no source map back from one to the other. Pointing at the
// whole query is honest about what we actually know; a caller (e.g. the
// GQL resolver) still gets a normal, non-fatal Diagnostic out of this
// instead of a bare top-level error, matching *ValidationError's own
// treatment.
type ExecutionError struct {
	Err        error
	Start, End parser.Position
}

func (e *ExecutionError) Error() string { return fmt.Sprintf("insights: %s", e.Err) }
func (e *ExecutionError) Unwrap() error { return e.Err }

// Diagnostic converts e into an ERROR-severity Diagnostic spanning the
// whole query, matching *ValidationError.Diagnostic()'s shape.
func (e *ExecutionError) Diagnostic() Diagnostic {
	return Diagnostic{Start: e.Start, End: e.End, Severity: DiagnosticError, Code: "execution-error", Message: e.Err.Error()}
}

// Column is one column of an executed insights query's result set.
type Column struct {
	Name string
	Type ColumnType
	// PathHints is TranspileResult.ColumnPathHints' entry for this
	// column, positioned the same way -- see buildColumnPathHints' own
	// doc comment for its shape. An entry with an empty Path is this
	// column's own whole-value hint, exactly like
	// knownColumn.pathHints/hint() in tables.go. Nil when this column
	// carries no hint at all.
	PathHints []PathHint
}

// Hint returns c's whole-column hint (its empty-Path PathHints entry),
// HintNone if it has none.
func (c Column) Hint() ColumnHint {
	for _, ph := range c.PathHints {
		if len(ph.Path) == 0 {
			return ph.Hint
		}
	}
	return HintNone
}

// Result is Execute's output: the executed query's columns and every row,
// each cell as the driver's own native decoded Go value (raw, not
// pre-stringified) — exactly the shape pkg/gql_scalars.MarshalUnknown
// expects, so the resolver layer needs no further conversion.
type Result struct {
	Columns []Column
	Rows    [][]any
}

// Execute runs tr's rewritten SQL against db, after checkRenderedSQL confirms
// DuckDB reads it as referencing only what Transpile allows. Column metadata comes from
// rows.ColumnTypes(), not a separate DESCRIBE call — the driver derives it
// unconditionally rather than by sniffing the first returned row, so this
// reports every column correctly even when the query matches zero rows.
// Combines that column list with tr.ColumnPathHints positionally, never
// by name. An index beyond len(tr.ColumnPathHints) (e.g. a UNION query
// past its reconciled column count) gets no hint at all rather than a
// panic or guess.
//
// Every call is bounded (see executeTimeout, maxConcurrentExecutes,
// maxResultBytes): Insights shares its *sql.DB, and so its connection pool
// and DuckDB subprocess, with dual-write.
func Execute(ctx context.Context, db *sql.DB, tr *TranspileResult) (*Result, error) {
	// One deadline covers waiting for a slot and running, so a backlog of
	// queued queries can't grow without bound either. The driver turns the
	// deadline into a real DuckDB interrupt (quack sends a CancelRequest),
	// not just an abandoned wait.
	ctx, cancel := context.WithTimeout(ctx, executeTimeout)
	defer cancel()

	select {
	case executeSlots <- struct{}{}:
		defer func() { <-executeSlots }()
	case <-ctx.Done():
		return nil, &ExecutionError{Err: fmt.Errorf("too many Insights queries are already running; try again shortly"), Start: tr.Start, End: tr.End}
	}

	ctx = driver.WithMaxResultBytes(ctx, maxResultBytes)

	cat := tr.cat
	if cat.tables == nil {
		cat = productCatalog // a hand-built result (tests)
	}
	// Only this mode's table macros pass: product queries can't reach the
	// raw (cross-tenant) ones.
	if err := checkRenderedSQL(ctx, db, tr.SQL, cat); err != nil {
		return nil, &ExecutionError{Err: limitError(ctx, err), Start: tr.Start, End: tr.End}
	}

	rows, err := db.QueryContext(ctx, tr.SQL, tr.Args...)
	if err != nil {
		return nil, &ExecutionError{Err: limitError(ctx, fmt.Errorf("executing query: %w", err)), Start: tr.Start, End: tr.End}
	}
	defer rows.Close()

	columns, err := resultColumns(rows, tr.ColumnPathHints)
	if err != nil {
		return nil, &ExecutionError{Err: err, Start: tr.Start, End: tr.End}
	}

	var out [][]any
	for rows.Next() {
		dest := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range dest {
			ptrs[i] = &dest[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, &ExecutionError{Err: fmt.Errorf("scanning row: %w", err), Start: tr.Start, End: tr.End}
		}
		out = append(out, dest)
	}
	if err := rows.Err(); err != nil {
		return nil, &ExecutionError{Err: fmt.Errorf("reading rows: %w", err), Start: tr.Start, End: tr.End}
	}
	return &Result{Columns: columns, Rows: out}, nil
}

// resultColumns builds Execute's []Column from rows.ColumnTypes(), combined
// with pathHints positionally.
func resultColumns(rows *sql.Rows, pathHints [][]PathHint) ([]Column, error) {
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("reading column types: %w", err)
	}

	columns := make([]Column, len(colTypes))
	for i, ct := range colTypes {
		var colPathHints []PathHint
		if i < len(pathHints) {
			colPathHints = pathHints[i]
		}
		columns[i] = Column{
			Name:      ct.Name(),
			Type:      DuckDBToColumnType(ct.DatabaseTypeName()),
			PathHints: colPathHints,
		}
	}
	return columns, nil
}
