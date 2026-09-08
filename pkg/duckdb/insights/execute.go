package insights

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

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
	Hint ColumnHint
}

// Result is Execute's output: the executed query's columns and every row,
// each cell as the driver's own native decoded Go value (raw, not
// pre-stringified) — exactly the shape pkg/gql_scalars.MarshalUnknown
// expects, so the resolver layer needs no further conversion.
type Result struct {
	Columns []Column
	Rows    [][]any
}

// Execute runs tr's rewritten SQL against db. Column metadata comes from
// rows.ColumnTypes(), not a separate DESCRIBE call — the driver derives it
// unconditionally rather than by sniffing the first returned row, so this
// reports every column correctly even when the query matches zero rows.
// Combines that column list with tr.ColumnHints positionally, never by
// name. An index beyond len(tr.ColumnHints) (e.g. a UNION query, whose
// ColumnHints is always empty) gets HintNone rather than a panic or guess.
func Execute(ctx context.Context, db *sql.DB, tr *TranspileResult) (*Result, error) {
	rows, err := db.QueryContext(ctx, tr.SQL, tr.Args...)
	if err != nil {
		return nil, &ExecutionError{Err: fmt.Errorf("executing query: %w", err), Start: tr.Start, End: tr.End}
	}
	defer rows.Close()

	columns, err := resultColumns(rows, tr.ColumnHints)
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
// with hints positionally.
func resultColumns(rows *sql.Rows, hints []ColumnHint) ([]Column, error) {
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("reading column types: %w", err)
	}

	columns := make([]Column, len(colTypes))
	for i, ct := range colTypes {
		hint := HintNone
		if i < len(hints) {
			hint = hints[i]
		}
		columns[i] = Column{Name: ct.Name(), Type: DuckDBToColumnType(ct.DatabaseTypeName()), Hint: hint}
	}
	return columns, nil
}
