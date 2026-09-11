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

func TestEmptyQuotedIdentifierViaLiteralText(t *testing.T) {
	// Exercises literalText's HasValue branch specifically (unlike the
	// StringLiteral case above, which reads alt.Value directly): an empty
	// quoted identifier `""` must decode to "" through identifierPrimitive
	// + literalText, not fall through to literalText's raw-text fallback
	// (which would see nonempty text, the two quote characters).
	e := parseExprForTest(t, `""`)
	id, ok := e.(*Ident)
	require.True(t, ok, "expected *Ident, got %T", e)
	require.Equal(t, []string{""}, id.Parts)
}
