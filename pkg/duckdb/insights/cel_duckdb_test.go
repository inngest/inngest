package insights_test

import (
	"fmt"
	"strings"
	"testing"

	"encoding/json"

	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/inngest/inngest/pkg/expressions"
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

	// VARIANT, like inngest.runs' own inputs/output columns.
	_, err := db.ExecContext(ctx, `CREATE TEMP TABLE cel_runs (id INTEGER, inputs VARIANT, output VARIANT);`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO cel_runs SELECT id, inputs::JSON, output::JSON FROM (VALUES
		(1, '[{"name":"x","data":{"a":1}}]', '{"data":{"ok":false}}'),
		(2, '[{"name":"y","data":{"a":2}}]', '{"data":{"ok":true}}'),
		(3, '[{"name":"x"}]', '{"data":{"ok":true}}'),
		(4, '[{"name":"z"}]', '{"data":{"ok":false}}'),
		(5, '[{"name":"x","data":{"a":2}},{"name":"y","data":{"a":1}}]', '{"data":{"ok":false}}')) v(id, inputs, output);`)
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
				where = append(where, fmt.Sprintf(`len(list_filter(json_transform(inputs::JSON, '["JSON"]'), lambda x: %s)) > 0`, eventFrag))
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

// TestCELNumericFilterSkipsNonNumericValues checks that a numeric CEL
// comparison treats a row whose JSON value isn't a number (a string, an
// object, an array) as not matching, like a missing key, rather than
// failing the whole query with a conversion error.
func TestCELNumericFilterSkipsNonNumericValues(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()

	_, err := db.ExecContext(ctx, `CREATE TEMP TABLE cel_outputs (id INTEGER, output JSON);`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO cel_outputs VALUES
		(1, '{"data":{"n":5}}'),
		(2, '{"data":{"n":"abc"}}'),
		(3, '{"data":{"n":{"x":1}}}'),
		(4, '{"data":{"n":[1,2]}}'),
		(5, '{"data":{}}'),
		(6, '{"data":{"n":1}}');`)
	require.NoError(t, err)

	filters, err := insights.CELOutputFilters(ctx, []string{`output.n > 2`})
	require.NoError(t, err)
	frag, args, err := insights.RenderWhereSQL(filters)
	require.NoError(t, err)

	rows, err := db.QueryContext(ctx, fmt.Sprintf("SELECT id FROM cel_outputs WHERE %s ORDER BY id;", frag), args...)
	require.NoError(t, err)
	defer rows.Close()
	got := []int{}
	for rows.Next() {
		var id int
		require.NoError(t, rows.Scan(&id))
		got = append(got, id)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []int{1}, got)
}

// TestCELNumericRangeFiltersAreASuperset checks the SQL prefilter for a
// numeric </<=/>/>= keeps every row pkg/expressions would match, notably a
// missing key (ordered below every number) and a JSON null (compared as 0),
// both of which extract as SQL NULL.
func TestCELNumericRangeFiltersAreASuperset(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()

	values := []string{`{}`, `{"n":null}`, `{"n":"abc"}`, `{"n":"1"}`, `{"n":{"x":1}}`, `{"n":true}`, `{"n":-3}`, `{"n":0}`, `{"n":2.5}`, `{"n":10}`}
	_, err := db.ExecContext(ctx, `CREATE TEMP TABLE cel_ranges (id INTEGER, output JSON);`)
	require.NoError(t, err)
	for i, v := range values {
		_, err := db.ExecContext(ctx, `INSERT INTO cel_ranges VALUES (?, ?);`, i, fmt.Sprintf(`{"data":%s}`, v))
		require.NoError(t, err)
	}

	for _, e := range []string{"output.n < 5", "output.n <= 0", "output.n < -5", "output.n > -1", "output.n >= 0", "output.n > 0", "output.n > 2", "output.n >= -0.5"} {
		t.Run(e, func(t *testing.T) {
			filters, err := insights.CELOutputFilters(ctx, []string{e})
			require.NoError(t, err)
			frag, args, err := insights.RenderWhereSQL(filters)
			require.NoError(t, err)
			if frag == "" {
				// No SQL filter at all (e.g. expr can't parse a negative
				// literal), which is trivially a superset.
				frag = "TRUE"
			}
			rows, err := db.QueryContext(ctx, fmt.Sprintf("SELECT id FROM cel_ranges WHERE %s ORDER BY id;", frag), args...)
			require.NoError(t, err)
			got := map[int]bool{}
			for rows.Next() {
				var id int
				require.NoError(t, rows.Scan(&id))
				got[id] = true
			}
			require.NoError(t, rows.Err())
			require.NoError(t, rows.Close())

			eval, err := expressions.NewBooleanEvaluator(ctx, e)
			require.NoError(t, err)
			for i, v := range values {
				var data map[string]any
				require.NoError(t, json.Unmarshal([]byte(v), &data))
				ok, _ := eval.Evaluate(ctx, expressions.NewData(map[string]any{"output": data}))
				if ok {
					require.True(t, got[i], "%s matches %s in Go but the SQL prefilter drops it", e, v)
				}
			}
		})
	}
}
