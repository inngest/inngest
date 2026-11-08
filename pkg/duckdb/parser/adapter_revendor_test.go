package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// These tests exercise the grammar positions Task 15's second-commit
// re-vendor changed cardinality on (single/'?' -> '*'/'+', plus one brand
// new alternative) — DuckDB added support for deeply nested catalog/schema
// qualification chains. None of this package's other fixtures use more
// than one level of schema qualification, so these cases would otherwise
// go completely unexercised: a passing `go test ./...` run doesn't by
// itself prove a re-vendor's cardinality changes were handled correctly,
// only that nothing *already covered* regressed. See Task 15's plan entry.

func TestAdaptNestedSchemaQualifiedTableRef(t *testing.T) {
	// CatalogReservedSchemaTable's ReservedSchemaQualification widened
	// from a single mandatory match to '+'.
	stmt, err := ParseString("SELECT * FROM cat.sch.tbl")
	require.NoError(t, err)
	ref := stmt.From.Refs[0].(*BaseTableRef)
	require.Equal(t, []string{"cat", "sch", "tbl"}, ref.Name)

	stmt, err = ParseString("SELECT * FROM cat.s1.s2.tbl")
	require.NoError(t, err)
	ref = stmt.From.Refs[0].(*BaseTableRef)
	require.Equal(t, []string{"cat", "s1", "s2", "tbl"}, ref.Name)
}

func TestAdaptNestedSchemaQualifiedTableFunction(t *testing.T) {
	// QualifiedTableFunction's SchemaQualification widened from '?' to '*'.
	stmt, err := ParseString("SELECT * FROM cat.sch.range(10)")
	require.NoError(t, err)
	ref := stmt.From.Refs[0].(*TableFunctionRef)
	require.Equal(t, []string{"cat", "sch", "range"}, ref.Name)
}

func TestAdaptNestedSchemaQualifiedFunctionCall(t *testing.T) {
	// CatalogReservedSchemaFunctionName's ReservedSchemaQualification
	// widened from '?' to '*'.
	e := parseExprForTest(t, "cat.sch.func(x)")
	f, ok := e.(*FunctionExpr)
	require.True(t, ok)
	require.Equal(t, []string{"cat", "sch", "func"}, f.Name)
}

func TestAdaptNestedSchemaTableColumnName(t *testing.T) {
	// NestedSchemaTableColumnName is a brand new ColumnReference
	// alternative (5+ component references deeper than catalog+schema).
	e := parseExprForTest(t, "cat.s1.s2.s3.col")
	id, ok := e.(*Ident)
	require.True(t, ok)
	require.Equal(t, []string{"cat", "s1", "s2", "s3", "col"}, id.Parts)
}
