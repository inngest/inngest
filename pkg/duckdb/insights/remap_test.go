package insights

import (
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/parser"
	"github.com/stretchr/testify/require"
)

var (
	testAccountID = uuid.MustParse("00000000-0000-4000-a000-000000000000")
	testEnvID     = uuid.MustParse("00000000-0000-4000-b000-000000000000")
)

func TestRemapTablesBareTable(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs")
	args := remapTables(stmt, testAccountID, testEnvID)

	require.Equal(t, []any{testAccountID.String(), testEnvID.String()}, args)
	require.Equal(t,
		"SELECT run_id FROM inngest.insights_runs(?, ?) AS runs",
		parser.String(stmt),
	)
}

func TestRemapTablesPreservesExplicitAlias(t *testing.T) {
	stmt := mustParse(t, "SELECT r.run_id FROM runs r")
	remapTables(stmt, testAccountID, testEnvID)

	require.Equal(t,
		"SELECT r.run_id FROM inngest.insights_runs(?, ?) AS r",
		parser.String(stmt),
	)
}

func TestRemapTablesJoinRewritesBothSides(t *testing.T) {
	stmt := mustParse(t, "SELECT * FROM runs JOIN events ON runs.run_id = events.id")
	args := remapTables(stmt, testAccountID, testEnvID)

	require.Equal(t, []any{testAccountID.String(), testEnvID.String(), testAccountID.String(), testEnvID.String()}, args)
	require.Equal(t,
		"SELECT * FROM inngest.insights_runs(?, ?) AS runs INNER JOIN inngest.insights_events(?, ?) AS events ON runs.run_id = events.id",
		parser.String(stmt),
	)
}

func TestRemapTablesUnion(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs UNION SELECT run_id FROM extended_trace_spans")
	args := remapTables(stmt, testAccountID, testEnvID)
	require.Equal(t, []any{testAccountID.String(), testEnvID.String(), testAccountID.String(), testEnvID.String()}, args)
}
