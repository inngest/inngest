package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitExcludeName(t *testing.T) {
	require.Equal(t, []string{"a"}, SplitExcludeName("a"))
	require.Equal(t, []string{"t", "a"}, SplitExcludeName("t.a"))
	require.Equal(t, []string{"T", "a.b"}, SplitExcludeName(`"T"."a.b"`))
	require.Equal(t, []string{`a"b`}, SplitExcludeName(`"a""b"`))
}

// An EXCLUDE entry is written back part by part, so a qualified or quoted
// entry keeps its meaning instead of being re-quoted as one identifier.
func TestWriteStarExcludeQualifiedAndQuoted(t *testing.T) {
	stmt, err := ParseString(`SELECT * EXCLUDE (t.a, "B", "x.y") FROM t`)
	require.NoError(t, err)
	require.Equal(t, `SELECT * EXCLUDE (t.a, B, "x.y") FROM t`, String(stmt))
}
