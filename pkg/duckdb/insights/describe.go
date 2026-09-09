package insights

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// ShortCircuitResult is what TryShortCircuit returns for a recognized
// schema-introspection statement (SHOW TABLES, DESCRIBE <table>) in place
// of a real Transpile+Execute round trip. Tables backs
// InsightsQueryInfo.Tables the same way TranspileResult.Tables does; there
// is no PrimaryTable/ColumnHints/Diagnostics/Limited -- none of those
// concepts apply to a query that never went through the pipeline.
type ShortCircuitResult struct {
	Result *Result
	Tables []string
}

// TryShortCircuit recognizes SHOW TABLES and DESCRIBE <table> and answers
// both entirely from this package's own static logicalTables registry --
// the same source of truth validate/buildColumnHints/remapTables already
// treat as authoritative -- with no SQL parsing, no rewrite, and no
// database round trip at all. ok is false for anything else (including a
// malformed DESCRIBE with no table name), so the caller falls through to
// the normal Transpile+Execute path and gets DuckDB's/this package's own
// error reporting instead of a confusing pass-through failure here.
func TryShortCircuit(sql string) (result *ShortCircuitResult, ok bool, err error) {
	stmt := strings.TrimSuffix(strings.TrimSpace(sql), ";")
	upper := strings.ToUpper(strings.TrimSpace(stmt))

	if upper == "SHOW TABLES" {
		return showTables(), true, nil
	}

	const describePrefix = "DESCRIBE "
	if !strings.HasPrefix(upper, describePrefix) {
		return nil, false, nil
	}
	name := strings.TrimSpace(stmt[len(describePrefix):])
	if name == "" {
		return nil, false, nil
	}
	result, err = describeTable(name)
	return result, true, err
}

func showTables() *ShortCircuitResult {
	names := slices.Sorted(maps.Keys(logicalTables))
	rows := make([][]any, len(names))
	for i, n := range names {
		rows[i] = []any{n, logicalTables[n].description}
	}
	return &ShortCircuitResult{
		Result: &Result{
			Columns: []Column{
				{Name: "name", Type: ColumnTypeString},
				{Name: "description", Type: ColumnTypeString},
			},
			Rows: rows,
		},
		Tables: names,
	}
}

// describeTable reports name's columns in the exact order its macro
// projects them (columnOrder) — the same order a SELECT * against it would
// use — along with each column's GQL-facing type bucket and description.
// Unknown table names are rejected the same way an unknown FROM-clause
// reference is (scope.go's addBaseTable) -- a *ValidationError, so the
// resolver renders it as a query diagnostic rather than a bare error,
// consistent with every other rejected-table case in this package.
func describeTable(name string) (*ShortCircuitResult, error) {
	schema, ok := LookupTableSchema(name)
	if !ok {
		return nil, &ValidationError{Message: fmt.Sprintf("unknown table %q", name)}
	}

	rows := make([][]any, len(schema.Columns))
	for i, col := range schema.Columns {
		rows[i] = []any{col.Name, col.Type.String(), col.Description}
	}
	return &ShortCircuitResult{
		Result: &Result{
			Columns: []Column{
				{Name: "column_name", Type: ColumnTypeString},
				{Name: "column_type", Type: ColumnTypeString},
				{Name: "description", Type: ColumnTypeString},
			},
			Rows: rows,
		},
		Tables: []string{schema.Name},
	}, nil
}

// TableSchema is one logical table's exported schema, columns in the same
// order a SELECT * against it returns them -- backs DESCRIBE's
// short-circuited output above and cmd/gen-insights-schema's JSON dump,
// embedded into the SQL editor UI at build time rather than fetched live.
type TableSchema struct {
	Name        string
	Description string
	Columns     []ColumnSchema
}

// ColumnSchema is one column of a TableSchema.
type ColumnSchema struct {
	Name        string
	Type        ColumnType
	Hint        ColumnHint
	Description string
}

// LookupTableSchema returns name's schema (case-insensitive), or ok=false
// if name isn't one of the six logical tables this package exposes.
func LookupTableSchema(name string) (TableSchema, bool) {
	tbl, ok := logicalTables[strings.ToLower(name)]
	if !ok {
		return TableSchema{}, false
	}
	return newTableSchema(tbl), true
}

// AllTableSchemas returns every logical table's schema, sorted by name --
// used by cmd/gen-insights-schema to dump the whole registry at once.
func AllTableSchemas() []TableSchema {
	names := slices.Sorted(maps.Keys(logicalTables))
	schemas := make([]TableSchema, len(names))
	for i, name := range names {
		schemas[i] = newTableSchema(logicalTables[name])
	}
	return schemas
}

func newTableSchema(tbl logicalTable) TableSchema {
	cols := make([]ColumnSchema, len(tbl.columnOrder))
	for i, colName := range tbl.columnOrder {
		col := tbl.columns[colName]
		cols[i] = ColumnSchema{Name: colName, Type: col.colType, Hint: col.hint, Description: col.description}
	}
	return TableSchema{Name: tbl.name, Description: tbl.description, Columns: cols}
}
