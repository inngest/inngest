// pkg/duckdb/parser/adapter_expr_test.go
package parser

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/inngest/inngest/pkg/duckdb/parser/grammar"
	"github.com/inngest/inngest/pkg/duckdb/parser/peg"
)

func parseExprForTest(t *testing.T, sql string) Expr {
	t.Helper()
	g, kl, err := grammar.Load()
	require.NoError(t, err)
	ks := NewKeywordSets(kl.Reserved, kl.Unreserved, kl.ColumnName, kl.FuncName, kl.TypeName)
	p := &peg.Parser{Grammar: g, Primitives: Primitives(ks), SkipTrivia: SkipSQLTrivia}
	n, err := p.Parse(sql, "Expression")
	require.NoErrorf(t, err, "parsing %q", sql)
	a := newAdapter(sql)
	return a.adaptExpression(n)
}

func TestAdaptArithmeticPrecedence(t *testing.T) {
	// 1 + 2 * 3 must bind as 1 + (2 * 3), not (1 + 2) * 3
	e := parseExprForTest(t, "1 + 2 * 3")
	add, ok := e.(*BinaryExpr)
	require.True(t, ok)
	require.Equal(t, "+", add.Op)
	require.Equal(t, "1", add.Left.(*Literal).Text)
	mul, ok := add.Right.(*BinaryExpr)
	require.True(t, ok)
	require.Equal(t, "*", mul.Op)
}

func TestAdaptLeftAssociativity(t *testing.T) {
	// 1 - 2 - 3 must bind as (1 - 2) - 3
	e := parseExprForTest(t, "1 - 2 - 3")
	outer, ok := e.(*BinaryExpr)
	require.True(t, ok)
	require.Equal(t, "-", outer.Op)
	require.Equal(t, "3", outer.Right.(*Literal).Text)
	inner, ok := outer.Left.(*BinaryExpr)
	require.True(t, ok)
	require.Equal(t, "1", inner.Left.(*Literal).Text)
	require.Equal(t, "2", inner.Right.(*Literal).Text)
}

func TestAdaptLogicalPrecedence(t *testing.T) {
	// a AND b OR c must bind as (a AND b) OR c
	e := parseExprForTest(t, "a AND b OR c")
	or, ok := e.(*BinaryExpr)
	require.True(t, ok)
	require.Equal(t, "OR", or.Op)
	and, ok := or.Left.(*BinaryExpr)
	require.True(t, ok)
	require.Equal(t, "AND", and.Op)
}

func TestAdaptDoubleNegation(t *testing.T) {
	e := parseExprForTest(t, "NOT NOT a")
	outer, ok := e.(*UnaryExpr)
	require.True(t, ok)
	require.Equal(t, "NOT", outer.Op)
	inner, ok := outer.X.(*UnaryExpr)
	require.True(t, ok)
	require.Equal(t, "NOT", inner.Op)
}

func TestAdaptIsNullFamily(t *testing.T) {
	// "IS [NOT] NULL" matches IsExpr, not NullTest: IsTest's ordered choice
	// tries IsLiteral (which itself accepts 'IS' 'NOT'? NULL/TRUE/FALSE/
	// UNKNOWN) before NotNull/IsNull, and IsLiteral matches first here.
	// NullTest only covers the single-token/no-IS spellings below.
	e := parseExprForTest(t, "a IS NOT NULL")
	is, ok := e.(*IsExpr)
	require.True(t, ok)
	require.True(t, is.Not)
	require.Equal(t, "NULL", is.Value)

	e = parseExprForTest(t, "a ISNULL")
	nt, ok := e.(*NullTest)
	require.True(t, ok)
	require.False(t, nt.Not)

	e = parseExprForTest(t, "a NOTNULL")
	nt, ok = e.(*NullTest)
	require.True(t, ok)
	require.True(t, nt.Not)

	e = parseExprForTest(t, "a NOT NULL")
	nt, ok = e.(*NullTest)
	require.True(t, ok)
	require.True(t, nt.Not)
}

func TestAdaptIsDistinctFrom(t *testing.T) {
	e := parseExprForTest(t, "a IS NOT DISTINCT FROM b")
	d, ok := e.(*DistinctFromExpr)
	require.True(t, ok)
	require.True(t, d.Not)
}

func TestAdaptBetween(t *testing.T) {
	e := parseExprForTest(t, "a NOT BETWEEN 1 AND 10")
	b, ok := e.(*BetweenExpr)
	require.True(t, ok)
	require.True(t, b.Not)
	require.Equal(t, "1", b.Low.(*Literal).Text)
	require.Equal(t, "10", b.High.(*Literal).Text)
}

func TestAdaptInList(t *testing.T) {
	e := parseExprForTest(t, "a IN (1, 2, 3)")
	in, ok := e.(*InExpr)
	require.True(t, ok)
	require.False(t, in.Not)
	require.Len(t, in.List, 3)
	require.Nil(t, in.Subquery)
}

func TestAdaptInSubquery(t *testing.T) {
	e := parseExprForTest(t, "a IN (SELECT id FROM t)")
	in, ok := e.(*InExpr)
	require.True(t, ok)
	require.NotNil(t, in.Subquery)
	require.Nil(t, in.List)
}

func TestAdaptLike(t *testing.T) {
	e := parseExprForTest(t, "a NOT LIKE '%x%' ESCAPE '\\'")
	l, ok := e.(*LikeExpr)
	require.True(t, ok)
	require.True(t, l.Not)
	require.Equal(t, "LIKE", l.Op)
	require.NotNil(t, l.Escape)
}

func TestAdaptCast(t *testing.T) {
	e := parseExprForTest(t, "CAST(a AS INTEGER)")
	c, ok := e.(*CastExpr)
	require.True(t, ok)
	require.False(t, c.TryCast)

	e = parseExprForTest(t, "a::VARCHAR")
	_, ok = e.(*CastExpr)
	require.True(t, ok)
}

func TestAdaptCase(t *testing.T) {
	e := parseExprForTest(t, "CASE a WHEN 1 THEN 'one' WHEN 2 THEN 'two' ELSE 'other' END")
	c, ok := e.(*CaseExpr)
	require.True(t, ok)
	require.NotNil(t, c.Operand)
	require.Len(t, c.Whens, 2)
	require.NotNil(t, c.Else)
}

func TestAdaptStarWithExclude(t *testing.T) {
	e := parseExprForTest(t, "t.* EXCLUDE (a, b)")
	s, ok := e.(*StarExpr)
	require.True(t, ok)
	require.Equal(t, []string{"t"}, s.Qualifier)
	require.Equal(t, []string{"a", "b"}, s.Exclude)
}

func TestAdaptFunctionCallShape(t *testing.T) {
	e := parseExprForTest(t, "count(DISTINCT a ORDER BY a) FILTER (WHERE a > 0)")
	f, ok := e.(*FunctionExpr)
	require.True(t, ok)
	require.Equal(t, []string{"count"}, f.Name)
	require.True(t, f.Distinct)
	require.NotNil(t, f.OrderBy)
	require.NotNil(t, f.Filter)
}

func TestAdaptWindowFunction(t *testing.T) {
	e := parseExprForTest(t, "row_number() OVER (PARTITION BY a ORDER BY b ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW)")
	f, ok := e.(*FunctionExpr)
	require.True(t, ok)
	require.NotNil(t, f.Over)
	require.Len(t, f.Over.PartitionBy, 1)
	require.NotNil(t, f.Over.OrderBy)
	require.NotNil(t, f.Over.Frame)
	require.Equal(t, "ROWS", f.Over.Frame.Unit)
	require.Equal(t, FrameUnboundedPreceding, f.Over.Frame.Start.Kind)
	require.Equal(t, FrameCurrentRow, f.Over.Frame.EndBound.Kind)
}

func TestAdaptNamedWindowReference(t *testing.T) {
	e := parseExprForTest(t, "row_number() OVER win")
	f, ok := e.(*FunctionExpr)
	require.True(t, ok)
	require.Equal(t, "win", f.Over.Name)
}

func TestAdaptSubqueryExists(t *testing.T) {
	// "NOT EXISTS (...)" — the outer boolean NOT (LogicalNotExpression)
	// greedily consumes the 'NOT' before SubqueryExpression's own optional
	// SubqueryNot ever sees it, so this is a UnaryExpr wrapping a
	// SubqueryExpr, not a SubqueryExpr with Not set directly.
	e := parseExprForTest(t, "NOT EXISTS (SELECT 1 FROM t)")
	u, ok := e.(*UnaryExpr)
	require.True(t, ok)
	require.Equal(t, "NOT", u.Op)
	s, ok := u.X.(*SubqueryExpr)
	require.True(t, ok)
	require.True(t, s.Exists)
	require.False(t, s.Not)
}

func TestAdaptListAndArray(t *testing.T) {
	e := parseExprForTest(t, "[1, 2, 3]")
	l, ok := e.(*ListExpr)
	require.True(t, ok)
	require.False(t, l.Paren)
	require.Len(t, l.Elems, 3)

	e = parseExprForTest(t, "(1, 2, 3)")
	l, ok = e.(*ListExpr)
	require.True(t, ok)
	require.True(t, l.Paren)
}

func TestAdaptGroupingParensIsTransparent(t *testing.T) {
	e := parseExprForTest(t, "(1 + 2) * 3")
	mul, ok := e.(*BinaryExpr)
	require.True(t, ok)
	require.Equal(t, "*", mul.Op)
	_, ok = mul.Left.(*BinaryExpr)
	require.True(t, ok, "grouping parens must not introduce a wrapper node")
}

func TestAdaptStruct(t *testing.T) {
	e := parseExprForTest(t, "{'a': 1, 'b': 2}")
	s, ok := e.(*StructExpr)
	require.True(t, ok)
	require.Len(t, s.Fields, 2)
	require.Equal(t, "a", s.Fields[0].Key)
}

func TestAdaptIndirectionChain(t *testing.T) {
	// ColumnReference itself greedily matches a dotted name like "a.b" as
	// a single qualified Ident, so a '.' only reaches BaseExpression's
	// IndirectionList as a DotExpr once something else (here, a slice)
	// has already ended the leading identifier chain.
	e := parseExprForTest(t, "a[1].b::INTEGER")
	cast, ok := e.(*CastExpr)
	require.True(t, ok)
	dot, ok := cast.X.(*DotExpr)
	require.True(t, ok)
	require.Equal(t, "b", dot.Field)
	sl, ok := dot.X.(*SliceExpr)
	require.True(t, ok)
	_, ok = sl.X.(*Ident)
	require.True(t, ok)
}

func TestAdaptLambdaArrow(t *testing.T) {
	e := parseExprForTest(t, "x -> x + 1")
	l, ok := e.(*LambdaExpr)
	require.True(t, ok)
	require.Equal(t, []string{"x"}, l.Params)
}

func TestAdaptQualifiedColumnReference(t *testing.T) {
	e := parseExprForTest(t, "s.t.c")
	id, ok := e.(*Ident)
	require.True(t, ok)
	require.Equal(t, []string{"s", "t", "c"}, id.Parts)
}

func TestAdaptQuotedIdentifierReservedWord(t *testing.T) {
	e := parseExprForTest(t, `"select"`)
	id, ok := e.(*Ident)
	require.True(t, ok)
	require.Equal(t, []string{"select"}, id.Parts)
}

func TestNodeSpanLocatesExactSourceSubstring(t *testing.T) {
	// Pos()/End() must bound exactly the substring that produced each node
	// — this is the whole point of carrying a span (see visitor.go's Node
	// doc comment): a caller locates/rewrites a symbol by slicing src.
	sql := "a.b + f(c, 1)"
	e := parseExprForTest(t, sql)
	add, ok := e.(*BinaryExpr)
	require.True(t, ok)

	left := add.Left.(*Ident)
	require.Equal(t, "a.b", sql[left.Pos().Offset:left.End().Offset])

	right := add.Right.(*FunctionExpr)
	require.Equal(t, "f(c, 1)", sql[right.Pos().Offset:right.End().Offset])

	arg := right.Args[0].(*Ident)
	require.Equal(t, "c", sql[arg.Pos().Offset:arg.End().Offset])
}
