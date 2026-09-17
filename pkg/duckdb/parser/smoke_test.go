package parser

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/inngest/inngest/pkg/duckdb/parser/grammar"
	"github.com/inngest/inngest/pkg/duckdb/parser/peg"
)

func newSmokeParser(t *testing.T) *peg.Parser {
	t.Helper()
	g, kl, err := grammar.Load()
	require.NoError(t, err)
	ks := NewKeywordSets(kl.Reserved, kl.Unreserved, kl.ColumnName, kl.FuncName, kl.TypeName)
	return &peg.Parser{Grammar: g, Primitives: Primitives(ks), SkipTrivia: SkipSQLTrivia}
}

func TestSmokeParseMinimalSelect(t *testing.T) {
	p := newSmokeParser(t)
	for _, sql := range []string{
		"SELECT 1",
		"SELECT a, b FROM t WHERE a = 1",
		"select * from orders",
		"SELECT a FROM t WHERE a IN (1, 2, 3) ORDER BY a LIMIT 10",
	} {
		t.Run(sql, func(t *testing.T) {
			n, err := p.Parse(sql, "SelectStatement")
			require.NoError(t, err)
			require.Equal(t, "SelectStatement", n.Name)
		})
	}
}

func TestSmokeParseRejectsGarbage(t *testing.T) {
	p := newSmokeParser(t)
	_, err := p.Parse("not sql at all !!!", "SelectStatement")
	require.Error(t, err)
}
