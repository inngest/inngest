// pkg/duckdb/parser/write_test.go
package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/parser/grammar"
	"github.com/stretchr/testify/require"
)

// TestWriteRoundTripsFixtures re-parses every valid statement in
// testdata/queries (the same fixtures TestParser dumps) through
// Write/String and checks the result is idempotent: printing it again
// produces byte-identical output. That's a weaker property than "prints
// back the original text" (Write normalizes spacing, keyword casing, and
// drops redundant parens — see Write's doc comment) but it's exactly the
// property that matters: the AST Write produces from its own output must
// mean the same thing Write already decided the first AST meant.
func TestWriteRoundTripsFixtures(t *testing.T) {
	files, err := filepath.Glob("testdata/queries/*.sql")
	require.NoError(t, err)
	require.NotEmpty(t, files, "no fixtures discovered; check the glob")

	for _, f := range files {
		if filepath.Base(f) == "malformed.sql" {
			continue
		}
		f := f
		name := strings.TrimSuffix(filepath.Base(f), ".sql")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(f)
			require.NoError(t, err)

			for _, stmt := range strings.Split(string(src), ";\n\n") {
				stmt = strings.TrimSpace(stmt)
				if stmt == "" {
					continue
				}
				t.Run(stmt, func(t *testing.T) {
					parsed, err := ParseString(stmt)
					require.NoError(t, err)

					out := String(parsed)
					reparsed, err := ParseString(out)
					require.NoError(t, err, "reparsing Write output %q", out)

					out2 := String(reparsed)
					require.Equal(t, out, out2, "Write is not idempotent")
				})
			}
		})
	}
}

// TestWritePrecedencePreservesGrouping checks the cases that would silently
// produce a different AST on reparse if Write got a precedence level or a
// parenthesization rule wrong — the failure mode a purely-idempotent check
// can't catch, since a wrong-but-*stable* paren choice would still be
// idempotent.
func TestWritePrecedencePreservesGrouping(t *testing.T) {
	cases := []struct {
		name, sql, want string
	}{
		{"no redundant parens", "1 + 2 * 3", "1 + 2 * 3"},
		{"explicit parens change grouping", "(1 + 2) * 3", "(1 + 2) * 3"},
		{"right-assoc minus needs parens", "1 - (2 - 3)", "1 - (2 - 3)"},
		{"left-assoc minus needs none", "(1 - 2) - 3", "1 - 2 - 3"},
		{"and binds tighter than or", "a OR b AND c", "a OR b AND c"},
		{"explicit parens over or/and", "(a OR b) AND c", "(a OR b) AND c"},
		{"not binds tighter than and", "NOT a AND b", "NOT a AND b"},
		{"double not", "NOT NOT a", "NOT NOT a"},
		{"cast binds a parenthesized sum", "(1 + 2)::BIGINT", "(1 + 2)::BIGINT"},
		{"exponent right of additive needs parens", "(a + b) ^ 2", "(a + b) ^ 2"},
		{"unary minus operand", "-(a + b)", "-(a + b)"},
		{"unary minus tight operand no parens", "-a + b", "-a + b"},
		{"not in comparison right operand keeps parens", "a = NOT b", "a = (NOT b)"},
		{"in list containment stays unparenthesized", "x IN list_col", "x IN list_col"},
		{"not in list containment", "x NOT IN [1, 2]", "x NOT IN [1, 2]"},
		{"in parenthesized single element stays a list", "x IN (list_col)", "x IN (list_col)"},
		{"comparison binds looser than json arrow", "'x' = data ->> 'name'", "'x' = data ->> 'name'"},
		{"comparison binds looser than concat", "name = 'a' || 'b'", "name = 'a' || 'b'"},
		{"explicit parens around comparison under concat", "(name = 'a') || 'b'", "(name = 'a') || 'b'"},
		{"not-equal is not factorial", "a!=b", "a != b"},
		{"comparison against unspaced negative", "a<=-1", "a <= -1"},
		{"like symbol binds looser than json arrow", "'x' ~~ data ->> 'name'", "'x' ~~ data ->> 'name'"},
		{"explicit parens around like symbol under json arrow", "('x' ~~ data) ->> 'name'", "('x' ~~ data) ->> 'name'"},
		{"regex match binds looser than concat", "a ~ 'b' || 'c'", "a ~ 'b' || 'c'"},
		{"not like symbol", "a NOT ~~* 'x'", "a NOT ~~* 'x'"},
		{"any subquery", "a = ANY (SELECT 1)", "a = ANY (SELECT 1)"},
		{"all list keeps its parens", "a >= ALL ([1, 2])", "a >= ALL ([1, 2])"},
		{"any over a column keeps its parens", "a <> ANY (xs)", "a <> ANY (xs)"},
		{"any operand binds looser than concat", "a || b = ANY (xs)", "a || b = ANY (xs)"},
		{"any under comparison", "(a = ANY (xs)) = TRUE", "a = ANY (xs) = TRUE"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := parseExprForTest(t, c.sql)
			got := String(e)
			require.Equal(t, c.want, got)

			// The printed form must reparse to something that prints the
			// same way again (idempotent) — guards against a paren choice
			// that "looks right" as text but reparses differently.
			e2 := parseExprForTest(t, got)
			require.Equal(t, got, String(e2))
		})
	}
}

func TestWriteSetOpScoping(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{
			// The right operand's parens in the source are redundant (a
			// plain simple select needs no scoping) — Write drops them,
			// same as it drops any other redundant grouping parens.
			"limit scopes to one operand, needs parens",
			"(SELECT a FROM t1 ORDER BY a LIMIT 5) UNION (SELECT a FROM t2)",
			"(SELECT a FROM t1 ORDER BY a LIMIT 5) UNION SELECT a FROM t2",
		},
		{
			"intersect binds tighter than union",
			"SELECT a FROM t1 UNION SELECT a FROM t2 INTERSECT SELECT a FROM t3",
			"SELECT a FROM t1 UNION SELECT a FROM t2 INTERSECT SELECT a FROM t3",
		},
		{
			"union parenthesized under intersect needs parens back",
			"(SELECT a FROM t1 UNION SELECT a FROM t2) INTERSECT SELECT a FROM t3",
			"(SELECT a FROM t1 UNION SELECT a FROM t2) INTERSECT SELECT a FROM t3",
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

// errWriter always fails, to confirm Write surfaces and short-circuits on
// the first write error rather than panicking or ignoring it.
type errWriter struct{}

var errBoom = errors.New("boom")

func (errWriter) Write(p []byte) (int, error) { return 0, errBoom }

func TestWriteReturnsWriterError(t *testing.T) {
	stmt, err := ParseString("SELECT a FROM t")
	require.NoError(t, err)
	err = Write(errWriter{}, stmt)
	require.ErrorIs(t, err, errBoom)
}

// An empty operator would render its operands adjacent (`f  (x)`), which
// DuckDB reparses as something else entirely, so Write refuses it.
func TestWriteRejectsEmptyOperator(t *testing.T) {
	for _, n := range []Node{
		&BinaryExpr{Left: &Ident{Parts: []string{"f"}}, Right: &Ident{Parts: []string{"x"}}},
		&UnaryExpr{X: &Ident{Parts: []string{"x"}}},
	} {
		require.EqualError(t, Write(&strings.Builder{}, n), "duckdb/parser: Write: empty operator")
	}
}

func TestWriteQuotesKeywordIdentifiers(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"reserved column", `SELECT "group" FROM t`, `SELECT "group" FROM t`},
		{
			"reserved aliases and table",
			`SELECT a AS "select" FROM "table" AS "order"`,
			`SELECT a AS "select" FROM "table" AS "order"`,
		},
		{
			"reserved dotted parts",
			`SELECT "select"."from", t."group".x FROM t GROUP BY "order"`,
			`SELECT "select"."from", t."group".x FROM t GROUP BY "order"`,
		},
		{"reserved cte name", `WITH "select" AS (SELECT 1) SELECT * FROM "select"`, `WITH "select" AS (SELECT 1) SELECT * FROM "select"`},
		{"embedded quote still escaped", `SELECT "a""b" FROM t`, `SELECT "a""b" FROM t`},
		{"unreserved keywords stay bare", `SELECT a AS year, name, value FROM t`, `SELECT a AS year, name, value FROM t`},
		{"keyword-named functions stay bare", `SELECT left(s, 2), right(s, 1) FROM t`, `SELECT left(s, 2), right(s, 1) FROM t`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stmt, err := ParseString(c.sql)
			require.NoError(t, err)
			got := String(stmt)
			require.Equal(t, c.want, got)
			_, err = ParseString(got)
			require.NoError(t, err)
		})
	}
}

// TestWriteRoundTripsEveryKeywordAsIdentifier checks every vendored keyword,
// used as a quoted column, alias, table name and table alias, survives
// Write -> reparse with the same AST.
func TestWriteRoundTripsEveryKeywordAsIdentifier(t *testing.T) {
	_, kl, err := grammar.Load()
	require.NoError(t, err)
	for _, list := range [][]string{kl.Reserved, kl.Unreserved, kl.ColumnName, kl.FuncName, kl.TypeName} {
		for _, kw := range list {
			kw := strings.ToLower(kw)
			src := fmt.Sprintf(`SELECT "%[1]s", x."%[1]s" AS "%[1]s" FROM "%[1]s" AS x, y AS "%[1]s"`, kw)
			stmt, err := ParseString(src)
			require.NoError(t, err, src)
			out := String(stmt)
			reparsed, err := ParseString(out)
			require.NoError(t, err, "keyword %q: reparsing %q", kw, out)
			require.Equal(t, Dump(stmt), Dump(reparsed), "keyword %q: %q", kw, out)
		}
	}
}

func TestStringNeverPanicsOnCoreNodeKinds(t *testing.T) {
	stmt, err := ParseString("SELECT a, b FROM t WHERE a > 1 ORDER BY a LIMIT 1")
	require.NoError(t, err)
	require.NotEmpty(t, String(stmt))
	require.NotEmpty(t, String(stmt.From))
	require.NotEmpty(t, String(stmt.Where))
	require.NotEmpty(t, String(stmt.OrderBy))
	require.NotEmpty(t, String(stmt.Limit))
	require.NotEmpty(t, String(stmt.Columns[0]))
}

// requireWriteRoundTrip checks sql (a full statement) writes as want, and
// that want reparses to the same AST as sql — the property a paren or
// separator bug breaks, even when the printed text is stable.
func requireWriteRoundTrip(t *testing.T, sql, want string) {
	t.Helper()
	stmt, err := ParseString(sql)
	require.NoError(t, err)
	got := String(stmt)
	require.Equal(t, want, got)
	reparsed, err := ParseString(got)
	require.NoError(t, err, "reparsing Write output %q", got)
	require.Equal(t, Dump(stmt), Dump(reparsed), "Write output %q reparses differently", got)
}

func TestWriteSliceBounds(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"single index", "SELECT x[2]", "SELECT x[2]"},
		{"open-ended slice keeps its colon", "SELECT x[2:]", "SELECT x[2:]"},
		{"empty slice", "SELECT x[:]", "SELECT x[:]"},
		{"minus stop", "SELECT x[1:-]", "SELECT x[1:-]"},
		{"minus stop with step", "SELECT x[1:-:2]", "SELECT x[1:-:2]"},
		{"full slice", "SELECT x[1:2:3]", "SELECT x[1:2:3]"},
		{"step only", "SELECT x[::2]", "SELECT x[::2]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireWriteRoundTrip(t, c.sql, c.want)
		})
	}
}

// TestWriteBetweenInLikeOperandParens checks the left operand of
// BETWEEN/IN/LIKE is parenthesized when it is itself one of them: the
// grammar's BetweenInLikeExpression takes a single, non-repeating
// BetweenInLikeOp, so a flat `a LIKE 'x' IN (TRUE)` doesn't parse.
func TestWriteBetweenInLikeOperandParens(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"like under in", "SELECT (a LIKE 'x') IN (TRUE)", "SELECT (a LIKE 'x') IN (TRUE)"},
		{"between under between", "SELECT (a BETWEEN 1 AND 2) BETWEEN FALSE AND TRUE", "SELECT (a BETWEEN 1 AND 2) BETWEEN FALSE AND TRUE"},
		{"in under like", "SELECT (a IN (1, 2)) NOT LIKE 'x'", "SELECT (a IN (1, 2)) NOT LIKE 'x'"},
		{"in containment under in", "SELECT (a IN xs) IN (TRUE)", "SELECT (a IN xs) IN (TRUE)"},
		{"other operator needs no parens", "SELECT a || b LIKE 'x'", "SELECT a || b LIKE 'x'"},
		{"escape binds a comparison", "SELECT a LIKE 'x' ESCAPE b = c", "SELECT a LIKE 'x' ESCAPE b = c"},
		{"escape keeps parens around or", "SELECT a LIKE 'x' ESCAPE (b OR c)", "SELECT a LIKE 'x' ESCAPE (b OR c)"},
		{"escape like under comparison", "SELECT (a LIKE b ESCAPE c) = d", "SELECT (a LIKE b ESCAPE c) = d"},
		{"escape like ending a comparison under comparison", "SELECT (x = a LIKE b ESCAPE c) = d", "SELECT (x = a LIKE b ESCAPE c) = d"},
		{"escape like right of comparison stays flat", "SELECT d = (a LIKE b ESCAPE c)", "SELECT d = a LIKE b ESCAPE c"},
		{"escape like under and stays flat", "SELECT (a LIKE b ESCAPE c) AND d", "SELECT a LIKE b ESCAPE c AND d"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireWriteRoundTrip(t, c.sql, c.want)
		})
	}
}

// TestWriteIntervalValueParens checks a non-literal INTERVAL value keeps
// its parens: IntervalParameter only accepts a string, an unsigned number,
// or a ParensExpression.
func TestWriteIntervalValueParens(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"string", "SELECT INTERVAL '1' DAY", "SELECT INTERVAL '1' DAY"},
		{"number", "SELECT INTERVAL 1.5 DAY", "SELECT INTERVAL 1.5 DAY"},
		{"column", "SELECT INTERVAL (n) DAY", "SELECT INTERVAL (n) DAY"},
		{"sum", "SELECT INTERVAL (1 + 2) DAY", "SELECT INTERVAL (1 + 2) DAY"},
		{"negative number", "SELECT INTERVAL (-1) DAY", "SELECT INTERVAL (-1) DAY"},
		{"no unit", "SELECT INTERVAL (n)", "SELECT INTERVAL (n)"},
		{"redundant parens around a literal dropped", "SELECT INTERVAL (2) HOUR", "SELECT INTERVAL 2 HOUR"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireWriteRoundTrip(t, c.sql, c.want)
		})
	}
}

// TestWriteSeparatesAdjacentOperators checks Write never glues two
// operator tokens into a different one: "--" starts a line comment, "~~"
// is LIKE, and DuckDB rejects "!!".
func TestWriteSeparatesAdjacentOperators(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"double minus", "SELECT - -1", "SELECT - -1"},
		{"parenthesized double minus", "SELECT -(-a), b", "SELECT - -a, b"},
		{"double tilde", "SELECT ~ ~a", "SELECT ~ ~a"},
		{"minus tilde", "SELECT -~a", "SELECT - ~a"},
		{"plus minus", "SELECT +-a", "SELECT + -a"},
		{"binary minus then unary minus", "SELECT a - -1", "SELECT a - -1"},
		{"divide by unary minus", "SELECT a / -1", "SELECT a / -1"},
		{"single prefix stays tight", "SELECT -a", "SELECT -a"},
		{"double factorial", "SELECT (a!)!", "SELECT a! !"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireWriteRoundTrip(t, c.sql, c.want)
		})
	}
}

// TestWriteSetOpRightOperandParens checks a set operation's right operand
// keeps its parens when it is itself a set operation: set operations are
// left-associative, so only a tighter-binding INTERSECT under UNION/EXCEPT
// can be printed flat on the right.
func TestWriteSetOpRightOperandParens(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"except under except", "SELECT a FROM t1 EXCEPT (SELECT a FROM t2 EXCEPT SELECT a FROM t3)", "SELECT a FROM t1 EXCEPT (SELECT a FROM t2 EXCEPT SELECT a FROM t3)"},
		{"union under union all", "SELECT 1 UNION ALL (SELECT 1 UNION SELECT 1)", "SELECT 1 UNION ALL (SELECT 1 UNION SELECT 1)"},
		{"union under except", "SELECT 1 EXCEPT (SELECT 1 UNION SELECT 2)", "SELECT 1 EXCEPT (SELECT 1 UNION SELECT 2)"},
		{"intersect under intersect", "SELECT 1 INTERSECT (SELECT 1 INTERSECT SELECT 2)", "SELECT 1 INTERSECT (SELECT 1 INTERSECT SELECT 2)"},
		{"intersect under union stays flat", "SELECT 1 UNION (SELECT 1 INTERSECT SELECT 2)", "SELECT 1 UNION SELECT 1 INTERSECT SELECT 2"},
		{"intersect under except stays flat", "SELECT 1 EXCEPT (SELECT 1 INTERSECT SELECT 2)", "SELECT 1 EXCEPT SELECT 1 INTERSECT SELECT 2"},
		{"left-nested stays flat", "(SELECT 1 EXCEPT SELECT 2) EXCEPT SELECT 3", "SELECT 1 EXCEPT SELECT 2 EXCEPT SELECT 3"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireWriteRoundTrip(t, c.sql, c.want)
		})
	}
}

// TestWriteQuotesBoundNames checks names that bind rather than reference
// (named-argument names, lambda params, list-comprehension loop vars, a
// PIVOT enum target) are quoted like any other identifier. Printed bare,
// a quoted name's contents became live SQL.
func TestWriteQuotesBoundNames(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"named arg", `SELECT struct_pack("a b" := 1)`, `SELECT struct_pack("a b" := 1)`},
		{"named arg keyword", `SELECT struct_pack("select" := 1)`, `SELECT struct_pack("select" := 1)`},
		{"named arg simple", `SELECT struct_pack(a := 1)`, `SELECT struct_pack(a := 1)`},
		{"lambda param", `SELECT list_transform(l, "a b" -> 1) FROM t`, `SELECT list_transform(l, "a b" -> 1) FROM t`},
		{"lambda params", `SELECT list_reduce(l, ("a b", c) -> 1) FROM t`, `SELECT list_reduce(l, ("a b", c) -> 1) FROM t`},
		{"list comprehension var", `SELECT [1 FOR "x IN [1]] a, (SELECT 1 FROM u) n, [1 FOR y" IN [1]] c FROM t`, `SELECT [1 FOR "x IN [1]] a, (SELECT 1 FROM u) n, [1 FOR y" IN [1]] AS c FROM t`},
		{"list comprehension vars", `SELECT [x FOR x, "select" IN l] FROM t`, `SELECT [x FOR x, "select" IN l] FROM t`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireWriteRoundTrip(t, c.sql, c.want)
		})
	}
}

func TestWritePivotEnumTargetKeepsQuoting(t *testing.T) {
	stmt, err := ParseString(`SELECT * FROM t PIVOT (sum(x) FOR y IN "a b")`)
	require.NoError(t, err)
	require.Contains(t, String(stmt), `IN ("a b")`)
}

// TestWriteOffsetOnly checks an OFFSET with no LIMIT (a LimitClause with a
// nil Limit) prints as a bare OFFSET wherever it appears, not just at the
// root, where it used to be the only place it didn't panic.
func TestWriteOffsetOnly(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"root", "SELECT a FROM t OFFSET 5", "SELECT a FROM t OFFSET 5"},
		{"from subquery", "SELECT * FROM (SELECT a FROM t OFFSET 5) s", "SELECT * FROM (SELECT a FROM t OFFSET 5) AS s"},
		{"in subquery", "SELECT a FROM t WHERE a IN (SELECT b FROM u OFFSET 1)", "SELECT a FROM t WHERE a IN (SELECT b FROM u OFFSET 1)"},
		{"cte body", "WITH c AS (SELECT a FROM t OFFSET 2) SELECT a FROM c", "WITH c AS (SELECT a FROM t OFFSET 2) SELECT a FROM c"},
		{"set-op operand", "(SELECT a FROM t OFFSET 1) UNION ALL SELECT a FROM u", "(SELECT a FROM t OFFSET 1) UNION ALL SELECT a FROM u"},
		{"order by then offset", "SELECT a FROM t ORDER BY a OFFSET 1", "SELECT a FROM t ORDER BY a OFFSET 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireWriteRoundTrip(t, c.sql, c.want)
		})
	}
}

// TestWriteReturnsErrorForUnwritableAST checks Write reports an AST it
// can't render as an error rather than panicking out to its caller.
func TestWriteReturnsErrorForUnwritableAST(t *testing.T) {
	stmt := &SelectStatement{Columns: []*SelectItem{{Expr: nil}}}
	var sb strings.Builder
	err := Write(&sb, stmt)
	require.ErrorContains(t, err, "duckdb/parser: Write: unhandled Expr <nil>")
}

// TestWriteParenthesizesIdentUnderFieldAccess checks a parenthesized Ident
// under a field access or method call keeps its parens: printed flat, it
// merges into the dotted name (`(t).x`, a struct field, becomes column x
// of table t, and `(x).f(1)` a schema-qualified function).
func TestWriteParenthesizesIdentUnderFieldAccess(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"field of ident", "SELECT (t).x FROM t", "SELECT (t).x FROM t"},
		{"field of qualified ident", "SELECT (t.a).x FROM t", "SELECT (t.a).x FROM t"},
		{"nested field", "SELECT ((t).a).x FROM t", "SELECT (t).a.x FROM t"},
		{"method on ident", "SELECT (x).f(1) FROM t", "SELECT (x).f(1) FROM t"},
		{"method on qualified ident", "SELECT (t.a).f(1) FROM t", "SELECT (t.a).f(1) FROM t"},
		{"plain dotted name stays flat", "SELECT t.a.b FROM t", "SELECT t.a.b FROM t"},
		{"field of call stays flat", "SELECT f(x).a FROM t", "SELECT f(x).a FROM t"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireWriteRoundTrip(t, c.sql, c.want)
		})
	}
}

// TestWriteKeepsSubqueryNot checks a SubqueryExpr's own NOT prefix is
// printed when it isn't also EXISTS.
func TestWriteKeepsSubqueryNot(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"not subquery", "SELECT TRUE IS NOT DISTINCT FROM NOT (SELECT TRUE)", "SELECT TRUE IS NOT DISTINCT FROM NOT (SELECT TRUE)"},
		{"not exists", "SELECT TRUE IS DISTINCT FROM NOT EXISTS (SELECT TRUE)", "SELECT TRUE IS DISTINCT FROM NOT EXISTS (SELECT TRUE)"},
		{"under prefix", "SELECT -NOT (SELECT 1)", "SELECT -NOT (SELECT 1)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireWriteRoundTrip(t, c.sql, c.want)
		})
	}
}

// TestWriteParenthesizesCastUnderIndirection checks a `::` cast keeps its
// parens under a subscript or field access, since its Type would otherwise
// take the indirection as part of the type (`a::INT[1]` is a cast to an
// array type). A number under a field access needs them too, or its dot
// lexes as a decimal point.
func TestWriteParenthesizesCastUnderIndirection(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"subscript of cast", "SELECT (a::INT)[1] FROM t", "SELECT (a::INT)[1] FROM t"},
		{"subscript of CAST call", "SELECT CAST(a AS INT)[1] FROM t", "SELECT (a::INT)[1] FROM t"},
		{"subscript of array cast", "SELECT (a::INT[])[1] FROM t", "SELECT (a::INT[])[1] FROM t"},
		{"field of cast", "SELECT (a::t).k FROM t", "SELECT (a::t).k FROM t"},
		{"method on cast", "SELECT (a::t).f(1) FROM t", "SELECT (a::t).f(1) FROM t"},
		{"subscript of try_cast stays flat", "SELECT TRY_CAST(a AS INT)[1] FROM t", "SELECT TRY_CAST(a AS INT)[1] FROM t"},
		{"cast of cast stays flat", "SELECT (a::INT)::TEXT FROM t", "SELECT a::INT::TEXT FROM t"},
		{"array type with a bound", "SELECT a::INT[][1] FROM t", "SELECT a::INT[][1] FROM t"},
		{"field of number", "SELECT (1).x", "SELECT (1).x"},
		{"subscript of number stays flat", "SELECT (1)[1]", "SELECT 1[1]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireWriteRoundTrip(t, c.sql, c.want)
		})
	}

	// The whole `INT[][1]` is the cast's type, matching DuckDB, not a
	// subscript of a cast to `INT[]`.
	stmt, err := ParseString("SELECT a::INT[][1] FROM t")
	require.NoError(t, err)
	cast, ok := stmt.Columns[0].Expr.(*CastExpr)
	require.True(t, ok, "got %T", stmt.Columns[0].Expr)
	require.Equal(t, "INT[][1]", cast.Type)
}

// TestWriteQuotesKeywordFunctionNames checks a function whose name is a
// keyword FunctionName rejects, or that starts some other syntax, is
// quoted, so it reparses as a call to that same function.
func TestWriteQuotesKeywordFunctionNames(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"reserved keyword", `SELECT "not"(x) FROM t`, `SELECT "not"(x) FROM t`},
		{"interval", `SELECT "interval"(x) FROM t`, `SELECT "interval"(x) FROM t`},
		{"distinct", `SELECT "distinct"(x) FROM t`, `SELECT "distinct"(x) FROM t`},
		{"grouping_id", `SELECT "grouping_id"(x) FROM t`, `SELECT "grouping_id"(x) FROM t`},
		{"special function syntax", `SELECT "position"(a, b) FROM t`, `SELECT "position"(a, b) FROM t`},
		{"func-name keyword stays bare", "SELECT left(a, 1), right(a, 1) FROM t", "SELECT left(a, 1), right(a, 1) FROM t"},
		{"keyword schema", `SELECT "select".f(x) FROM t`, `SELECT "select".f(x) FROM t`},
		{"keyword after schema stays bare", `SELECT s."not"(x) FROM t`, `SELECT s.not(x) FROM t`},
		{"keyword method stays bare", "SELECT (x).not(1) FROM t", "SELECT (x).not(1) FROM t"},
		{"case-sensitive name", `SELECT "Foo"(x) FROM t`, `SELECT Foo(x) FROM t`},
		{"non-identifier name", `SELECT "a b"(x) FROM t`, `SELECT "a b"(x) FROM t`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireWriteRoundTrip(t, c.sql, c.want)
		})
	}
}

// TestWriteSpecialFunctions checks the keyword-argument call forms adapt
// to the plain function call DuckDB itself desugars them to, and print as
// that call.
func TestWriteSpecialFunctions(t *testing.T) {
	cases := []struct{ name, sql, want string }{
		{"coalesce", "SELECT coalesce(a, b) FROM t", "SELECT COALESCE(a, b) FROM t"},
		{"quoted coalesce", `SELECT "coalesce"(a) FROM t`, "SELECT COALESCE(a) FROM t"},
		{"nullif", "SELECT NULLIF(a, b) FROM t", `SELECT "nullif"(a, b) FROM t`},
		{"extract keyword", "SELECT EXTRACT(year FROM a) FROM t", "SELECT date_part('YEAR', a) FROM t"},
		{"extract plural keyword", "SELECT EXTRACT(years FROM a) FROM t", "SELECT date_part('YEAR', a) FROM t"},
		{"extract identifier", "SELECT EXTRACT(epoch FROM a) FROM t", "SELECT date_part('epoch', a) FROM t"},
		{"extract quoted identifier", `SELECT EXTRACT("Foo" FROM a) FROM t`, "SELECT date_part('Foo', a) FROM t"},
		{"extract string", "SELECT EXTRACT('Epoch' FROM a) FROM t", "SELECT date_part('Epoch', a) FROM t"},
		{"trim", "SELECT TRIM(a) FROM t", `SELECT "trim"(a) FROM t`},
		{"trim list", "SELECT TRIM(a, 'x') FROM t", `SELECT "trim"(a, 'x') FROM t`},
		{"trim both from", "SELECT TRIM(BOTH 'x' FROM a) FROM t", `SELECT "trim"(a, 'x') FROM t`},
		{"trim from", "SELECT TRIM(FROM a) FROM t", `SELECT "trim"(a) FROM t`},
		{"trim leading", "SELECT TRIM(LEADING 'x' FROM a) FROM t", "SELECT ltrim(a, 'x') FROM t"},
		{"trim trailing", "SELECT TRIM(TRAILING FROM a) FROM t", "SELECT rtrim(a) FROM t"},
		{"trim source then list", "SELECT TRIM('x' FROM a, b) FROM t", `SELECT "trim"(a, b, 'x') FROM t`},
		{"substring from for", "SELECT SUBSTRING(a FROM 1 FOR 2) FROM t", `SELECT "substring"(a, 1, 2) FROM t`},
		{"substring from", "SELECT SUBSTRING(a FROM 2) FROM t", `SELECT "substring"(a, 2) FROM t`},
		{"substring for", "SELECT SUBSTRING(a FOR 2) FROM t", `SELECT "substring"(a, 1, 2) FROM t`},
		{"substring list", "SELECT SUBSTRING(a, 1, 2) FROM t", `SELECT "substring"(a, 1, 2) FROM t`},
		{"position", "SELECT POSITION('a' IN b) FROM t", `SELECT "position"(b, 'a') FROM t`},
		{"overlay", "SELECT OVERLAY(a PLACING b FROM 1 FOR 2) FROM t", `SELECT "overlay"(a, b, 1, 2) FROM t`},
		{"overlay list", "SELECT OVERLAY(a, b, 1) FROM t", `SELECT "overlay"(a, b, 1) FROM t`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			requireWriteRoundTrip(t, c.sql, c.want)
		})
	}
}
