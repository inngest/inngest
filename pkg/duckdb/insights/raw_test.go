package insights_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/federate"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/stretchr/testify/require"
)

func TestTranspileRaw(t *testing.T) {
	acct := "00000000-0000-0000-0000-00000000000a"
	res, err := insights.TranspileRaw(`SELECT account_id, run_id, status FROM runs WHERE account_id = '` + acct + `' AND status = 'Failed'`)
	require.NoError(t, err)
	require.True(t, res.Raw())
	require.Equal(t, "WITH __inngest_runs AS NOT MATERIALIZED (SELECT * FROM inngest.raw_runs()) "+
		"SELECT account_id, run_id, status FROM __inngest_runs AS runs WHERE account_id = '"+acct+"' AND status = 'Failed' LIMIT 1000", res.SQL)
	require.Empty(t, res.Args, "raw mode is unscoped")

	// account_id is an ordinary immutable column there, and Failed a
	// terminal status, so both are pushed.
	tables, preds, _ := res.Federated()
	require.Equal(t, []federate.Table{federate.TableRuns}, tables)
	require.Equal(t, map[federate.Table][]federate.Predicate{federate.TableRuns: {
		{Column: "account_id", Op: federate.OpEq, Value: acct},
		{Column: "status", Op: federate.OpEq, Value: "Failed"},
	}}, preds)

	tests := []struct {
		name    string
		raw     bool
		sql     string
		wantErr string
	}{
		{name: "raw mode exposes run_trace_spans", raw: true, sql: `SELECT span_id, attributes FROM run_trace_spans`},
		{name: "raw mode exposes events", raw: true, sql: `SELECT event_id, event_data FROM events`},
		{name: "raw mode has no product-only tables", raw: true, sql: `SELECT run_id FROM metadata`, wantErr: `unknown table "metadata"`},
		{name: "product mode doesn't expose the raw tables", sql: `SELECT span_id FROM run_trace_spans`, wantErr: `unknown table "run_trace_spans"`},
		{name: "product runs has no account_id column", sql: `SELECT account_id FROM runs`, wantErr: "account_id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.raw {
				_, err = insights.TranspileRaw(tt.sql)
			} else {
				_, err = insights.Transpile(tt.sql, uuid.New(), uuid.New())
			}
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestExecuteRaw(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	now := time.Now().UTC().Truncate(time.Millisecond)
	a, b := uuid.New(), uuid.New()
	insertRunRow(t, db, a, uuid.New(), "run-a", "Completed", now)
	insertRunRow(t, db, b, uuid.New(), "run-b", "Running", now)

	t.Run("reads every tenant", func(t *testing.T) {
		res, err := insights.TranspileRaw(`SELECT run_id FROM runs ORDER BY run_id`)
		require.NoError(t, err)
		out, err := insights.Execute(t.Context(), db, res)
		require.NoError(t, err)
		require.Equal(t, [][]any{{"run-a"}, {"run-b"}}, out.Rows)
	})

	t.Run("a product query can't call the raw macros", func(t *testing.T) {
		_, err := insights.Execute(t.Context(), db, &insights.TranspileResult{SQL: `SELECT run_id FROM inngest.raw_runs()`})
		require.ErrorContains(t, err, "not allowed")
	})
}
