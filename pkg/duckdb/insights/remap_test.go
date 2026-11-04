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

func TestRemapTablesInQuantifiedSubquery(t *testing.T) {
	// Tenant scoping must reach a subquery under = ANY exactly as it
	// reaches one under IN.
	stmt := mustParse(t, "SELECT run_id FROM runs WHERE run_id = ANY (SELECT id FROM events)")
	args := remapTables(stmt, testAccountID, testEnvID)

	require.Equal(t, []any{testAccountID.String(), testEnvID.String(), testAccountID.String(), testEnvID.String()}, args)
	require.Equal(t,
		"SELECT run_id FROM inngest.insights_runs(?, ?) AS runs WHERE run_id = ANY (SELECT id FROM inngest.insights_events(?, ?) AS events)",
		parser.String(stmt),
	)
}

func TestRemapTablesRewritesSetOpModifierSubqueries(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs UNION ALL SELECT run_id FROM runs ORDER BY (SELECT max(id) FROM events) LIMIT 5 OFFSET (SELECT count(*) FROM runs)")
	args := remapTables(stmt, testAccountID, testEnvID)

	require.Len(t, args, 8)
	require.Equal(t,
		"SELECT run_id FROM inngest.insights_runs(?, ?) AS runs UNION ALL SELECT run_id FROM inngest.insights_runs(?, ?) AS runs ORDER BY (SELECT max(id) FROM inngest.insights_events(?, ?) AS events) LIMIT 5 OFFSET (SELECT count(*) FROM inngest.insights_runs(?, ?) AS runs)",
		parser.String(stmt),
	)
}

// DuckDB resolves table names case-insensitively, quoted or not, so a
// differently-cased reference to a logical table must still become its
// macro call -- left bare, it would skip env/account scoping.
func TestRemapTablesCaseInsensitive(t *testing.T) {
	cases := map[string]string{
		"SELECT run_id FROM RUNS":                                                           "SELECT run_id FROM inngest.insights_runs(?, ?) AS RUNS",
		`SELECT run_id FROM "Runs"`:                                                         `SELECT run_id FROM inngest.insights_runs(?, ?) AS Runs`,
		"SELECT * FROM Runs r JOIN EVENTS e ON r.run_id = e.id":                             "SELECT * FROM inngest.insights_runs(?, ?) AS r INNER JOIN inngest.insights_events(?, ?) AS e ON r.run_id = e.id",
		"WITH R AS (SELECT run_id FROM RUNS) SELECT * FROM r":                               "WITH R AS (SELECT run_id FROM inngest.insights_runs(?, ?) AS RUNS) SELECT * FROM r",
		"WITH Runs AS (SELECT 1 AS x) SELECT * FROM RUNS":                                   "WITH Runs AS (SELECT 1 AS x) SELECT * FROM RUNS",
		"SELECT run_id FROM runs WHERE run_id IN (SELECT run_id FROM Extended_Trace_Spans)": "SELECT run_id FROM inngest.insights_runs(?, ?) AS runs WHERE run_id IN (SELECT run_id FROM inngest.insights_extended_trace_spans(?, ?) AS Extended_Trace_Spans)",
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			stmt := mustParse(t, in)
			_, _, _, err := validate(stmt)
			require.NoError(t, err)
			remapTables(stmt, testAccountID, testEnvID)
			require.Equal(t, want, parser.String(stmt))
		})
	}
}
