// pkg/duckdb/parser/adapter_from_test.go
package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdaptInnerJoinOn(t *testing.T) {
	stmt, err := ParseString("SELECT * FROM a JOIN b ON a.id = b.id")
	require.NoError(t, err)
	join, ok := stmt.From.Refs[0].(*JoinRef)
	require.True(t, ok)
	require.Equal(t, "INNER", join.Type)
	require.NotNil(t, join.On)
	_, ok = join.Left.(*BaseTableRef)
	require.True(t, ok)
	_, ok = join.Right.(*BaseTableRef)
	require.True(t, ok)
}

func TestAdaptLeftOuterJoinUsing(t *testing.T) {
	stmt, err := ParseString("SELECT * FROM a LEFT OUTER JOIN b USING (id)")
	require.NoError(t, err)
	join := stmt.From.Refs[0].(*JoinRef)
	require.Equal(t, "LEFT OUTER", join.Type)
	require.Equal(t, []string{"id"}, join.Using)
}

func TestAdaptCrossAndNaturalJoin(t *testing.T) {
	stmt, err := ParseString("SELECT * FROM a CROSS JOIN b")
	require.NoError(t, err)
	require.Equal(t, "CROSS", stmt.From.Refs[0].(*JoinRef).Type)

	stmt, err = ParseString("SELECT * FROM a NATURAL LEFT JOIN b")
	require.NoError(t, err)
	require.Equal(t, "NATURAL LEFT", stmt.From.Refs[0].(*JoinRef).Type)
}

func TestAdaptSemiAntiJoin(t *testing.T) {
	stmt, err := ParseString("SELECT * FROM a SEMI JOIN b ON a.id = b.id")
	require.NoError(t, err)
	require.Equal(t, "SEMI", stmt.From.Refs[0].(*JoinRef).Type)
}

func TestAdaptChainedJoins(t *testing.T) {
	stmt, err := ParseString("SELECT * FROM a JOIN b ON a.id = b.id JOIN c ON b.id = c.id")
	require.NoError(t, err)
	outer, ok := stmt.From.Refs[0].(*JoinRef)
	require.True(t, ok)
	_, ok = outer.Left.(*JoinRef) // left-associative: (a JOIN b) JOIN c
	require.True(t, ok)
}

func TestAdaptSubqueryInFrom(t *testing.T) {
	stmt, err := ParseString("SELECT * FROM (SELECT id FROM t) sub")
	require.NoError(t, err)
	sub, ok := stmt.From.Refs[0].(*TableSubqueryRef)
	require.True(t, ok)
	require.Equal(t, "sub", sub.Alias)
	require.NotNil(t, sub.Select)
}

func TestAdaptLateralSubquery(t *testing.T) {
	stmt, err := ParseString("SELECT * FROM a, LATERAL (SELECT * FROM b WHERE b.id = a.id) sub")
	require.NoError(t, err)
	sub, ok := stmt.From.Refs[1].(*TableSubqueryRef)
	require.True(t, ok)
	require.True(t, sub.Lateral)
}

func TestAdaptTableFunction(t *testing.T) {
	stmt, err := ParseString("SELECT * FROM range(10) t")
	require.NoError(t, err)
	ref, ok := stmt.From.Refs[0].(*TableFunctionRef)
	require.True(t, ok)
	require.Equal(t, []string{"range"}, ref.Name)
	require.Len(t, ref.Args, 1)
	require.Equal(t, "t", ref.Alias)
}

func TestAdaptParensTableRef(t *testing.T) {
	stmt, err := ParseString("SELECT * FROM (a JOIN b ON a.id = b.id) x")
	require.NoError(t, err)
	pt, ok := stmt.From.Refs[0].(*ParensTableRef)
	require.True(t, ok)
	require.Equal(t, "x", pt.Alias)
	_, ok = pt.Ref.(*JoinRef)
	require.True(t, ok)
}

func TestAdaptMultipleFromRefsAreImplicitCrossJoin(t *testing.T) {
	stmt, err := ParseString("SELECT * FROM a, b")
	require.NoError(t, err)
	require.Len(t, stmt.From.Refs, 2)
}
