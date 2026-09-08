package parser

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/inngest/inngest/pkg/duckdb/parser/peg"
)

// testAlloc stands in for the evaluator's arena allocator in tests that
// call a peg.Primitive directly rather than through a full Parse.
func testAlloc(n peg.Node) *peg.Node { return &n }

func testKeywordSets() *KeywordSets {
	return NewKeywordSets(
		[]string{"ALL", "SELECT", "AND"}, // reserved
		[]string{"ABORT"},                // unreserved
		[]string{"BETWEEN"},              // column_name
		[]string{"ASOF"},                 // func_name
		[]string{"INT"},                  // type_name
	)
}

func TestStrictIdentifierPrimitive(t *testing.T) {
	prims := Primitives(testKeywordSets())
	ident := prims["TableName"]

	n, next, ok := ident(testAlloc, "orders", 0)
	require.True(t, ok)
	require.Equal(t, "orders", n.Value)
	require.Equal(t, 6, next)

	// unreserved keyword is fine as an identifier
	n, _, ok = ident(testAlloc, "abort", 0)
	require.True(t, ok)
	require.Equal(t, "abort", n.Value)

	// a reserved keyword is NOT fine, unquoted
	_, _, ok = ident(testAlloc, "select", 0)
	require.False(t, ok)

	// quoted identifiers are always fine, even reserved words, with "" unescaping
	n, next, ok = ident(testAlloc, `"say ""hi"""`, 0)
	require.True(t, ok)
	require.Equal(t, `say "hi"`, n.Value)
	require.Equal(t, len(`"say ""hi"""`), next)
}

func TestPermissiveIdentifierPrimitive(t *testing.T) {
	prims := Primitives(testKeywordSets())
	reserved := prims["ReservedTableName"]

	n, _, ok := reserved(testAlloc, "select", 0)
	require.True(t, ok)
	require.Equal(t, "select", n.Value)
}

func TestKeywordCategoryPrimitive(t *testing.T) {
	prims := Primitives(testKeywordSets())
	unreserved := prims["UnreservedKeyword"]

	n, _, ok := unreserved(testAlloc, "ABORT", 0)
	require.True(t, ok)
	require.Equal(t, "ABORT", n.Value)

	_, _, ok = unreserved(testAlloc, "orders", 0)
	require.False(t, ok)
}

func TestNumberLiteralPrimitive(t *testing.T) {
	prims := Primitives(testKeywordSets())
	num := prims["NumberLiteral"]

	for _, tc := range []string{"42", "-3.14", "+7", "1.5e10", "1E-3"} {
		n, next, ok := num(testAlloc, tc, 0)
		require.Truef(t, ok, "expected %q to match", tc)
		require.Equal(t, tc, n.Text)
		require.Equal(t, len(tc), next)
	}
	_, _, ok := num(testAlloc, "abc", 0)
	require.False(t, ok)
}

func TestStringLiteralPrimitive(t *testing.T) {
	prims := Primitives(testKeywordSets())
	str := prims["StringLiteral"]

	n, next, ok := str(testAlloc, `'it''s here'`, 0)
	require.True(t, ok)
	require.Equal(t, "it's here", n.Value)
	require.Equal(t, len(`'it''s here'`), next)
}

func TestOperatorLiteralPrimitive(t *testing.T) {
	prims := Primitives(testKeywordSets())
	op := prims["OperatorLiteral"]

	n, next, ok := op(testAlloc, "<=> rest", 0)
	require.True(t, ok)
	require.Equal(t, "<=>", n.Text)
	require.Equal(t, 3, next)
}

func TestEndOfInputPrimitive(t *testing.T) {
	prims := Primitives(testKeywordSets())
	eof := prims["EndOfInput"]

	_, _, ok := eof(testAlloc, "", 0)
	require.True(t, ok)
	_, _, ok = eof(testAlloc, "x", 0)
	require.False(t, ok)
}

func TestSkipSQLTrivia(t *testing.T) {
	input := "  -- a comment\n/* block\n comment */  select"
	pos := SkipSQLTrivia(input, 0)
	require.Equal(t, "select", input[pos:])
}
