package insights_test

import (
	"context"
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
