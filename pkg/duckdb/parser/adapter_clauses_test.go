// pkg/duckdb/parser/adapter_clauses_test.go
package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdaptDistinct(t *testing.T) {
	stmt, err := ParseString("SELECT DISTINCT a FROM t")
	require.NoError(t, err)
	require.NotNil(t, stmt.Distinct)
	require.Nil(t, stmt.Distinct.On)
}

func TestAdaptDistinctOn(t *testing.T) {
	stmt, err := ParseString("SELECT DISTINCT ON (a, b) a, b, c FROM t")
	require.NoError(t, err)
	require.Len(t, stmt.Distinct.On, 2)
}

func TestAdaptSelectAllIsNotDistinct(t *testing.T) {
	stmt, err := ParseString("SELECT ALL a FROM t")
	require.NoError(t, err)
	require.Nil(t, stmt.Distinct)
}

func TestAdaptGroupByList(t *testing.T) {
	stmt, err := ParseString("SELECT a, count(*) FROM t GROUP BY a, b")
	require.NoError(t, err)
	require.Len(t, stmt.GroupBy.Items, 2)
	require.False(t, stmt.GroupBy.All)
	_, ok := stmt.GroupBy.Items[0].(*GroupByExprItem)
	require.True(t, ok)
}

func TestAdaptGroupByAll(t *testing.T) {
	stmt, err := ParseString("SELECT a FROM t GROUP BY ALL")
	require.NoError(t, err)
	require.True(t, stmt.GroupBy.All)
}

func TestAdaptGroupByCubeRollupAndGroupingSets(t *testing.T) {
	stmt, err := ParseString("SELECT a FROM t GROUP BY CUBE (a, b), ROLLUP (c), GROUPING SETS ((a), (b), ())")
	require.NoError(t, err)
	require.Len(t, stmt.GroupBy.Items, 3)
	cube, ok := stmt.GroupBy.Items[0].(*GroupByCube)
	require.True(t, ok)
	require.Len(t, cube.Items, 2)
	rollup, ok := stmt.GroupBy.Items[1].(*GroupByRollup)
	require.True(t, ok)
	require.Len(t, rollup.Items, 1)
	sets, ok := stmt.GroupBy.Items[2].(*GroupingSets)
	require.True(t, ok)
	require.Len(t, sets.Sets, 3)
	_, ok = sets.Sets[2].(*GroupByEmpty)
	require.True(t, ok)
}

func TestAdaptHaving(t *testing.T) {
	stmt, err := ParseString("SELECT a, count(*) FROM t GROUP BY a HAVING count(*) > 1")
	require.NoError(t, err)
	require.NotNil(t, stmt.Having)
}

func TestAdaptOrderByAll(t *testing.T) {
	stmt, err := ParseString("SELECT a FROM t ORDER BY ALL DESC")
	require.NoError(t, err)
	require.True(t, stmt.OrderBy.All)
	require.True(t, stmt.OrderBy.Items[0].Desc)
}

func TestAdaptLimitOffset(t *testing.T) {
	stmt, err := ParseString("SELECT a FROM t LIMIT 10 OFFSET 5")
	require.NoError(t, err)
	require.Equal(t, "10", stmt.Limit.Limit.(*Literal).Text)
	require.Equal(t, "5", stmt.Limit.Offset.(*Literal).Text)
}

func TestAdaptOffsetFetch(t *testing.T) {
	stmt, err := ParseString("SELECT a FROM t OFFSET 5 FETCH FIRST 10 ROWS ONLY")
	require.NoError(t, err)
	require.Equal(t, "10", stmt.Limit.Limit.(*Literal).Text)
	require.Equal(t, "5", stmt.Limit.Offset.(*Literal).Text)
}

func TestAdaptLimitPercent(t *testing.T) {
	stmt, err := ParseString("SELECT a FROM t LIMIT 10 PERCENT")
	require.NoError(t, err)
	require.True(t, stmt.Limit.Percent)
}
