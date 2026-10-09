package federate

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestExecutorTopN checks a pushed TopN keeps the lake side to its first
// rows without changing the answer. The fake delta streamer ignores the
// limit, as a streamer that can't apply every predicate exactly would.
func TestExecutorTopN(t *testing.T) {
	withFixture(t, func(t *testing.T, f *fixture) {
		q := f.base
		done := q.Add(time.Hour)
		// Lake: 30 finished runs, a minute apart. Delta: 5 newer runs, 2
		// still running.
		for i := range 30 {
			f.lakeRun(t, f.account, runIDf("lake", i), "Completed", q.Add(-time.Duration(i)*time.Minute), &done)
		}
		for i := range 5 {
			status, ended := "Completed", &done
			if i < 2 {
				status, ended = "Running", nil
			}
			f.deltaRun(runIDf("delta", i), status, q.Add(time.Duration(i+1)*time.Second), ended, "")
		}

		completed := []Predicate{{Column: "status", Op: OpEq, Value: "Completed"}}
		top := TopN{OrderBy: []OrderKey{{Column: "queued_at", Desc: true}, {Column: "run_id"}}, Limit: 5}
		run := func(tn *TopN, sqlText string) [][]string {
			query := Query{
				AccountID: f.account, EnvID: f.env, From: f.from, To: f.to,
				Lake: LakeSource{Catalog: "inngest"}, Watermark: f.watermark(), Tables: []Table{TableRuns},
				Predicates: map[Table][]Predicate{TableRuns: completed}, RowCap: 1000, SQL: sqlText,
			}
			if tn != nil {
				query.TopN = map[Table]TopN{TableRuns: *tn}
			}
			rows, err := f.exec.Query(context.Background(), query)
			require.NoError(t, err)
			defer rows.Close()
			return scanStrings(t, rows)
		}

		list := `SELECT run_id, status FROM runs WHERE status = 'Completed' ORDER BY queued_at DESC, run_id LIMIT 5`
		want := run(nil, list)
		require.Len(t, want, 5)
		require.Equal(t, want, run(&top, list))
		require.Equal(t, 5, f.streamer.reqs[len(f.streamer.reqs)-1].Limit, "the limit reaches the delta")

		// The lake side holds 5 rows, not 30 (the fake delta's 3 completed
		// runs are unlimited).
		count := `SELECT count(*)::VARCHAR, '' FROM runs`
		require.Equal(t, "33", run(nil, count)[0][0])
		require.Equal(t, "8", run(&top, count)[0][0])
	})
}

func TestValidateTopN(t *testing.T) {
	byQueued := []OrderKey{{Column: "queued_at", Desc: true}}
	tests := []struct {
		name  string
		tn    TopN
		preds []Predicate
		want  string
	}{
		{"valid", TopN{OrderBy: byQueued, Limit: 10}, []Predicate{{Column: "status", Op: OpEq, Value: "Failed"}}, ""},
		{"no limit", TopN{OrderBy: byQueued}, nil, "positive limit"},
		{"no order", TopN{Limit: 10}, nil, "positive limit"},
		{"unknown column", TopN{OrderBy: []OrderKey{{Column: "nope"}}, Limit: 10}, nil, "no column"},
		{"UUID order", TopN{OrderBy: []OrderKey{{Column: "function_id"}}, Limit: 10}, nil, "UUID"},
		{"JSON order", TopN{OrderBy: []OrderKey{{Column: "attributes"}}, Limit: 10}, nil, "isn't a scalar"},
		{"JSON key filter", TopN{OrderBy: byQueued, Limit: 10}, []Predicate{{Column: "attributes", Path: "k", Op: OpEq, Value: "v"}}, "isn't a scalar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTopN(TableRuns, tt.tn, tt.preds)
			if tt.want == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.want)
		})
	}
}

// runIDf is a sortable run id.
func runIDf(prefix string, i int) string { return fmt.Sprintf("%s-%03d", prefix, i) }
