package insights_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// TestExecuteJSONColumns checks the lake's JSON text columns still come
// back from Insights as JSON: typed JSON, decoded to their values, and
// readable w/ ->>, dot access and json_* (in product and raw mode).
func TestExecuteJSONColumns(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	ctx := t.Context()
	accountID, envID := uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Now(), nil).String()
	insertStepSpan(t, db, accountID, envID, runID, "span-1", "step-1", 2, time.Now())
	_, err := db.ExecContext(ctx,
		`INSERT INTO inngest.events (account_id, env_id, internal_id, received_at, source, event_id, event_name, event_data, event_v, event_ts, event_meta)
		 VALUES (?, ?, 'internal-1', now(), 'api', 'evt-1', 'app/x', ?, '1', now(), '{}');`,
		accountID.String(), envID.String(), json.RawMessage(`{"user":{"id":"u1","n":12345678901},"tags":["a","b"]}`),
	)
	require.NoError(t, err)

	run := func(t *testing.T, tr *insights.TranspileResult) *insights.Result {
		t.Helper()
		res, err := insights.Execute(ctx, db, tr)
		require.NoError(t, err)
		require.Len(t, res.Rows, 1)
		return res
	}

	t.Run("columns", func(t *testing.T) {
		tr, err := insights.Transpile("SELECT data, attributes FROM events, steps", accountID, envID)
		require.NoError(t, err)
		res := run(t, tr)
		require.Equal(t, insights.ColumnTypeJSON, res.Columns[0].Type)
		require.Equal(t, insights.ColumnTypeJSON, res.Columns[1].Type)
		require.Equal(t, map[string]any{
			"user": map[string]any{"id": "u1", "n": float64(12345678901)},
			"tags": []any{"a", "b"},
		}, res.Rows[0][0])
		require.Equal(t, map[string]any{"_inngest.step.userland.id": "step-1", "_inngest.step.attempt": float64(2)}, res.Rows[0][1])
	})

	t.Run("access", func(t *testing.T) {
		tr, err := insights.Transpile(`SELECT data->>'$.user.id', data.user.id, json_extract_string(data, '$.tags[1]'), data->'user', attributes->>'$."_inngest.step.userland.id"'
			FROM events, steps WHERE data->>'$.user.id' = 'u1' AND step_id = 'step-1' AND step_attempt = 2`, accountID, envID)
		require.NoError(t, err)
		res := run(t, tr)
		require.Equal(t, []any{"u1", "u1", "b", map[string]any{"id": "u1", "n": float64(12345678901)}, "step-1"}, res.Rows[0])
		// Dot access is JSON, like DuckDB's own JSON dot.
		require.Equal(t, insights.ColumnTypeJSON, res.Columns[1].Type)
	})

	t.Run("raw", func(t *testing.T) {
		tr, err := insights.TranspileRaw("SELECT event_data, event_data->>'$.user.id' FROM events WHERE event_id = 'evt-1'")
		require.NoError(t, err)
		res := run(t, tr)
		require.Equal(t, insights.ColumnTypeJSON, res.Columns[0].Type)
		require.Equal(t, "u1", res.Rows[0][1])
	})
}
