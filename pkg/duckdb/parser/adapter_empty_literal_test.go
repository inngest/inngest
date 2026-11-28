package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmptyStringLiteralDecodesToEmptyString(t *testing.T) {
	e := parseExprForTest(t, "''")
	lit, ok := e.(*Literal)
	require.True(t, ok, "expected *Literal, got %T", e)
	require.Equal(t, LitString, lit.Kind)
	require.Equal(t, "", lit.Text)
}

func TestEmptyStringIdentifierViaLiteralText(t *testing.T) {
	// Exercises literalText's HasValue branch specifically (unlike the
	// StringLiteral case above, which reads alt.Value directly): an empty
	// string used as an identifier (`''` as a struct key, via
	// ColIdOrString's StringLiteralIdentifier) must decode to "" through
	// literalText, not fall through to its raw-text fallback (which would
	// see nonempty text, the two quote characters).
	e := parseExprForTest(t, "{'': 1}")
	s, ok := e.(*StructExpr)
	require.True(t, ok, "expected *StructExpr, got %T", e)
	require.Equal(t, "", s.Fields[0].Key)
}

func TestEmptyQuotedIdentifierIsRejected(t *testing.T) {
	// DuckDB rejects a zero-length delimited identifier everywhere; an
	// empty alias used to be accepted and then dropped by Write.
	for _, sql := range []string{`SELECT 1 AS ""`, `SELECT "" FROM t`, `SELECT * FROM t AS ""`, `WITH "" AS (SELECT 1) SELECT 1`} {
		_, err := ParseString(sql)
		var perr *ParseError
		require.ErrorAs(t, err, &perr, sql)
		require.Equal(t, "zero-length delimited identifier", perr.Message, sql)
	}
	// An escaped quote isn't empty.
	stmt, err := ParseString(`SELECT """" FROM t`)
	require.NoError(t, err)
	require.Equal(t, []string{`"`}, stmt.Columns[0].Expr.(*Ident).Parts)
}
