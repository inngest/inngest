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

func TestRemapTables(t *testing.T) {
	tests := []struct {
		name       string
		sql        string
		wantSQL    string
		wantTables []string
	}{
		{
			name:       "bare table keeps its name as the alias",
			sql:        "SELECT run_id FROM runs",
			wantSQL:    "SELECT run_id FROM __inngest_runs AS runs",
			wantTables: []string{"runs"},
		},
		{
			name:       "explicit alias is preserved",
			sql:        "SELECT r.run_id FROM runs r",
			wantSQL:    "SELECT r.run_id FROM __inngest_runs AS r",
			wantTables: []string{"runs"},
		},
		{
			name:       "join rewrites both sides",
			sql:        "SELECT * FROM runs JOIN events ON runs.run_id = events.id",
			wantSQL:    "SELECT * FROM __inngest_runs AS runs INNER JOIN __inngest_events AS events ON runs.run_id = events.id",
			wantTables: []string{"runs", "events"},
		},
		{
			name:       "union operands",
			sql:        "SELECT run_id FROM runs UNION SELECT run_id FROM extended_trace_spans",
			wantSQL:    "SELECT run_id FROM __inngest_runs AS runs UNION SELECT run_id FROM __inngest_extended_trace_spans AS extended_trace_spans",
			wantTables: []string{"runs", "extended_trace_spans"},
		},
		{
			// Tenant scoping must reach a subquery under = ANY exactly as
			// it reaches one under IN.
			name:       "quantified subquery",
			sql:        "SELECT run_id FROM runs WHERE run_id = ANY (SELECT id FROM events)",
			wantSQL:    "SELECT run_id FROM __inngest_runs AS runs WHERE run_id = ANY (SELECT id FROM __inngest_events AS events)",
			wantTables: []string{"runs", "events"},
		},
		{
			name: "set-op modifier subqueries; a table read several times is listed once",
			sql:  "SELECT run_id FROM runs UNION ALL SELECT run_id FROM runs ORDER BY (SELECT max(id) FROM events) LIMIT 5 OFFSET (SELECT count(*) FROM runs)",
			wantSQL: "SELECT run_id FROM __inngest_runs AS runs UNION ALL SELECT run_id FROM __inngest_runs AS runs " +
				"ORDER BY (SELECT max(id) FROM __inngest_events AS events) LIMIT 5 OFFSET (SELECT count(*) FROM __inngest_runs AS runs)",
			wantTables: []string{"runs", "events"},
		},
		{
			name:       "a user CTE shadowing a table name is left alone",
			sql:        "WITH runs AS (SELECT id AS run_id FROM events) SELECT run_id FROM runs",
			wantSQL:    "WITH runs AS (SELECT id AS run_id FROM __inngest_events AS events) SELECT run_id FROM runs",
			wantTables: []string{"events"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := mustParse(t, tt.sql)
			require.Equal(t, tt.wantTables, remapTables(stmt, logicalTables))
			require.Equal(t, tt.wantSQL, parser.String(stmt))
		})
	}
}

func TestRender(t *testing.T) {
	acct, env := testAccountID.String(), testEnvID.String()
	runsMacro := "__inngest_runs AS NOT MATERIALIZED (SELECT * FROM inngest_local.insights_runs(?, ?))"
	eventsMacro := "__inngest_events AS NOT MATERIALIZED (SELECT * FROM inngest_local.insights_events(?, ?))"

	tests := []struct {
		name     string
		sql      string
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "no WITH: one is added",
			sql:      "SELECT run_id FROM runs LIMIT 5",
			wantSQL:  "WITH " + runsMacro + " SELECT run_id FROM __inngest_runs AS runs LIMIT 5",
			wantArgs: []any{acct, env},
		},
		{
			name: "user WITH: reserved CTEs go first, so user CTEs can read them",
			sql:  "WITH e AS (SELECT id FROM events) SELECT run_id FROM runs JOIN e ON e.id = runs.run_id LIMIT 5",
			wantSQL: "WITH " + eventsMacro + ", " + runsMacro + ", e AS (SELECT id FROM __inngest_events AS events) " +
				"SELECT run_id FROM __inngest_runs AS runs INNER JOIN e ON e.id = runs.run_id LIMIT 5",
			wantArgs: []any{acct, env, acct, env},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Transpile(tt.sql, testAccountID, testEnvID)
			require.NoError(t, err)
			require.Equal(t, tt.wantSQL, res.SQL)
			require.Equal(t, tt.wantArgs, res.Args)
		})
	}

	t.Run("a custom source replaces the bodies, args first", func(t *testing.T) {
		res, err := Transpile("SELECT run_id FROM runs WHERE status = 'Failed' LIMIT 5", testAccountID, testEnvID)
		require.NoError(t, err)
		require.Equal(t, []string{"runs"}, res.ReadTables())
		body, err := parser.ParseString("SELECT * FROM lake_union WHERE x = ?")
		require.NoError(t, err)
		stmt, args, err := res.Render(func(table string) (TableCTE, error) {
			return TableCTE{Select: body, Args: []any{"x"}}, nil
		})
		require.NoError(t, err)
		require.Equal(t, "WITH __inngest_runs AS (SELECT * FROM lake_union WHERE x = ?) "+
			"SELECT run_id FROM __inngest_runs AS runs WHERE status = 'Failed' LIMIT 5", parser.String(stmt))
		require.Equal(t, []any{"x"}, args)
	})

	t.Run("user CTEs may not use the reserved prefix", func(t *testing.T) {
		_, err := Transpile("WITH __inngest_runs AS (SELECT id FROM events) SELECT * FROM __inngest_runs", testAccountID, testEnvID)
		var verr *ValidationError
		require.ErrorAs(t, err, &verr)
		require.Contains(t, verr.Message, "__inngest_")
	})
}

// validateProduct is validate against the product catalog.
func validateProduct(stmt *parser.SelectStatement) (*tableScope, map[string]logicalTable, []Diagnostic, error) {
	return validate(stmt, productCatalog)
}

// DuckDB resolves table names case-insensitively, quoted or not, so a
// differently-cased reference to a logical table must still point at its
// reserved CTE -- left bare, it would skip env/account scoping.
func TestRemapTablesCaseInsensitive(t *testing.T) {
	cases := map[string]string{
		"SELECT run_id FROM RUNS":                                                           "SELECT run_id FROM __inngest_runs AS RUNS",
		`SELECT run_id FROM "Runs"`:                                                         `SELECT run_id FROM __inngest_runs AS Runs`,
		"SELECT * FROM Runs r JOIN EVENTS e ON r.run_id = e.id":                             "SELECT * FROM __inngest_runs AS r INNER JOIN __inngest_events AS e ON r.run_id = e.id",
		"WITH R AS (SELECT run_id FROM RUNS) SELECT * FROM r":                               "WITH R AS (SELECT run_id FROM __inngest_runs AS RUNS) SELECT * FROM r",
		"WITH Runs AS (SELECT 1 AS x) SELECT * FROM RUNS":                                   "WITH Runs AS (SELECT 1 AS x) SELECT * FROM RUNS",
		"SELECT run_id FROM runs WHERE run_id IN (SELECT run_id FROM Extended_Trace_Spans)": "SELECT run_id FROM __inngest_runs AS runs WHERE run_id IN (SELECT run_id FROM __inngest_extended_trace_spans AS Extended_Trace_Spans)",
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			stmt := mustParse(t, in)
			_, _, _, err := validate(stmt, productCatalog)
			require.NoError(t, err)
			remapTables(stmt, logicalTables)
			require.Equal(t, want, parser.String(stmt))
		})
	}
}

// TestTranspileRejectsReservedCTENames checks a user CTE can't take a
// logical table's reserved CTE name, in any case (DuckDB's CTE names are
// case-insensitive, so __INNGEST_runs would shadow __inngest_runs).
func TestTranspileRejectsReservedCTENames(t *testing.T) {
	for _, sql := range []string{
		"WITH __inngest_runs AS (SELECT 1 AS x) SELECT x FROM __inngest_runs",
		"WITH __INNGEST_runs AS (SELECT 1 AS x) SELECT x FROM __INNGEST_runs",
		"WITH __Inngest_Runs AS (SELECT run_id FROM runs) SELECT run_id FROM __Inngest_Runs",
	} {
		t.Run(sql, func(t *testing.T) {
			_, err := Transpile(sql, testAccountID, testEnvID)
			require.ErrorContains(t, err, `CTE names may not start with "__inngest_"`)
		})
	}
}
