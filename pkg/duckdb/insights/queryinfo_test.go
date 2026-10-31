package insights

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractQueryInfoSingleTable(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs")
	info := extractQueryInfo(stmt)
	require.Equal(t, "runs", info.PrimaryTable)
	require.Equal(t, []string{"runs"}, info.Tables)
}

func TestExtractQueryInfoJoin(t *testing.T) {
	stmt := mustParse(t, "SELECT * FROM runs JOIN events ON runs.run_id = events.id")
	info := extractQueryInfo(stmt)
	require.Equal(t, "runs", info.PrimaryTable)
	require.Equal(t, []string{"runs", "events"}, info.Tables)
}

func TestExtractQueryInfoDedupes(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs UNION SELECT run_id FROM runs")
	info := extractQueryInfo(stmt)
	require.Equal(t, []string{"runs"}, info.Tables)
}

func TestExtractQueryInfoUnion(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs UNION SELECT run_id FROM extended_trace_spans")
	info := extractQueryInfo(stmt)
	require.Equal(t, "runs", info.PrimaryTable)
	require.Equal(t, []string{"runs", "extended_trace_spans"}, info.Tables)
}

func TestExtractQueryInfoSetOpModifierSubquery(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs UNION SELECT run_id FROM runs ORDER BY (SELECT max(id) FROM events)")
	info := extractQueryInfo(stmt)
	require.Equal(t, []string{"runs", "events"}, info.Tables)
}

func TestExtractQueryInfoCaseInsensitive(t *testing.T) {
	stmt := mustParse(t, `WITH X AS (SELECT run_id FROM RUNS) SELECT * FROM x JOIN "Events" ON x.run_id = "Events".id`)
	info := extractQueryInfo(stmt)
	require.Equal(t, "runs", info.PrimaryTable)
	require.Equal(t, []string{"runs", "events"}, info.Tables)
}

// A CTE only hides a same-named real table within its own WITH's scope.
func TestExtractQueryInfoScopesCTENamesLexically(t *testing.T) {
	stmt := mustParse(t, "SELECT * FROM (WITH runs AS (SELECT 1 AS x) SELECT * FROM runs) a, runs")
	require.Equal(t, []string{"runs"}, extractQueryInfo(stmt).Tables)

	stmt = mustParse(t, "SELECT (SELECT count(*) FROM (WITH events AS (SELECT 1 AS x) SELECT * FROM events) e) AS n FROM events")
	require.Equal(t, []string{"events"}, extractQueryInfo(stmt).Tables)

	// A CTE body doesn't see its own name.
	stmt = mustParse(t, "WITH runs AS (SELECT run_id FROM runs) SELECT * FROM runs")
	require.Equal(t, []string{"runs"}, extractQueryInfo(stmt).Tables)
}
