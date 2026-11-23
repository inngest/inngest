package federate

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// lineageOf runs sourceColumnsRead over sqlText assembled the way
// deltaColumnsUsed does (the views as CTEs over each table's source), and
// returns each table's columns, sorted ("*" for every column), or nil when
// the lineage gave up.
func lineageOf(t *testing.T, sqlText string, tables ...Table) map[Table][]string {
	t.Helper()
	stmt, _, err := assemble(Query{Select: sel(sqlText), Tables: tables, RunsCollapsed: true}, tables, map[Table]string{}, nil, nil)
	require.NoError(t, err)
	read, err := sourceColumnsRead(stmt, tables)
	if err != nil {
		require.ErrorIs(t, err, errLineage)
		return nil
	}
	out := map[Table][]string{}
	for tbl, d := range read {
		if d.all {
			out[tbl] = []string{"*"}
			continue
		}
		names := []string{}
		for n := range d.names {
			names = append(names, n)
		}
		sort.Strings(names)
		out[tbl] = names
	}
	return out
}

func TestSourceColumnsRead(t *testing.T) {
	spansCols := physicalColumns(TableSpans)
	sort.Strings(spansCols)
	runsCols := physicalColumns(TableRuns)
	sort.Strings(runsCols)
	without := func(cols []string, drop ...string) []string {
		out := []string{}
		for _, c := range cols {
			keep := true
			for _, d := range drop {
				keep = keep && c != d
			}
			if keep {
				out = append(out, c)
			}
		}
		return out
	}

	for _, tc := range []struct {
		name   string
		sql    string
		tables []Table
		want   map[Table][]string // nil: the lineage gives up
	}{
		{
			name: "projection", sql: `SELECT run_id FROM runs`, tables: []Table{TableRuns},
			want: map[Table][]string{TableRuns: {"run_id"}},
		},
		{
			name: "count reads nothing", sql: `SELECT count(*) FROM spans`, tables: []Table{TableSpans},
			want: map[Table][]string{TableSpans: {}},
		},
		{
			// An unnest multiplies the rows, read or not.
			name: "unread unnest", sql: `SELECT count(*) FROM (SELECT unnest(event_ids) FROM runs)`, tables: []Table{TableRuns},
			want: map[Table][]string{TableRuns: {"event_ids"}},
		},
		{
			name: "unread unnest in an expression", sql: `SELECT run_id FROM (SELECT run_id, upper(unnest(event_ids)) AS e FROM runs)`, tables: []Table{TableRuns},
			want: map[Table][]string{TableRuns: {"event_ids", "run_id"}},
		},
		{
			name: "unread unnest in a CTE", sql: `WITH c AS (SELECT run_id, unnest(event_ids) AS e, status FROM runs) SELECT count(*) FROM c`, tables: []Table{TableRuns},
			want: map[Table][]string{TableRuns: {"event_ids"}},
		},
		{
			// A filter literal naming another table's column is just a
			// value (the old plan-based attribution misread it).
			name:   "join w/ a literal naming a column",
			sql:    `SELECT m.span_id, m.kind FROM spans s JOIN metadata m ON m.run_id = s.run_id AND m.span_id = s.span_id WHERE m.kind = 'name' AND s.name = 'kind'`,
			tables: []Table{TableSpans, TableMetadata},
			want: map[Table][]string{
				TableSpans:    {"name", "run_id", "span_id"},
				TableMetadata: {"kind", "run_id", "span_id"},
			},
		},
		{
			name:   "self join",
			sql:    `SELECT a.output FROM spans a JOIN spans b ON a.parent_span_id = b.span_id WHERE b.name = 'x'`,
			tables: []Table{TableSpans},
			want:   map[Table][]string{TableSpans: {"name", "output", "parent_span_id", "span_id"}},
		},
		{
			name: "star", sql: `SELECT * FROM spans`, tables: []Table{TableSpans},
			want: map[Table][]string{TableSpans: spansCols},
		},
		{
			name: "star exclude", sql: `SELECT * EXCEPT (output, error, input, attributes) FROM spans`, tables: []Table{TableSpans},
			want: map[Table][]string{TableSpans: without(spansCols, "output", "error", "input", "attributes")},
		},
		{
			name: "star replace", sql: `SELECT * EXCLUDE (error, input, attributes) REPLACE (upper(name) AS output) FROM spans`, tables: []Table{TableSpans},
			want: map[Table][]string{TableSpans: without(spansCols, "output", "error", "input", "attributes")},
		},
		{
			name: "star rename", sql: `SELECT x FROM (SELECT * RENAME (run_id AS x) FROM spans)`, tables: []Table{TableSpans},
			want: map[Table][]string{TableSpans: {"run_id"}},
		},
		{
			name:   "CTE w/ ORDER BY an alias",
			sql:    `WITH c AS (SELECT run_id, output AS o FROM spans) SELECT run_id FROM c ORDER BY o`,
			tables: []Table{TableSpans},
			want:   map[Table][]string{TableSpans: {"output", "run_id"}},
		},
		{
			name:   "correlated subquery",
			sql:    `SELECT run_id FROM runs r WHERE EXISTS (SELECT 1 FROM spans s WHERE s.run_id = r.run_id AND s.name = 'x')`,
			tables: []Table{TableRuns, TableSpans},
			want:   map[Table][]string{TableRuns: {"run_id"}, TableSpans: {"name", "run_id"}},
		},
		{
			name: "struct field and JSON arrow", sql: `SELECT sessions[1].key, attributes -> 'a', inputs->>'$.b' FROM runs`, tables: []Table{TableRuns},
			want: map[Table][]string{TableRuns: {"attributes", "inputs", "sessions"}},
		},
		{
			name: "lateral unnest", sql: `SELECT e FROM runs, unnest(event_ids) AS e`, tables: []Table{TableRuns},
			want: map[Table][]string{TableRuns: {"event_ids"}},
		},
		{
			name: "GROUP BY ALL reads every item", sql: `SELECT n FROM (SELECT name AS n, count(*) AS c FROM spans GROUP BY ALL)`, tables: []Table{TableSpans},
			want: map[Table][]string{TableSpans: {"name"}},
		},
		{
			name: "DISTINCT reads every item", sql: `SELECT n FROM (SELECT DISTINCT name AS n, output FROM spans)`, tables: []Table{TableSpans},
			want: map[Table][]string{TableSpans: {"name", "output"}},
		},
		{
			name: "a DuckDB-generated name", sql: `SELECT "count_star()" FROM (SELECT count(*) FROM spans)`, tables: []Table{TableSpans},
			want: map[Table][]string{TableSpans: {}},
		},
		{
			// Every column of each operand, which is just its select list.
			name: "union", sql: `SELECT run_id FROM spans UNION ALL SELECT span_id AS run_id FROM metadata`, tables: []Table{TableSpans, TableMetadata},
			want: map[Table][]string{TableSpans: {"run_id"}, TableMetadata: {"span_id"}},
		},
		{
			// #n reads every column in scope rather than counting.
			name: "positional column", sql: `SELECT #13 FROM runs`, tables: []Table{TableRuns},
			want: map[Table][]string{TableRuns: runsCols},
		},
		{
			name: "positional ORDER BY", sql: `SELECT r FROM (SELECT run_id AS r, output FROM runs ORDER BY #2)`, tables: []Table{TableRuns},
			want: map[Table][]string{TableRuns: runsCols},
		},
		{
			name: "positional GROUP BY ROLLUP", sql: `SELECT r FROM (SELECT run_id AS r, output FROM runs GROUP BY ROLLUP(1, 2))`, tables: []Table{TableRuns},
			want: map[Table][]string{TableRuns: {"output", "run_id"}},
		},
		{
			name: "positional GROUP BY CUBE", sql: `SELECT r FROM (SELECT run_id AS r, output FROM runs GROUP BY CUBE(1, 2))`, tables: []Table{TableRuns},
			want: map[Table][]string{TableRuns: {"output", "run_id"}},
		},
		{
			name: "positional GROUP BY GROUPING SETS", sql: `SELECT r FROM (SELECT run_id AS r, output FROM runs GROUP BY GROUPING SETS ((1), (2)))`, tables: []Table{TableRuns},
			want: map[Table][]string{TableRuns: {"output", "run_id"}},
		},
		{
			name:   "star REPLACE over unknown-columns relation still walks replacement expr",
			sql:    `SELECT * REPLACE (s.output AS c) FROM (SELECT count(*) FROM runs) u, spans s`,
			tables: []Table{TableRuns, TableSpans},
			want:   map[Table][]string{TableRuns: {}, TableSpans: spansCols},
		},
		{
			name:   "star REPLACE over unknown-columns relation selected by replaced alias",
			sql:    `SELECT c FROM (SELECT u.* REPLACE (s.output AS c) FROM (SELECT count(*) FROM runs) u, spans s)`,
			tables: []Table{TableRuns, TableSpans},
			want:   map[Table][]string{TableRuns: {}, TableSpans: {"output"}},
		},
		{
			name:   "lambda/arrow on lateral SELECT alias reads underlying column",
			sql:    `SELECT output AS a, a -> 'k' FROM spans`,
			tables: []Table{TableSpans},
			want:   map[Table][]string{TableSpans: {"output"}},
		},
		{
			// Unknown columns in scope: fail closed (nil -> stream all columns).
			name:   "lambda/arrow in scope with unknown columns fails closed",
			sql:    `SELECT a -> 'k' FROM (SELECT count(*) FROM runs) u`,
			tables: []Table{TableRuns},
		},
		{
			// ORDER BY prefers the alias, WHERE the column: both are read.
			name:   "ORDER BY an alias shadowing a column",
			sql:    `WITH t AS (SELECT run_id, output AS status FROM runs ORDER BY status LIMIT 1) SELECT run_id FROM t`,
			tables: []Table{TableRuns},
			want:   map[Table][]string{TableRuns: {"output", "run_id", "status"}},
		},
		{
			// x's columns are known only through its WITH, which sits on a
			// UNION; s.output is the outer query's.
			name:   "a set op's WITH and an outer reference",
			sql:    `SELECT s.span_id FROM spans s WHERE EXISTS (SELECT 1 FROM (WITH c AS (SELECT run_id AS y FROM runs) SELECT y FROM c UNION ALL SELECT run_id FROM runs) x WHERE x.y = s.run_id AND s.output IS NOT NULL)`,
			tables: []Table{TableSpans, TableRuns},
			want:   map[Table][]string{TableSpans: {"output", "run_id", "span_id"}, TableRuns: {"run_id"}},
		},
		{
			// A subquery w/ an unnamed column has unknown columns, which
			// mustn't swallow the outer query's s.output.
			name:   "an outer reference past unknown columns",
			sql:    `SELECT s.span_id FROM spans s WHERE EXISTS (SELECT 1 FROM (SELECT count(*) FROM runs) x WHERE s.output IS NOT NULL)`,
			tables: []Table{TableSpans, TableRuns},
			want:   map[Table][]string{TableSpans: {"output", "span_id"}, TableRuns: {}},
		},
		{
			// IN's subquery is held directly (InExpr.Subquery), not in a
			// SubqueryExpr: its r is spans, not the outer runs.
			name:   "IN subquery w/ a shadowing alias",
			sql:    `SELECT r.run_id, s.span_id FROM runs r JOIN spans s ON r.run_id = s.run_id WHERE r.run_id IN (SELECT r.run_id FROM spans r WHERE r.output IS NOT NULL)`,
			tables: []Table{TableRuns, TableSpans},
			want:   map[Table][]string{TableRuns: {"run_id"}, TableSpans: {"output", "run_id", "span_id"}},
		},
		{
			name:   "a set op's LIMIT subquery",
			sql:    `SELECT span_id FROM spans UNION ALL SELECT span_id FROM spans LIMIT (SELECT count(output) FROM spans)`,
			tables: []Table{TableSpans},
			want:   map[Table][]string{TableSpans: {"output", "span_id"}},
		},
		{
			name:   "a set op's ORDER BY its output",
			sql:    `SELECT span_id FROM spans UNION ALL SELECT kind AS span_id FROM metadata ORDER BY span_id`,
			tables: []Table{TableSpans, TableMetadata},
			want:   map[Table][]string{TableSpans: {"span_id"}, TableMetadata: {"kind"}},
		},
		{
			name: "unknown table", sql: `SELECT a FROM somewhere`, tables: []Table{TableSpans},
		},
		{
			name: "unknown column", sql: `SELECT nope FROM spans`, tables: []Table{TableSpans},
		},
		{
			name: "unread table", sql: `SELECT run_id FROM spans`, tables: []Table{TableSpans, TableMetadata},
			want: map[Table][]string{TableSpans: {"run_id"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, lineageOf(t, tc.sql, tc.tables...))
		})
	}
}

// TestSourceColumnsReadBoundsWork checks a query whose CTEs each read the
// one before twice (exponentially many paths to the source) gives up
// quickly instead of walking every path.
func TestSourceColumnsReadBoundsWork(t *testing.T) {
	q := "WITH c0 AS (SELECT run_id FROM spans)"
	for i := 1; i <= 40; i++ {
		q += fmt.Sprintf(", c%d AS (SELECT a.run_id FROM c%d a JOIN c%d b ON a.run_id = b.run_id)", i, i-1, i-1)
	}
	q += " SELECT run_id FROM c40"
	start := time.Now()
	require.Nil(t, lineageOf(t, q, TableSpans))
	require.Less(t, time.Since(start), 5*time.Second)
}

// TestSourceColumnsReadThroughMacros checks the lineage sees through the
// views.sql macros the Insights transpiler renders against the sources:
// their projections, joins, filters and collapses.
func TestSourceColumnsReadThroughMacros(t *testing.T) {
	src := func(tbl Table) string { return "'" + SourceName(tbl) + "'" }
	for _, tc := range []struct {
		name string
		sql  string
		want map[Table][]string
	}{
		{
			// raw_runs' collapse and run_metadata_rollup's, and the join
			// between them, but not the payloads nobody selected.
			name: "insights_runs",
			sql:  `SELECT run_id, status FROM inngest_local.insights_runs(?, ?, p_runs := ` + src(TableRuns) + `, p_metadata := ` + src(TableMetadata) + `)`,
			want: map[Table][]string{
				TableRuns:     {"account_id", "ended_at", "env_id", "queued_at", "run_id", "started_at", "status"},
				TableMetadata: {"account_id", "created_at", "env_id", "is_user", "kind", "run_id", "scope", "span_id", "values"},
			},
		},
		{
			name: "insights_step_attempts",
			sql:  `SELECT step_id FROM inngest_local.insights_step_attempts(?, ?, p_spans := ` + src(TableSpans) + `, p_metadata := ` + src(TableMetadata) + `)`,
			want: map[Table][]string{
				TableSpans:    {"account_id", "attributes", "env_id", "name", "run_id", "span_id"},
				TableMetadata: {"account_id", "created_at", "env_id", "is_user", "kind", "run_id", "scope", "span_id", "values"},
			},
		},
		{
			// Linell's query: the step_attempts delta must stream output.
			name: "insights_step_attempts w/ a correlated set op",
			sql: `SELECT s.step_id FROM inngest_local.insights_step_attempts(?, ?, p_spans := ` + src(TableSpans) + `, p_metadata := ` + src(TableMetadata) + `) s
				WHERE EXISTS (SELECT 1 FROM (WITH c AS (SELECT run_id AS y FROM inngest_local.insights_runs(?, ?, p_runs := ` + src(TableRuns) + `, p_metadata := ` + src(TableMetadata) + `))
					SELECT y FROM c UNION ALL SELECT run_id FROM inngest_local.insights_runs(?, ?, p_runs := ` + src(TableRuns) + `, p_metadata := ` + src(TableMetadata) + `)) x
				WHERE x.y = s.run_id AND s.output IS NOT NULL)`,
			want: map[Table][]string{
				TableSpans:    {"account_id", "attributes", "env_id", "name", "output", "run_id", "span_id"},
				TableMetadata: {"account_id", "created_at", "env_id", "is_user", "kind", "run_id", "scope", "span_id", "values"},
				TableRuns:     {"account_id", "ended_at", "env_id", "queued_at", "run_id", "started_at", "status"},
			},
		},
		{
			name: "insights_events",
			sql:  `SELECT name FROM inngest_local.insights_events(?, ?, p_events := ` + src(TableEvents) + `) WHERE data->>'$.x' = '1'`,
			want: map[Table][]string{TableEvents: {"account_id", "env_id", "event_data", "event_name"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var tables []Table
			for tbl := range tc.want {
				tables = append(tables, tbl)
			}
			read, err := sourceColumnsRead(sel(tc.sql), tables)
			require.NoError(t, err)
			got := map[Table][]string{}
			for tbl, d := range read {
				require.False(t, d.all, "%s reads every column", tbl)
				names := []string{}
				for n := range d.names {
					names = append(names, n)
				}
				sort.Strings(names)
				got[tbl] = names
			}
			require.Equal(t, tc.want, got)
		})
	}
}
