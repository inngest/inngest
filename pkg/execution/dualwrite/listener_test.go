package dualwrite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestListenerOnEventReceivedNonBlockingWhenBufferFull(t *testing.T) {
	l := newListenerWithChannels(1, 1, 1) // capacity 1 for each of runs/spans/events

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
	l := newListenerWithChannels(10, 10, 10)

	l.OnFunctionScheduled(context.Background(), sv2.Metadata{}, queue.Item{}, nil)

	select {
	case row := <-l.runs:
		require.Equal(t, "function_scheduled", row["event_type"])
	default:
		t.Fatal("expected a row on the runs channel")
	}
}

// TestNewListenerEndToEndEventFlow proves a row makes it from a hook call,
// through the non-blocking channel send, through a real batcher flush,
// into the events_staging table via a real duckdb subprocess. Compaction
// (rolling staged rows to Parquet) is out of scope for this task — see
// listener.go's NewListener doc comment — so this test stops at the
// staging-table insert.
func TestNewListenerEndToEndEventFlow(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	evt := event.NewBaseTrackedEvent(event.Event{Name: "e2e/test"}, nil)
	l.OnEventReceived(context.Background(), evt)

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM events_staging;")
		_ = row.Scan(&count)
		return count == 1
	}, 2*time.Second, 20*time.Millisecond, "row should land in staging after a batch flush")
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

func rowByEventType(t *testing.T, rows []map[string]any, eventType string) map[string]any {
	t.Helper()
	for _, row := range rows {
		if row["event_type"] == eventType {
			return row
		}
	}
	t.Fatalf("no row with event_type %q in %v", eventType, rows)
	return nil
}

// TestNewListenerEndToEndRunFlow is the runs_staging counterpart to
// TestNewListenerEndToEndEventFlow: runs_staging had no end-to-end coverage
// against the real migrated schema at all, so nothing proved the rows the
// hooks build actually satisfy the table's columns and NOT NULL constraints.
// This drives the real OnFunctionScheduled/Started/Finished hooks through a
// real duckdb subprocess and asserts every column lands.
func TestNewListenerEndToEndRunFlow(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	md := testMetadata(t)
	ctx := context.Background()
	l.OnFunctionScheduled(ctx, md, queue.Item{}, nil)
	l.OnFunctionStarted(ctx, md, queue.Item{}, nil)
	l.OnFunctionFinished(ctx, md, queue.Item{}, nil, statev1.DriverResponse{})

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(ctx, "SELECT count(*) FROM runs_staging;")
		_ = row.Scan(&count)
		return count == 3
	}, 5*time.Second, 20*time.Millisecond, "rows should land in runs_staging after a batch flush")

	rows := selectRows(t, db,
		"SELECT run_id, function_id, account_id, workspace_id, event_type, status, year, month FROM runs_staging;")
	require.Len(t, rows, 3)

	ts := ulid.Time(md.ID.RunID.Time())
	for _, eventType := range []string{"function_scheduled", "function_started", "function_finished"} {
		row := rowByEventType(t, rows, eventType)
		require.Equal(t, md.ID.RunID.String(), row["run_id"], eventType)
		require.Equal(t, md.ID.FunctionID.String(), row["function_id"], eventType)
		require.Equal(t, md.ID.Tenant.AccountID.String(), row["account_id"], eventType)
		require.Equal(t, md.ID.Tenant.EnvID.String(), row["workspace_id"], eventType)
		require.EqualValues(t, ts.Year(), row["year"], eventType)
		require.EqualValues(t, int(ts.Month()), row["month"], eventType)
	}

	// OnFunctionFinished is the only runs hook that populates status.
	require.Equal(t, "completed", rowByEventType(t, rows, "function_finished")["status"])
	require.Nil(t, rowByEventType(t, rows, "function_scheduled")["status"])
}

// TestNewListenerEndToEndGeneratorOpcodeHooks covers the sleep/wait/invoke
// hooks added to execution.SyncLifecycleListener, proving their rows satisfy
// run_spans_staging's real schema — including the optional step_name and error
// columns, which arrive on only some of the rows in a single batch.
func TestNewListenerEndToEndGeneratorOpcodeHooks(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	md := testMetadata(t)
	ctx := context.Background()

	l.OnSleep(ctx, md, queue.Item{}, statev1.GeneratorOpcode{Name: "my-sleep"}, time.Now().Add(time.Minute))
	l.OnWaitForEvent(ctx, md, queue.Item{}, statev1.GeneratorOpcode{Name: "my-wait"}, statev1.Pause{})
	l.OnWaitForEventResumed(ctx, md, statev1.Pause{StepName: "my-wait"}, execution.ResumeRequest{IsTimeout: true})
	l.OnWaitForSignal(ctx, md, queue.Item{}, statev1.GeneratorOpcode{Name: "my-signal"}, statev1.Pause{})
	l.OnWaitForSignalResumed(ctx, md, statev1.Pause{}, execution.ResumeRequest{StepName: "my-signal"})
	l.OnInvokeFunction(ctx, md, queue.Item{}, statev1.GeneratorOpcode{Name: "my-invoke"}, event.Event{Name: "inngest/function.invoked"})
	l.OnInvokeFunctionResumed(ctx, md, statev1.Pause{StepName: "my-invoke"}, execution.ResumeRequest{})
	l.OnStepGatewayRequestFinished(ctx, md, queue.Item{}, inngest.Edge{}, statev1.GeneratorOpcode{Name: "my-gateway"}, nil, &statev1.UserError{Message: "boom"})

	wantStepNames := map[string]string{
		"sleep":                         "my-sleep",
		"wait_for_event":                "my-wait",
		"wait_for_event_resumed":        "my-wait",
		"wait_for_signal":               "my-signal",
		"wait_for_signal_resumed":       "my-signal",
		"invoke_function":               "my-invoke",
		"invoke_function_resumed":       "my-invoke",
		"step_gateway_request_finished": "my-gateway",
	}

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(ctx, "SELECT count(*) FROM run_spans_staging;")
		_ = row.Scan(&count)
		return count == len(wantStepNames)
	}, 5*time.Second, 20*time.Millisecond, "every generator-opcode hook should stage a run_spans row")

	rows := selectRows(t, db, "SELECT event_type, step_name, error, run_id, year, month FROM run_spans_staging;")
	require.Len(t, rows, len(wantStepNames))

	ts := ulid.Time(md.ID.RunID.Time())
	for eventType, wantStep := range wantStepNames {
		row := rowByEventType(t, rows, eventType)
		require.Equal(t, md.ID.RunID.String(), row["run_id"], eventType)
		require.Equal(t, wantStep, row["step_name"], eventType)
		require.EqualValues(t, ts.Year(), row["year"], eventType)
		require.EqualValues(t, int(ts.Month()), row["month"], eventType)
	}

	// A timed-out resume records the timeout in the error column; a normal
	// resume leaves it NULL.
	require.Equal(t, "timeout", rowByEventType(t, rows, "wait_for_event_resumed")["error"])
	require.Nil(t, rowByEventType(t, rows, "invoke_function_resumed")["error"])
	require.Equal(t, "boom", rowByEventType(t, rows, "step_gateway_request_finished")["error"])
}
