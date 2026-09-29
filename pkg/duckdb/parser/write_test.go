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
