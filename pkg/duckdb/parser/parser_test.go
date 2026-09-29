// pkg/duckdb/parser/parser_test.go
package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	files, err := filepath.Glob("testdata/queries/*.sql")
	require.NoError(t, err)
	require.NotEmpty(t, files, "no fixtures discovered; check the glob")

	g := goldie.New(t,
		goldie.WithFixtureDir("testdata/queries/parser"),
		goldie.WithNameSuffix(".sql.out"),
		goldie.WithDiffEngine(goldie.ColoredDiff),
	)

	for _, f := range files {
		f := f
		name := strings.TrimSuffix(filepath.Base(f), ".sql")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(f)
			require.NoError(t, err)

			var out strings.Builder
			for _, stmt := range strings.Split(string(src), ";\n\n") {
				stmt = strings.TrimSpace(stmt)
				if stmt == "" {
					continue
				}
				out.WriteString("-- " + stmt + "\n")
				parsed, err := ParseString(stmt)
				if err != nil {
					out.WriteString("ERROR: " + err.Error() + "\n\n")
					continue
				}
				out.WriteString(Dump(parsed))
				out.WriteString("\n")
			}
			g.Assert(t, name, []byte(out.String()))
		})
	}
}

func TestParseStringRecoversPanicAsError(t *testing.T) {
	// DESCRIBE as a standalone statement stays out of scope permanently
	// (see Task 11's plan entry) — the adapter panics internally;
	// ParseString must turn that into a plain error, never let it cross
	// out. (This example has been swapped three times now, from GROUP BY
	// to JOIN to WITH to PIVOT, as each gained real support in a later
	// task — this one shouldn't need to move again.)
	_, err := ParseString("DESCRIBE t")
	require.Error(t, err)
}

func TestParseStringPrefixesRecoveredPanicsOnce(t *testing.T) {
	// Adapter panics already carry the package prefix; the recover in
	// ParseString must not add a second one.
	for _, sql := range []string{
		"DESCRIBE t",
		"SELECT * REPLACE (a + 1 AS a) FROM t",
		"SELECT * RENAME (a AS b) FROM t",
		"SELECT * FROM t TABLESAMPLE 10%",
	} {
		_, err := ParseString(sql)
		require.Error(t, err, sql)
		require.True(t, strings.HasPrefix(err.Error(), "duckdb/parser: "), err.Error())
		require.Equal(t, 1, strings.Count(err.Error(), "duckdb/parser:"), err.Error())
	}
}

func TestParseStringRejectsOversizedInput(t *testing.T) {
	// Padding stays within the limit and parses normally.
	atLimit := "SELECT 1" + strings.Repeat(" ", MaxSQLBytes-len("SELECT 1"))
	_, err := ParseString(atLimit)
	require.NoError(t, err)

	_, err = ParseString(atLimit + " ")
	var perr *ParseError
	require.ErrorAs(t, err, &perr)
	require.Contains(t, perr.Message, "exceeding the 65536-byte limit")
}

func TestParseStringRejectsDeepNesting(t *testing.T) {
	nestedParens := func(depth int) string {
		return "SELECT " + strings.Repeat("(", depth) + "1" + strings.Repeat(")", depth)
	}
	nestedSubqueries := func(depth int) string {
		return strings.Repeat("SELECT * FROM (", depth) + "SELECT 1" + strings.Repeat(") AS s", depth)
	}

	// Realistic nesting is unaffected.
	_, err := ParseString(nestedParens(100))
	require.NoError(t, err)
	_, err = ParseString(nestedSubqueries(100))
	require.NoError(t, err)

	// 20k levels fits in MaxSQLBytes but, unguarded, overflows the
	// goroutine stack — a fatal, unrecoverable crash rather than a panic.
	for _, sql := range []string{nestedParens(20000), nestedSubqueries(3000)} {
		require.Less(t, len(sql), MaxSQLBytes)
		_, err = ParseString(sql)
		var perr *ParseError
		require.ErrorAs(t, err, &perr)
		require.Equal(t, "query is nested too deeply", perr.Message)
	}

	// The pooled session that hit the limit is reusable afterwards.
	_, err = ParseString("SELECT a FROM t")
	require.NoError(t, err)
}

func TestParseStringRejectsMalformedSQL(t *testing.T) {
	_, err := ParseString("SELEC * FROM")
	require.Error(t, err)
}
