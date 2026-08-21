package dualwrite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// eventRow builds a row shaped exactly like the listener's OnEventReceived
// output (see eventPartitionFields in listener.go): every real row includes
// account_id/workspace_id/year/month unconditionally, and events_staging's
// year/month columns are NOT NULL, so a test row omitting them would trip a
// real constraint violation against the actual duckdb subprocess.
func eventRow(id, name string) map[string]any {
	return map[string]any{
		"event_id":     id,
		"event_name":   name,
		"account_id":   "acct-1",
		"workspace_id": "ws-1",
		"year":         2026,
		"month":        8,
		"occurred_at":  time.Now().UTC(),
	}
}

func TestBatcherFlushesOnSize(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	ch := make(chan map[string]any, 10)
	b := newBatcher(db, "events_staging", ch, batcherOpts{maxSize: 2, flushInterval: time.Hour})
	go b.run(t.Context())
	defer b.stop()

	ch <- eventRow("1", "a")
	ch <- eventRow("2", "b")

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM events_staging;")
		if err := row.Scan(&count); err != nil {
			return false
		}
		return count == 2
	}, 2*time.Second, 50*time.Millisecond)
}

func TestBatcherFlushesOnTimeout(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	ch := make(chan map[string]any, 10)
	b := newBatcher(db, "events_staging", ch, batcherOpts{maxSize: 100, flushInterval: 50 * time.Millisecond})
	go b.run(t.Context())
	defer b.stop()

	ch <- eventRow("1", "a")

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM events_staging;")
		if err := row.Scan(&count); err != nil {
			return false
		}
		return count == 1
	}, 2*time.Second, 20*time.Millisecond)
}

// TestBatcherHandlesMixedColumnsAcrossRowsInABatch is a regression test for
// a real bug found while fixing this batcher: insert() used to build its
// column list from rows[0]'s keys alone. Several listener hooks only add an
// optional key conditionally (OnStepScheduled's step_name, OnStepFinished's
// error), so a run_spans_staging batch can freely mix rows with different
// key sets. With the old rows[0]-only logic, a step_started row (no
// step_name key) landing first in the batch would silently drop step_name
// from the INSERT for every row in the batch — including a later
// step_scheduled row that *did* have a step_name value. This proves the
// fixed union-of-keys column list preserves every row's data regardless of
// its position in the batch.
func TestBatcherHandlesMixedColumnsAcrossRowsInABatch(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	ch := make(chan map[string]any, 10)
	b := newBatcher(db, "run_spans_staging", ch, batcherOpts{maxSize: 2, flushInterval: time.Hour})
	go b.run(t.Context())
	defer b.stop()

	// step_started row first, deliberately without a step_name key, so a
	// rows[0]-only column list would omit step_name from the whole batch.
	ch <- map[string]any{
		"run_id":     "run-1",
		"event_type": "step_started",
		"account_id": "acct-1", "workspace_id": "ws-1", "year": 2026, "month": 8,
		"created_at": time.Now().UTC(),
	}
	ch <- map[string]any{
		"run_id":     "run-1",
		"event_type": "step_scheduled",
		"step_name":  "my-step",
		"account_id": "acct-1", "workspace_id": "ws-1", "year": 2026, "month": 8,
		"created_at": time.Now().UTC(),
	}

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM run_spans_staging;")
		if err := row.Scan(&count); err != nil {
			return false
		}
		return count == 2
	}, 2*time.Second, 50*time.Millisecond)

	var stepName sql.NullString
	require.NoError(t, db.QueryRowContext(context.Background(),
		"SELECT step_name FROM run_spans_staging WHERE event_type = 'step_scheduled';",
	).Scan(&stepName))
	require.True(t, stepName.Valid, "step_scheduled row's step_name should not have been dropped")
	require.Equal(t, "my-step", stepName.String)

	require.NoError(t, db.QueryRowContext(context.Background(),
		"SELECT step_name FROM run_spans_staging WHERE event_type = 'step_started';",
	).Scan(&stepName))
	require.False(t, stepName.Valid, "step_started row never had a step_name")
}
