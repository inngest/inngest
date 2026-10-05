package federate

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// scanStrings reads every row as strings (NULL as "NULL", times as RFC3339).
func scanStrings(t *testing.T, rows interface {
	Columns() ([]string, error)
	Next() bool
	Scan(...any) error
	Err() error
}) [][]string {
	t.Helper()
	cols, err := rows.Columns()
	require.NoError(t, err)
	var out [][]string
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		require.NoError(t, rows.Scan(ptrs...))
		row := make([]string, len(cols))
		for i, v := range vals {
			switch v := v.(type) {
			case nil:
				row[i] = "NULL"
			case time.Time:
				row[i] = v.UTC().Format(time.RFC3339)
			default:
				row[i] = fmt.Sprint(v)
			}
		}
		out = append(out, row)
	}
	require.NoError(t, rows.Err())
	return out
}

// TestExecutorAggregate checks Aggregate against the same GROUP BY run over
// the federated runs view, across lake and delta rows. The fake streamer
// can't aggregate, so this covers the row fallback; chdelta's integration
// tests cover the pushed path against ClickHouse.
func TestExecutorAggregate(t *testing.T) {
	forModes(t, func(t *testing.T, f *fixture) {
		q := f.base.Truncate(time.Hour)
		done := q.Add(time.Minute)
		// Lake: final rows only. Delta: a finished run and an open one.
		f.lakeRun(t, f.account, "run-1", "Completed", q, &done)
		f.lakeRun(t, f.account, "run-2", "Completed", q.Add(20*time.Minute), &done)
		f.lakeRun(t, f.account, "run-3", "Failed", q.Add(40*time.Minute), &done)
		f.deltaRun("run-4", "Completed", q.Add(41*time.Minute), &done, "")
		f.deltaRun("run-5", "Running", q.Add(50*time.Minute), nil, "")

		tests := []struct {
			name  string
			agg   AggregateQuery
			check string // the same aggregation over the runs view
			want  [][]string
		}{
			{
				name: "count by status",
				agg: AggregateQuery{
					GroupBy: []GroupKey{{Column: "status", As: "status"}},
					Aggs:    []Agg{{Func: AggCount, As: "n"}},
					SQL:     "SELECT * FROM agg ORDER BY status",
				},
				check: "SELECT status, count(*) FROM runs GROUP BY ALL ORDER BY status",
				want:  [][]string{{"Completed", "3"}, {"Failed", "1"}, {"Running", "1"}},
			},
			{
				name: "count per 30m of queued_at, with min and max",
				agg: AggregateQuery{
					GroupBy: []GroupKey{{Column: "queued_at", Bucket: 30 * time.Minute, As: "period"}},
					Aggs: []Agg{
						{Func: AggCount, As: "n"},
						{Func: AggMin, Column: "run_id", As: "first"},
						{Func: AggMax, Column: "run_id", As: "last"},
					},
					SQL: "SELECT * FROM agg ORDER BY period",
				},
				check: "SELECT time_bucket(INTERVAL 30 MINUTE, queued_at), count(*), min(run_id), max(run_id) FROM runs GROUP BY ALL ORDER BY 1",
				want: [][]string{
					{q.UTC().Format(time.RFC3339), "2", "run-1", "run-2"},
					{q.Add(30 * time.Minute).UTC().Format(time.RFC3339), "3", "run-3", "run-5"},
				},
			},
			{
				name: "filtered, counting a nullable column",
				agg: AggregateQuery{
					Predicates: []Predicate{{Column: "status", Op: OpIn, Value: []any{"Completed", "Running"}}},
					Aggs:       []Agg{{Func: AggCount, As: "runs"}, {Func: AggCount, Column: "ended_at", As: "ended"}},
				},
				check: "SELECT count(*), count(ended_at) FROM runs WHERE status IN ('Completed', 'Running')",
				want:  [][]string{{"4", "3"}},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				q := tt.agg
				q.AccountID, q.EnvID, q.From, q.To = f.account, f.env, f.from, f.to
				q.Lake, q.Watermark, q.Table, q.RowCap = LakeSource{Catalog: "inngest"}, f.watermark(), TableRuns, 1000
				rows, err := f.exec.Aggregate(context.Background(), q)
				require.NoError(t, err)
				got := scanStrings(t, rows)
				require.NoError(t, rows.Close())
				require.Equal(t, tt.want, got)

				check, err := f.exec.Query(context.Background(), Query{
					AccountID: f.account, EnvID: f.env, From: f.from, To: f.to,
					Lake: LakeSource{Catalog: "inngest"}, Watermark: f.watermark(),
					Tables: []Table{TableRuns}, RowCap: 1000, SQL: tt.check,
				})
				require.NoError(t, err)
				require.Equal(t, got, scanStrings(t, check), "Aggregate and a GROUP BY over the runs view disagree")
				require.NoError(t, check.Close())
			})
		}
	})
}

func TestValidateAggregate(t *testing.T) {
	base := AggregateQuery{Table: TableRuns, Aggs: []Agg{{Func: AggCount, As: "n"}}}
	tests := []struct {
		name    string
		mod     func(q *AggregateQuery)
		wantErr string
	}{
		{"ok", func(q *AggregateQuery) {}, ""},
		{"unknown table", func(q *AggregateQuery) { q.Table = "metadata" }, "unknown table"},
		{"nothing to compute", func(q *AggregateQuery) { q.Aggs = nil }, "no keys or aggregates"},
		{"JSON key", func(q *AggregateQuery) { q.GroupBy = []GroupKey{{Column: "attributes", As: "a"}} }, "isn't a scalar"},
		{"bucket on a string", func(q *AggregateQuery) {
			q.GroupBy = []GroupKey{{Column: "status", Bucket: time.Hour, As: "s"}}
		}, "isn't a timestamp"},
		{"bucket not dividing a day", func(q *AggregateQuery) {
			q.GroupBy = []GroupKey{{Column: "queued_at", Bucket: 7 * time.Hour, As: "w"}}
		}, "dividing a day"},
		{"repeated name", func(q *AggregateQuery) { q.GroupBy = []GroupKey{{Column: "status", As: "n"}} }, "repeated"},
		{"unsupported function", func(q *AggregateQuery) { q.Aggs = []Agg{{Func: "avg", Column: "queued_at", As: "x"}} }, "unsupported"},
		{"output name that isn't an identifier", func(q *AggregateQuery) { q.Aggs = []Agg{{Func: AggCount, As: "n` FROM x --"}} }, "plain identifier"},
		{"min of a UUID", func(q *AggregateQuery) { q.Aggs = []Agg{{Func: AggMin, Column: "app_id", As: "a"}} }, "of UUID"},
		{"JSON path predicate", func(q *AggregateQuery) {
			q.Predicates = []Predicate{{Column: "attributes", Path: "_inngest.function.slug", Op: OpEq, Value: "fn"}}
		}, "isn't a scalar"},
		{"UUID range", func(q *AggregateQuery) {
			q.Predicates = []Predicate{{Column: "app_id", Op: OpLt, Value: "00000000-0000-0000-0000-000000000001"}}
		}, "only takes"},
		{"SQL starting with WITH", func(q *AggregateQuery) { q.SQL = "WITH x AS (SELECT 1) SELECT * FROM x" }, "WITH"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := base
			tt.mod(&q)
			err := validateAggregate(q)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
