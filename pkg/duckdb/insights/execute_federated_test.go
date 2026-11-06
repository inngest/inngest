package insights_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/federate"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/stretchr/testify/require"
)

// cannedStreamer serves each table's delta rows (values by column name), or
// blocks until ctx ends when block is set.
type cannedStreamer struct {
	rows  map[federate.Table][]map[string]any
	block bool
}

func (s cannedStreamer) Stream(_ context.Context, req federate.DeltaRequest) (federate.BatchReader, error) {
	if s.block {
		return blockingReader{schema: req.Columns}, nil
	}
	cols := make([][]any, len(req.Columns))
	for i, c := range req.Columns {
		cols[i] = []any{}
		for _, row := range s.rows[req.Table] {
			cols[i] = append(cols[i], row[c.Name])
		}
	}
	return federate.NewSliceReader(req.Columns, &federate.Batch{Schema: req.Columns, Columns: cols}), nil
}

type blockingReader struct{ schema []federate.Column }

func (r blockingReader) Schema() []federate.Column { return r.schema }
func (r blockingReader) Close() error              { return nil }
func (r blockingReader) Next(ctx context.Context) (*federate.Batch, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// TestExecuteFederated checks ExecuteFederated runs a transpiled query over
// lake ∪ delta (the rendered-SQL guard passing its federated render), under
// Execute's own time limit.
func TestExecuteFederated(t *testing.T) {
	db := newTestQuackDuckDB(t)
	account, env, app, fn := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	w := time.Now().UTC().Truncate(time.Millisecond).Add(-time.Hour)
	queued := w.Add(-time.Minute)

	_, err := db.ExecContext(t.Context(), `INSERT INTO inngest.runs
		(account_id, env_id, run_id, queued_at, scheduled_at, started_at, ended_at, app_id, app_name, function_id, function_slug,
		 status, attributes, inputs, output, bucket_at)
		VALUES (?::UUID, ?::UUID, 'run-lake', ?, ?, ?, ?, ?::UUID, 'app', ?::UUID, 'fn', 'Completed', '{}'::JSON::VARIANT, '[]'::JSON::VARIANT, NULL, ?)`,
		account.String(), env.String(), queued, queued, queued, queued, app.String(), fn.String(), w.Add(-time.Second))
	require.NoError(t, err)
	delta := map[federate.Table][]map[string]any{federate.TableRuns: {{
		"account_id": account, "env_id": env, "run_id": "run-delta", "app_id": app, "app_name": "app",
		"function_id": fn, "function_slug": "fn", "queued_at": queued, "started_at": queued, "status": "Running",
		"attributes": `{}`, "inputs": `[]`, "event_ids": []string{"evt-1"}, "bucket_at": w.Add(time.Second),
	}}}

	query := federate.Query{
		AccountID: account, EnvID: env, From: w.Add(-24 * time.Hour), To: w.Add(time.Hour),
		Lake: federate.LakeSource{Catalog: "inngest"}, Watermark: w, RowCap: 100,
	}
	tr, err := insights.Transpile(`SELECT run_id, status FROM runs ORDER BY run_id`, account, env)
	require.NoError(t, err)

	t.Run("lake and delta", func(t *testing.T) {
		exec := &federate.Executor{DB: db, Delta: cannedStreamer{rows: delta}}
		res, err := insights.ExecuteFederated(t.Context(), exec, tr, query)
		require.NoError(t, err)
		require.Equal(t, [][]any{{"run-delta", "Running"}, {"run-lake", "Completed"}}, res.Rows)
	})

	t.Run("user CTEs can't collide w/ the executor's generated names", func(t *testing.T) {
		require.True(t, strings.HasPrefix(federate.SourceName(federate.TableRuns), insights.TableCTEPrefix))
		exec := &federate.Executor{DB: db, Delta: cannedStreamer{rows: delta}}
		for _, name := range []string{"__fed_runs", "__delta_runs", "fed_runs"} {
			tr, err := insights.Transpile(`WITH `+name+` AS (SELECT run_id, status FROM runs) SELECT run_id, status FROM `+name+` ORDER BY run_id`, account, env)
			require.NoError(t, err)
			res, err := insights.ExecuteFederated(t.Context(), exec, tr, query)
			require.NoError(t, err, name)
			require.Equal(t, [][]any{{"run-delta", "Running"}, {"run-lake", "Completed"}}, res.Rows, name)
		}

		_, err := insights.Transpile(`WITH `+federate.SourceName(federate.TableRuns)+` AS (SELECT run_id FROM runs) SELECT run_id FROM runs`, account, env)
		require.ErrorContains(t, err, "may not start with")
	})

	t.Run("metadata rolls up across lake and delta", func(t *testing.T) {
		// run-lake's "cost" is in the lake, then replaced by a newer
		// emission in the delta; run-delta's only emission is in the delta.
		_, err := db.ExecContext(t.Context(), `INSERT INTO inngest.run_metadata
			(account_id, env_id, run_id, run_queued_at, span_id, scope, kind, is_user, values, created_at, bucket_at)
			VALUES (?::UUID, ?::UUID, 'run-lake', ?, 'span-lake', 'run', 'cost', true, '{"usd":1}'::JSON::VARIANT, ?, ?),
			       (?::UUID, ?::UUID, 'run-lake', ?, 'span-lake', 'run', 'region', true, '{"name":"us"}'::JSON::VARIANT, ?, ?)`,
			account.String(), env.String(), queued, queued, w.Add(-time.Second),
			account.String(), env.String(), queued, queued, w.Add(-time.Second))
		require.NoError(t, err)
		emission := func(run, span, kind, values string, at time.Time) map[string]any {
			return map[string]any{
				"account_id": account, "env_id": env, "run_id": run, "run_queued_at": queued, "span_id": span,
				"scope": "run", "kind": kind, "is_user": true, "values": values, "created_at": at, "bucket_at": at,
			}
		}
		delta := map[federate.Table][]map[string]any{
			federate.TableRuns: delta[federate.TableRuns],
			federate.TableMetadata: {
				emission("run-lake", "span-lake", "cost", `{"usd":2}`, w.Add(time.Second)),
				emission("run-delta", "span-delta", "cost", `{"usd":3}`, w.Add(time.Second)),
			},
		}
		exec := &federate.Executor{DB: db, Delta: cannedStreamer{rows: delta}}
		want := [][]any{
			{"run-delta", map[string]any{"cost": map[string]any{"usd": float64(3)}}},
			{"run-lake", map[string]any{"cost": map[string]any{"usd": float64(2)}, "region": map[string]any{"name": "us"}}},
		}
		for _, sql := range []string{
			`SELECT run_id, metadata FROM runs ORDER BY run_id`,
			`SELECT run_id, metadata FROM metadata ORDER BY run_id`,
			`SELECT m.run_id, m.metadata FROM runs r JOIN metadata m ON m.run_id = r.run_id ORDER BY m.run_id`,
		} {
			tr, err := insights.Transpile(sql, account, env)
			require.NoError(t, err)
			res, err := insights.ExecuteFederated(t.Context(), exec, tr, query)
			require.NoError(t, err, sql)
			require.Equal(t, want, res.Rows, sql)
		}
	})

	t.Run("steps read federated spans and metadata", func(t *testing.T) {
		at := w.Add(time.Second)
		delta := map[federate.Table][]map[string]any{
			federate.TableSpans: {{
				"account_id": account, "env_id": env, "run_id": "run-delta", "run_queued_at": queued, "app_id": app,
				"app_name": "app", "function_id": fn, "function_slug": "fn", "name": "executor.step", "start_time": at,
				"end_time": at, "trace_id": "trace", "span_id": "span-step", "attributes": `{"_inngest.step.userland.id":"s1","_inngest.step.attempt":0}`,
				"bucket_at": at,
			}},
			federate.TableMetadata: {{
				"account_id": account, "env_id": env, "run_id": "run-delta", "run_queued_at": queued, "span_id": "span-step",
				"scope": "step", "kind": "cost", "is_user": true, "values": `{"usd":4}`, "created_at": at, "bucket_at": at,
			}},
		}
		exec := &federate.Executor{DB: db, Delta: cannedStreamer{rows: delta}}
		for _, table := range []string{"steps", "step_attempts"} {
			tr, err := insights.Transpile(`SELECT step_id, metadata FROM `+table, account, env)
			require.NoError(t, err)
			res, err := insights.ExecuteFederated(t.Context(), exec, tr, query)
			require.NoError(t, err, table)
			require.Equal(t, [][]any{{"s1", map[string]any{"cost": map[string]any{"usd": float64(4)}}}}, res.Rows, table)
		}
	})

	t.Run("time limit", func(t *testing.T) {
		insights.SetExecuteTimeout(t, 500*time.Millisecond)
		exec := &federate.Executor{DB: db, Delta: cannedStreamer{block: true}}
		start := time.Now()
		_, err := insights.ExecuteFederated(context.Background(), exec, tr, query)
		var eerr *insights.ExecutionError
		require.ErrorAs(t, err, &eerr)
		require.ErrorContains(t, err, "time limit")
		require.Less(t, time.Since(start), 10*time.Second)
	})
}
