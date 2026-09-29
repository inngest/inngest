package insights_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/stretchr/testify/require"
)

// TestCELEventAndOutputFiltersMatchWholeExpression applies CELEventFilters
// and CELOutputFilters the way pkg/duckdb/query's GetTraceRuns does -- the
// event fragment wrapped in an inputs-array match, the output fragment as
// a separate conjunct, both ANDed -- and checks the rows that combination
// selects against what the original CEL expression means, so a split that
// narrows or widens a mixed expression fails here.
func TestCELEventAndOutputFiltersMatchWholeExpression(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()

	_, err := db.ExecContext(ctx, `CREATE TEMP TABLE cel_runs (id INTEGER, inputs JSON, output JSON);`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO cel_runs VALUES
		(1, '[{"name":"x","data":{"a":1}}]', '{"data":{"ok":false}}'),
		(2, '[{"name":"y","data":{"a":2}}]', '{"data":{"ok":true}}'),
		(3, '[{"name":"x"}]', '{"data":{"ok":true}}'),
		(4, '[{"name":"z"}]', '{"data":{"ok":false}}'),
		(5, '[{"name":"x","data":{"a":2}},{"name":"y","data":{"a":1}}]', '{"data":{"ok":false}}');`)
	require.NoError(t, err)

	tests := []struct {
		cel  string
		want []int
	}{
		{`event.name == "x" && output.ok == true`, []int{3}},
		{`event.name == "x" || output.ok == true`, []int{1, 2, 3, 5}},
		{`output.ok == true || event.data.a == 2`, []int{2, 3, 5}},
		// Event predicates ANDed together must hold for the same
		// triggering event, so batch run 5 (x with a=2, y with a=1) is out.
		{`(event.name == "x" && event.data.a == 1) || output.ok == true`, []int{1, 2, 3}},
		{`event.name == "y" && (event.data.a == 1 || output.ok == true)`, []int{2, 5}},
		{`(event.name == "x" && (event.data.a == 1 || event.data.a == 3)) || event.name == "z"`, []int{1, 4}},
	}

	for _, test := range tests {
		t.Run(test.cel, func(t *testing.T) {
			exprs := []string{test.cel}
			where := []string{"TRUE"}
			var args []any

			eventFilters, err := insights.CELEventFilters(ctx, exprs)
			require.NoError(t, err)
			eventFrag, eventArgs, err := insights.RenderWhereSQL(eventFilters)
			require.NoError(t, err)
			if eventFrag != "" {
				// Mirrors pkg/duckdb/query's eventCELArrayMatchClause.
				where = append(where, fmt.Sprintf(`len(list_filter(json_transform(inputs, '["JSON"]'), lambda x: %s)) > 0`, eventFrag))
				args = append(args, eventArgs...)
			}

			outputFilters, err := insights.CELOutputFilters(ctx, exprs)
			require.NoError(t, err)
			outputFrag, outputArgs, err := insights.RenderWhereSQL(outputFilters)
			require.NoError(t, err)
			if outputFrag != "" {
				where = append(where, outputFrag)
				args = append(args, outputArgs...)
			}

			rows, err := db.QueryContext(ctx,
				fmt.Sprintf("SELECT id FROM cel_runs WHERE %s ORDER BY id;", strings.Join(where, " AND ")), args...)
			require.NoError(t, err)
			defer rows.Close()
			got := []int{}
			for rows.Next() {
				var id int
				require.NoError(t, rows.Scan(&id))
				got = append(got, id)
			}
			require.NoError(t, rows.Err())
			require.Equal(t, test.want, got)
		})
	}
}
