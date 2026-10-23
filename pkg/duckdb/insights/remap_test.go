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
	runsMacro := "__inngest_runs AS NOT MATERIALIZED (SELECT * FROM inngest.insights_runs(?, ?))"
	eventsMacro := "__inngest_events AS NOT MATERIALIZED (SELECT * FROM inngest.insights_events(?, ?))"

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
		sql, args, err := res.Render(func(table string) (TableCTE, error) {
			return TableCTE{SQL: "SELECT * FROM lake_union WHERE x = ?", Args: []any{"x"}}, nil
		})
		require.NoError(t, err)
		require.Equal(t, "WITH __inngest_runs AS (SELECT * FROM lake_union WHERE x = ?) "+
			"SELECT run_id FROM __inngest_runs AS runs WHERE status = 'Failed' LIMIT 5", sql)
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
