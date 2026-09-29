package insights

import (
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/parser"
	"github.com/stretchr/testify/require"
)

func TestTranspileHappyPath(t *testing.T) {
	tr, err := Transpile("SELECT run_id, app_id FROM runs WHERE status = 'Completed'", testAccountID, testEnvID)
	require.NoError(t, err)

	require.Equal(t, []any{testAccountID.String(), testEnvID.String()}, tr.Args)
	require.Equal(t, "runs", tr.PrimaryTable)
	require.Equal(t, []string{"runs"}, tr.Tables)
	require.True(t, tr.Limited)
	require.Equal(t, [][]PathHint{{{Hint: HintRunID}}, {{Hint: HintAppID}}}, tr.ColumnPathHints)
	require.Contains(t, tr.SQL, "inngest.insights_runs")
	require.Contains(t, tr.SQL, "LIMIT 1000")

	require.Len(t, tr.Diagnostics, 1)
	require.Equal(t, "default-limit-applied", tr.Diagnostics[0].Code)
	require.Equal(t, DiagnosticInfo, tr.Diagnostics[0].Severity)
}

func TestTranspileNoLimitDiagnosticWhenLimitSpecified(t *testing.T) {
	tr, err := Transpile("SELECT run_id FROM runs LIMIT 5", testAccountID, testEnvID)
	require.NoError(t, err)
	require.Empty(t, tr.Diagnostics)
}

func TestTranspileArrayOfStructDiagnostic(t *testing.T) {
	tr, err := Transpile("SELECT sessions.key FROM runs LIMIT 5", testAccountID, testEnvID)
	require.NoError(t, err)
	require.Contains(t, tr.SQL, "sessions ->> '$[*].key'")
	require.Len(t, tr.Diagnostics, 1)
	require.Equal(t, "array-of-struct-access-rewritten", tr.Diagnostics[0].Code)
}

func TestTranspileRejectsInvalidQuery(t *testing.T) {
	_, err := Transpile("SELECT * FROM nonexistent_table", testAccountID, testEnvID)
	require.Error(t, err)
}

func TestTranspileRejectsMalformedSQL(t *testing.T) {
	_, err := Transpile("SELEKT * FROM runs", testAccountID, testEnvID)
	require.Error(t, err)
}

func TestTranspileRejectsMultipleStatements(t *testing.T) {
	_, err := Transpile("SELECT run_id FROM runs; SELECT run_id FROM runs;", testAccountID, testEnvID)
	require.Error(t, err)
}

func TestTranspileRespectsExistingLimit(t *testing.T) {
	tr, err := Transpile("SELECT run_id FROM runs LIMIT 5", testAccountID, testEnvID)
	require.NoError(t, err)
	require.False(t, tr.Limited)
}

// A quoted list-comprehension loop variable used to be written back
// unquoted, so arbitrary text inside it -- here a bare inngest.runs that
// skips the tenant-scoping macro -- reached DuckDB as live SQL.
func TestTranspileQuotesListComprehensionVars(t *testing.T) {
	tr, err := Transpile(`SELECT [1 FOR "x IN [1]] a, (SELECT count(*) FROM inngest.runs) n, [1 FOR y" IN [1]] c FROM runs`, testAccountID, testEnvID)
	require.NoError(t, err)
	require.Contains(t, tr.SQL, `[1 FOR "x IN [1]] a, (SELECT count(*) FROM inngest.runs) n, [1 FOR y" IN [1]]`)
	requireSingleMacroScopedTable(t, tr.SQL)
}

// Same bypass through a named argument's name (struct_pack is allowlisted).
func TestTranspileQuotesNamedArgNames(t *testing.T) {
	tr, err := Transpile(`SELECT struct_pack("a := 1, b := (SELECT count(*) FROM inngest.runs), c" := 1) AS s FROM runs`, testAccountID, testEnvID)
	require.NoError(t, err)
	require.Contains(t, tr.SQL, `struct_pack("a := 1, b := (SELECT count(*) FROM inngest.runs), c" := 1)`)
	requireSingleMacroScopedTable(t, tr.SQL)
}

// requireSingleMacroScopedTable reparses sql and checks its only table
// reference is a tenant-scoped insights macro call.
func requireSingleMacroScopedTable(t *testing.T, sql string) {
	t.Helper()
	stmt, err := parser.ParseString(sql)
	require.NoError(t, err, "reparsing %q", sql)
	v := &bareTableCollector{}
	parser.Walk(v, stmt)
	require.Empty(t, v.names, "bare table references in %q", sql)
}

type bareTableCollector struct{ names []string }

func (c *bareTableCollector) Visit(n parser.Node) parser.Visitor {
	if r, ok := n.(*parser.BaseTableRef); ok {
		c.names = append(c.names, strings.Join(r.Name, "."))
	}
	return c
}

// OPERATOR(op) used to adapt to an empty-Op BinaryExpr, which Write
// rendered as `getenv  (...)` -- DuckDB reparses that as a call to a
// function the allowlist never saw.
func TestTranspileRejectsQualifiedOperator(t *testing.T) {
	for _, sql := range []string{
		`WITH c AS (SELECT run_id AS getenv FROM runs) SELECT getenv OPERATOR(+) ('HOME' || '') AS leaked FROM c`,
		`SELECT OPERATOR(-) 1 AS x FROM runs`,
	} {
		t.Run(sql, func(t *testing.T) {
			_, err := Transpile(sql, testAccountID, testEnvID)
			require.ErrorContains(t, err, "OPERATOR(...) is not supported")
		})
	}
}
