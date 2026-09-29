// pkg/duckdb/parser/adapter_cte_test.go
package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdaptSimpleCTE(t *testing.T) {
	stmt, err := ParseString("WITH cte AS (SELECT a FROM t) SELECT a FROM cte")
	require.NoError(t, err)
	require.NotNil(t, stmt.With)
	require.Len(t, stmt.With.CTEs, 1)
	require.Equal(t, "cte", stmt.With.CTEs[0].Name)
	require.NotNil(t, stmt.With.CTEs[0].Select)
	require.False(t, stmt.With.Recursive)
	// the outer statement itself is a plain select referencing the CTE by name
	require.Equal(t, SetOpNone, stmt.SetOp)
	ref := stmt.From.Refs[0].(*BaseTableRef)
	require.Equal(t, []string{"cte"}, ref.Name)
}

func TestAdaptRecursiveCTEWithMultipleCTEs(t *testing.T) {
	stmt, err := ParseString(`
WITH RECURSIVE
  base AS (SELECT 1 AS n),
  next AS (SELECT n + 1 FROM base WHERE n < 10)
SELECT n FROM next`)
	require.NoError(t, err)
	require.True(t, stmt.With.Recursive)
	require.Len(t, stmt.With.CTEs, 2)
	require.Equal(t, "base", stmt.With.CTEs[0].Name)
	require.Equal(t, "next", stmt.With.CTEs[1].Name)
}

func TestAdaptCTEMaterializedHint(t *testing.T) {
	stmt, err := ParseString("WITH cte AS MATERIALIZED (SELECT 1) SELECT * FROM cte")
	require.NoError(t, err)
	require.NotNil(t, stmt.With.CTEs[0].Materialized)
	require.True(t, *stmt.With.CTEs[0].Materialized)

	stmt, err = ParseString("WITH cte AS NOT MATERIALIZED (SELECT 1) SELECT * FROM cte")
	require.NoError(t, err)
	require.False(t, *stmt.With.CTEs[0].Materialized)

	stmt, err = ParseString("WITH cte AS (SELECT 1) SELECT * FROM cte")
	require.NoError(t, err)
	require.Nil(t, stmt.With.CTEs[0].Materialized)
}

func TestAdaptUnion(t *testing.T) {
	stmt, err := ParseString("SELECT a FROM t1 UNION SELECT a FROM t2")
	require.NoError(t, err)
	require.Equal(t, SetOpUnion, stmt.SetOp)
	require.False(t, stmt.SetAll)
	require.NotNil(t, stmt.SetLeft)
	require.NotNil(t, stmt.SetRight)
	require.Equal(t, SetOpNone, stmt.SetLeft.SetOp)
}

func TestAdaptUnionAllByName(t *testing.T) {
	stmt, err := ParseString("SELECT a FROM t1 UNION ALL BY NAME SELECT a FROM t2")
	require.NoError(t, err)
	require.True(t, stmt.SetAll)
	require.True(t, stmt.SetByName)
}

func TestAdaptExcept(t *testing.T) {
	stmt, err := ParseString("SELECT a FROM t1 EXCEPT SELECT a FROM t2")
	require.NoError(t, err)
	require.Equal(t, SetOpExcept, stmt.SetOp)
}

func TestAdaptIntersectBindsTighterThanUnion(t *testing.T) {
	// a INTERSECT b UNION c must parse as (a INTERSECT b) UNION c
	stmt, err := ParseString("SELECT a FROM t1 INTERSECT SELECT a FROM t2 UNION SELECT a FROM t3")
	require.NoError(t, err)
	require.Equal(t, SetOpUnion, stmt.SetOp)
	require.Equal(t, SetOpIntersect, stmt.SetLeft.SetOp)
}

func TestAdaptParenthesizedSetOpOperand(t *testing.T) {
	stmt, err := ParseString("(SELECT a FROM t1 ORDER BY a LIMIT 5) UNION (SELECT a FROM t2)")
	require.NoError(t, err)
	require.Equal(t, SetOpUnion, stmt.SetOp)
	require.NotNil(t, stmt.SetLeft.OrderBy)
	require.NotNil(t, stmt.SetLeft.Limit)
}

func TestAdaptValues(t *testing.T) {
	stmt, err := ParseString("VALUES (1, 'a'), (2, 'b')")
	require.NoError(t, err)
	require.Len(t, stmt.Values, 2)
	require.Len(t, stmt.Values[0], 2)
	require.Equal(t, "1", stmt.Values[0][0].(*Literal).Text)
}

func TestAdaptCTEWithSubquery(t *testing.T) {
	// exercises adaptExpression -> adaptSelectStatementInternal -> WithClause
	// nested inside a scalar subquery, not just at the statement top level
	stmt, err := ParseString("SELECT (WITH cte AS (SELECT 1 AS n) SELECT n FROM cte) AS x")
	require.NoError(t, err)
	sub := stmt.Columns[0].Expr.(*SubqueryExpr)
	require.NotNil(t, sub.Select.With)
}

// TestAdaptParenSelectKeepsInnerModifiers checks that a parenthesized
// statement's own WITH/ORDER BY/LIMIT survive when the outer statement adds
// clauses of its own — merging those onto the inner statement would
// silently change which rows the query returns.
func TestAdaptParenSelectKeepsInnerModifiers(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{
			"outer limit over inner limit",
			"(SELECT a FROM t LIMIT 10) LIMIT 100",
			"SELECT * FROM (SELECT a FROM t LIMIT 10) AS paren_select LIMIT 100",
		},
		{
			"outer order by applies after inner limit",
			"(SELECT a FROM t ORDER BY a LIMIT 5) ORDER BY b",
			"SELECT * FROM (SELECT a FROM t ORDER BY a LIMIT 5) AS paren_select ORDER BY b",
		},
		{
			"outer with over inner with",
			"WITH x AS (SELECT 1) (WITH y AS (SELECT 2) SELECT * FROM y)",
			"WITH x AS (SELECT 1) SELECT * FROM (WITH y AS (SELECT 2) SELECT * FROM y) AS paren_select",
		},
		{
			"inner with kept when there is no outer with",
			"(WITH y AS (SELECT 2) SELECT * FROM y)",
			"WITH y AS (SELECT 2) SELECT * FROM y",
		},
		{
			"non-conflicting outer limit merges",
			"(SELECT a FROM t ORDER BY a) LIMIT 5",
			"SELECT a FROM t ORDER BY a LIMIT 5",
		},
		{
			"non-conflicting outer with merges",
			"WITH x AS (SELECT 1) (SELECT * FROM x LIMIT 3)",
			"WITH x AS (SELECT 1) SELECT * FROM x LIMIT 3",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stmt, err := ParseString(c.sql)
			require.NoError(t, err)
			got := String(stmt)
			require.Equal(t, c.want, got)

			stmt2, err := ParseString(got)
			require.NoError(t, err)
			require.Equal(t, got, String(stmt2))
		})
	}
}
