package insights

import (
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/parser"
	"github.com/stretchr/testify/require"
)

func mustRewriteArrayOfStructAccess(t *testing.T, sql string) (*parser.SelectStatement, []Diagnostic) {
	t.Helper()
	stmt := mustParse(t, sql)
	_, ctes, _, err := validate(stmt)
	require.NoError(t, err)
	diags, err := rewriteArrayOfStructAccess(stmt, ctes, nil)
	require.NoError(t, err)
	return stmt, diags
}

func TestRewriteArrayOfStructAccessBareColumn(t *testing.T) {
	stmt, diags := mustRewriteArrayOfStructAccess(t, "SELECT sessions.key FROM runs")
	require.Equal(t, "SELECT sessions ->> '$[*].key' FROM runs", parser.String(stmt))
	require.Len(t, diags, 1)
	require.Equal(t, "array-of-struct-access-rewritten", diags[0].Code)
	require.Equal(t, DiagnosticInfo, diags[0].Severity)
	require.Contains(t, diags[0].Message, "sessions.key")
	require.Contains(t, diags[0].Message, "sessions ->> '$[*].key'")
}

func TestRewriteArrayOfStructAccessQualifiedColumn(t *testing.T) {
	stmt, diags := mustRewriteArrayOfStructAccess(t, "SELECT r.sessions.key FROM runs r")
	require.Equal(t, "SELECT r.sessions ->> '$[*].key' FROM runs AS r", parser.String(stmt))
	require.Len(t, diags, 1)
}

func TestRewriteArrayOfStructAccessInWhereGroupByOrderBy(t *testing.T) {
	stmt, diags := mustRewriteArrayOfStructAccess(t,
		"SELECT sessions.key FROM runs WHERE sessions.id = 'x' GROUP BY sessions.key ORDER BY sessions.key")
	require.Equal(t,
		"SELECT sessions ->> '$[*].key' FROM runs WHERE sessions ->> '$[*].id' = 'x' GROUP BY sessions ->> '$[*].key' ORDER BY sessions ->> '$[*].key'",
		parser.String(stmt),
	)
	// One diagnostic per rewritten occurrence, not deduplicated by column
	// -- sessions.key appears 3 times (SELECT/GROUP BY/ORDER BY), sessions.id once.
	require.Len(t, diags, 4)
}

func TestRewriteArrayOfStructAccessInsideFunctionCall(t *testing.T) {
	stmt, diags := mustRewriteArrayOfStructAccess(t, "SELECT UPPER(sessions.key) FROM runs")
	require.Equal(t, "SELECT UPPER(sessions ->> '$[*].key') FROM runs", parser.String(stmt))
	require.Len(t, diags, 1)
}

func TestRewriteArrayOfStructAccessInJoinCondition(t *testing.T) {
	// The rewrite still finds sessions.key on the correct side (runs) of
	// a JOIN, resolved against the join's own combined scope.
	stmt, diags := mustRewriteArrayOfStructAccess(t,
		"SELECT run_id FROM runs JOIN events ON runs.run_id = events.id AND sessions.key = 'x'")
	require.Equal(t,
		"SELECT run_id FROM runs INNER JOIN events ON runs.run_id = events.id AND sessions ->> '$[*].key' = 'x'",
		parser.String(stmt),
	)
	require.Len(t, diags, 1)
}

func TestRewriteArrayOfStructAccessInCorrelatedSubquery(t *testing.T) {
	stmt, diags := mustRewriteArrayOfStructAccess(t,
		"SELECT run_id FROM runs WHERE run_id IN (SELECT id FROM events WHERE status = runs.sessions.key)")
	require.Equal(t,
		"SELECT run_id FROM runs WHERE run_id IN (SELECT id FROM events WHERE status = runs.sessions ->> '$[*].key')",
		parser.String(stmt),
	)
	require.Len(t, diags, 1)
}

func TestRewriteArrayOfStructAccessInCTE(t *testing.T) {
	stmt, diags := mustRewriteArrayOfStructAccess(t,
		"WITH x AS (SELECT sessions.key AS k FROM runs) SELECT k FROM x")
	require.Equal(t,
		"WITH x AS (SELECT sessions ->> '$[*].key' AS k FROM runs) SELECT k FROM x",
		parser.String(stmt),
	)
	require.Len(t, diags, 1)
}

func TestRewriteArrayOfStructAccessLeavesGenuineJSONColumnsAlone(t *testing.T) {
	// data.function_id (events) and inngest.score (runs) are genuine JSON
	// columns -- DuckDB's dot operator already works on those natively
	// (unlike a real STRUCT(...)[] column), so this rewrite must leave
	// them untouched and emit no diagnostic.
	stmt, diags := mustRewriteArrayOfStructAccess(t, "SELECT data.function_id FROM events")
	require.Equal(t, "SELECT data.function_id FROM events", parser.String(stmt))
	require.Empty(t, diags)

	stmt, diags = mustRewriteArrayOfStructAccess(t, "SELECT runs.inngest.score FROM runs")
	require.Equal(t, "SELECT runs.inngest.score FROM runs", parser.String(stmt))
	require.Empty(t, diags)
}

func TestRewriteArrayOfStructAccessLeavesTableQualifiedColumnAlone(t *testing.T) {
	// events.id is table.column, not column.field -- must not be
	// mistaken for array-of-struct access.
	stmt, diags := mustRewriteArrayOfStructAccess(t, "SELECT events.id FROM events")
	require.Equal(t, "SELECT events.id FROM events", parser.String(stmt))
	require.Empty(t, diags)
}
