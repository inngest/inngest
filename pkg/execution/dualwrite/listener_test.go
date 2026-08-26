package dualwrite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestListenerOnEventReceivedNonBlockingWhenBufferFull(t *testing.T) {
	l := newListenerWithChannels(1, 1) // capacity 1 for each of runs/events

	evt := event.NewBaseTrackedEvent(event.Event{Name: "test"}, nil)

	// Fill the events channel to capacity.
	l.OnEventReceived(context.Background(), evt)
	require.Equal(t, int64(0), l.droppedEvents.Load())

	// This second call must return immediately (not block) even though the
	// channel is full, and must increment the dropped counter.
	done := make(chan struct{})
	go func() {
		l.OnEventReceived(context.Background(), evt)
		close(done)
	}()

	select {
	case <-done:
	case <-timeAfter():
		t.Fatal("OnEventReceived blocked on a full channel")
	}
	require.Equal(t, int64(1), l.droppedEvents.Load())
}

func TestListenerOnFunctionScheduledEnqueuesRunRow(t *testing.T) {
	l := newListenerWithChannels(10, 10)
	md := testMetadata(t)

	l.OnFunctionScheduled(context.Background(), md, queue.Item{}, nil)

	select {
	case row := <-l.runs:
		require.Equal(t, md.ID.RunID, row["run_id"])
		require.Equal(t, md.ID.FunctionID, row["function_id"])
		require.Equal(t, enums.StepStatusQueued, row["status"])
	default:
		t.Fatal("expected a row on the runs channel")
	}
}

// TestListenerOnFunctionScheduledSetsEventIDsFromMetadataConfig proves
// runCommonFields sources event_ids from md.Config.EventIDs — the run's own
// persisted trigger event ID list (set once at Schedule time in
// pkg/execution/executor/executor.go, round-tripped through state on every
// subsequent load) — as a real []string, stored as a DuckDB VARCHAR[] (see
// pkg/db/duckdb/literal.go's []string encoding).
func TestListenerOnFunctionScheduledSetsEventIDsFromMetadataConfig(t *testing.T) {
	l := newListenerWithChannels(10, 10)
	md := testMetadata(t)
	evt1, evt2 := ulid.MustNew(ulid.Now(), rand.Reader), ulid.MustNew(ulid.Now(), rand.Reader)
	md.Config.EventIDs = []ulid.ULID{evt1, evt2}

	l.OnFunctionScheduled(context.Background(), md, queue.Item{}, nil)

	select {
	case row := <-l.runs:
		require.Equal(t, []string{evt1.String(), evt2.String()}, row["event_ids"])
	default:
		t.Fatal("expected a row on the runs channel")
	}
}

// TestListenerOnFunctionScheduledOmitsEventIDsForCronRuns proves a
// cron-triggered run (no triggering event, so md.Config.EventIDs is empty)
// never sets event_ids at all, rather than an empty string — matching how
// "status" is handled for the same "column not applicable to this hook"
// reasoning elsewhere in this file.
func TestListenerOnFunctionScheduledOmitsEventIDsForCronRuns(t *testing.T) {
	l := newListenerWithChannels(10, 10)
	md := testMetadata(t) // Config.EventIDs left empty, as a cron run would have it

	l.OnFunctionScheduled(context.Background(), md, queue.Item{}, nil)

	select {
	case row := <-l.runs:
		_, hasEventIDs := row["event_ids"]
		require.False(t, hasEventIDs, "a cron-triggered run must not set event_ids")
	default:
		t.Fatal("expected a row on the runs channel")
	}
}

// TestNewListenerEndToEndEventFlow proves a row makes it from a hook call,
// through the non-blocking channel send, through a real batcher flush,
// into the inngest.events table via a real duckdb subprocess. Compaction
// (rolling staged rows to Parquet) is out of scope for this task — see
// listener.go's NewListener doc comment — so this test stops at the insert.
func TestNewListenerEndToEndEventFlow(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	evt := event.NewBaseTrackedEvent(event.Event{Name: "e2e/test"}, nil)
	l.OnEventReceived(context.Background(), evt)

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.events;")
		_ = row.Scan(&count)
		return count == 1
	}, 2*time.Second, 20*time.Millisecond, "row should land in inngest.events after a batch flush")
}

// testMetadata builds metadata with real IDs so the row's partition columns
// (account_id/workspace_id/year/month, derived from the run ID's ULID
// timestamp in Go — see runPartitionFields) carry the values a real run would
// produce, rather than zero values that would hide a NOT NULL problem.
func testMetadata(t *testing.T) sv2.Metadata {
	t.Helper()
	return sv2.Metadata{
		ID: sv2.ID{
			RunID:      ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader),
			FunctionID: uuid.New(),
			Tenant: sv2.Tenant{
				AccountID: uuid.New(),
				EnvID:     uuid.New(),
			},
		},
	}
}

// selectRows reads a query's results keyed by column name rather than by
// position. The driver derives driver.Rows.Columns() from a Go map's key set
// (see newMapRows in pkg/db/duckdb/rows.go), so the column order it reports
// is not the order the SELECT asked for — positional Scan against a
// multi-column read is unreliable. Reads are explicitly out of scope for this
// phase of the POC (the query-layer spec owns them), so these tests work by
// name instead of expanding scope here.
func selectRows(t *testing.T, db *sql.DB, query string) []map[string]any {
	t.Helper()

	rows, err := db.QueryContext(context.Background(), query)
	require.NoError(t, err)
	defer rows.Close()

	cols, err := rows.Columns()
	require.NoError(t, err)

	var out []map[string]any
	for rows.Next() {
		dest := make([]any, len(cols))
		for i := range dest {
			dest[i] = new(any)
		}
		require.NoError(t, rows.Scan(dest...))

		row := make(map[string]any, len(cols))
		for i, col := range cols {
			row[col] = *(dest[i].(*any))
		}
		out = append(out, row)
	}
	require.NoError(t, rows.Err())
	return out
}

// rowByStatus finds the one row in rows whose "status" column equals want.
// inngest.runs has no per-hook discriminator column — each of
// OnFunctionScheduled/Started/Finished/Cancelled inserts its own row for the
// same run_id, every one carrying a status (Queued/Running/Completed/...:
// see enums.StepStatus), so status doubles as the discriminator here.
func rowByStatus(t *testing.T, rows []map[string]any, want string) map[string]any {
	t.Helper()
	for _, row := range rows {
		status, _ := row["status"].(string)
		if status == want {
			return row
		}
	}
	t.Fatalf("no row with status %q in %v", want, rows)
	return nil
}

// TestNewListenerEndToEndRunFlow is the inngest.runs counterpart to
// TestNewListenerEndToEndEventFlow: inngest.runs had no end-to-end coverage
// against the real migrated schema at all, so nothing proved the rows the
// hooks build actually satisfy the table's columns and NOT NULL constraints.
// This drives the real OnFunctionScheduled/Started/Finished hooks through a
// real duckdb subprocess and asserts every column lands. Each hook inserts
// its own row (append-only, not an update-in-place — see listener.go), so
// three rows for the same run_id is the correct, expected outcome.
func TestNewListenerEndToEndRunFlow(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	md := testMetadata(t)
	ctx := context.Background()
	l.OnFunctionScheduled(ctx, md, queue.Item{}, nil)
	l.OnFunctionStarted(ctx, md, queue.Item{}, nil)
	l.OnFunctionFinished(ctx, md, queue.Item{}, nil, statev1.DriverResponse{}, time.Now())

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(ctx, "SELECT count(*) FROM inngest.runs;")
		_ = row.Scan(&count)
		return count == 3
	}, 5*time.Second, 20*time.Millisecond, "rows should land in inngest.runs after a batch flush")

	rows := selectRows(t, db,
		"SELECT run_id, function_id, account_id, env_id, status, started_at, ended_at FROM inngest.runs;")
	require.Len(t, rows, 3)

	for _, row := range rows {
		require.Equal(t, md.ID.RunID.String(), row["run_id"])
		require.Equal(t, md.ID.FunctionID.String(), row["function_id"])
		require.Equal(t, md.ID.Tenant.AccountID.String(), row["account_id"])
		require.Equal(t, md.ID.Tenant.EnvID.String(), row["env_id"])
	}

	scheduled := rowByStatus(t, rows, enums.StepStatusQueued.String())
	require.Nil(t, scheduled["started_at"])
	require.Nil(t, scheduled["ended_at"])

	started := rowByStatus(t, rows, enums.StepStatusRunning.String())
	require.NotNil(t, started["started_at"])
	require.Nil(t, started["ended_at"])

	finished := rowByStatus(t, rows, enums.StepStatusCompleted.String())
	require.NotNil(t, finished["started_at"])
	require.NotNil(t, finished["ended_at"])
}

// TestListenerEmitsGroupIDOnSpansWithQueueItem proves createSpan sets
// meta.Attrs.GroupID (job.group.id) from queue.Item.GroupID on every span
// built with a QueueItem — see tracing.go's addQueueItemAttrs doc comment
// for why this package must set it explicitly rather than getting it for
// free from pkg/tracing's ambient ExecutionContext the way the real
// executionProcessor does.
func TestListenerEmitsGroupIDOnSpansWithQueueItem(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	md := testMetadata(t)
	item := queue.Item{GroupID: "group-abc-123"}
	stepName := "my-step"
	l.OnStepScheduled(context.Background(), md, item, &stepName, time.Now())

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.run_trace_spans;")
		_ = row.Scan(&count)
		return count == 1
	}, 2*time.Second, 20*time.Millisecond, "span row should land in inngest.run_trace_spans after a batch flush")

	rows := selectRows(t, db, "SELECT attributes FROM inngest.run_trace_spans;")
	require.Len(t, rows, 1)
	// The attributes column is JSON-typed, so the stdio transport's
	// -jsonlines output already embeds it as a native decoded value (a
	// map[string]any), not a string requiring a second json.Unmarshal.
	attrs, ok := rows[0]["attributes"].(map[string]any)
	require.True(t, ok, "attributes column should decode to a map, got %T", rows[0]["attributes"])
	require.Equal(t, item.GroupID, attrs[meta.Attrs.GroupID.Key()])
}

// TestListenerOmitsGroupIDOnSpansWithoutQueueItem proves addQueueItemAttrs
// is a no-op (not a panic, not a bogus empty-string attribute) for the one
// span-building path in this package that never has a queue.Item to draw
// from: createPauseSpan, used only by the *Resumed hooks (OnWaitForEvent
// Resumed/OnWaitForSignalResumed/OnInvokeFunctionResumed), which receive a
// pause/result, not a queue.Item.
func TestListenerOmitsGroupIDOnSpansWithoutQueueItem(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	md := testMetadata(t)
	l.OnWaitForEventResumed(context.Background(), md, statev1.Pause{}, execution.ResumeRequest{})

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.run_trace_spans;")
		_ = row.Scan(&count)
		return count == 1
	}, 2*time.Second, 20*time.Millisecond, "span row should land in inngest.run_trace_spans after a batch flush")

	rows := selectRows(t, db, "SELECT attributes FROM inngest.run_trace_spans;")
	require.Len(t, rows, 1)
	attrs, ok := rows[0]["attributes"].(map[string]any)
	require.True(t, ok, "attributes column should decode to a map, got %T", rows[0]["attributes"])
	_, hasGroupID := attrs[meta.Attrs.GroupID.Key()]
	require.False(t, hasGroupID, "a span with no queue.Item must not set GroupID")
}
