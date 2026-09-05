package dualwrite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	tracingv3 "github.com/inngest/inngest/pkg/tracing/v3"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

// TestListenerOnFunctionScheduledSetsSessionsFromTriggeringEvents proves
// runCommonFields sources "sessions" from each triggering event's own
// Meta.Sessions (evts is this package's own json.Marshal(event.Event)
// output, per executor.go's Schedule path), sorted by (key, then id) and
// deduped across events — matching pkg/execution/executor's
// normalizeRunSessions, which builds the same shape for the real run
// span's "event.sessions" attribute.
func TestListenerOnFunctionScheduledSetsSessionsFromTriggeringEvents(t *testing.T) {
	l := newListenerWithChannels(10, 10)
	md := testMetadata(t)

	evt1, err := json.Marshal(event.Event{Name: "test", Meta: event.EventMeta{
		Sessions: event.Sessions{"customer": "acme", "cart": "123"},
	}})
	require.NoError(t, err)
	evt2, err := json.Marshal(event.Event{Name: "test", Meta: event.EventMeta{
		Sessions: event.Sessions{"cart": "123"}, // duplicate of evt1's cart session
	}})
	require.NoError(t, err)

	l.OnFunctionScheduled(context.Background(), md, queue.Item{}, []json.RawMessage{evt1, evt2})

	select {
	case row := <-l.runs:
		require.Equal(t, meta.EventSessions{
			{Key: "cart", ID: "123"},
			{Key: "customer", ID: "acme"},
		}, row["sessions"])
	default:
		t.Fatal("expected a row on the runs channel")
	}
}

// TestListenerOnFunctionScheduledOmitsSessionsWhenNoTriggeringEventHasOne
// mirrors TestListenerOnFunctionScheduledOmitsEventIDsForCronRuns' "column
// not applicable to this hook" reasoning: a run with no session-tagged
// triggering event (or none at all) must never set "sessions", not set it
// to an empty value.
func TestListenerOnFunctionScheduledOmitsSessionsWhenNoTriggeringEventHasOne(t *testing.T) {
	l := newListenerWithChannels(10, 10)
	md := testMetadata(t)

	evt, err := json.Marshal(event.Event{Name: "test"})
	require.NoError(t, err)

	l.OnFunctionScheduled(context.Background(), md, queue.Item{}, []json.RawMessage{evt})

	select {
	case row := <-l.runs:
		_, hasSessions := row["sessions"]
		require.False(t, hasSessions, "a run with no session-tagged triggering event must not set sessions")
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
func selectRows(t *testing.T, db *sql.DB, query string, args ...any) []map[string]any {
	t.Helper()

	rows, err := db.QueryContext(context.Background(), query, args...)
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
	l.OnWaitForEventResumed(context.Background(), md, statev1.Pause{}, execution.ResumeRequest{}, time.Now())

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

// TestListenerEmitsEmptyArrayNotNullForSpansWithNoLinks proves
// spanExportRow substitutes an empty slice for a nil span.Links() before
// marshaling, so the links column is always a JSON array ([]) rather than
// sometimes JSON null — no hook in this package ever sets span links today,
// so every span exercises this path.
func TestListenerEmitsEmptyArrayNotNullForSpansWithNoLinks(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	md := testMetadata(t)
	stepName := "my-step"
	l.OnStepScheduled(context.Background(), md, queue.Item{}, &stepName, time.Now())

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.run_trace_spans;")
		_ = row.Scan(&count)
		return count == 1
	}, 2*time.Second, 20*time.Millisecond, "span row should land in inngest.run_trace_spans after a batch flush")

	rows := selectRows(t, db, "SELECT links FROM inngest.run_trace_spans;")
	require.Len(t, rows, 1)
	links, ok := rows[0]["links"].([]any)
	require.True(t, ok, "links column should decode to a slice, got %T", rows[0]["links"])
	require.Empty(t, links)
}

// singleSpanAttrs drives a hook expected to emit exactly one span through a
// real subprocess and returns that span's decoded attributes map.
func singleSpanAttrs(t *testing.T, db *sql.DB, fire func()) map[string]any {
	t.Helper()
	fire()
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
	return attrs
}

// TestListenerSetsWaitForEventAttrsOnPauseStartedSpan proves
// OnWaitForEvent's pause-started marker span carries the opcode-specific
// attributes convertFlatSpanToGQL's WaitForEventStepInfo reads
// (step.wait_for_event.name/if, step.wait.expiry) — previously missing
// entirely, since the hook built its span attrs via a bespoke StepName/
// StepOp-only helper instead of tracing.GeneratorAttrs.
func TestListenerSetsWaitForEventAttrsOnPauseStartedSpan(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	ifExpr := "true"
	gen := statev1.GeneratorOpcode{
		ID:   "step-1",
		Name: "wait-step",
		Op:   enums.OpcodeWaitForEvent,
		Opts: &statev1.WaitForEventOpts{Event: "app/some.event", If: &ifExpr, Timeout: "1h"},
	}

	attrs := singleSpanAttrs(t, db, func() {
		l.OnWaitForEvent(context.Background(), testMetadata(t), queue.Item{}, gen, statev1.Pause{})
	})

	require.Equal(t, "app/some.event", attrs[meta.Attrs.StepWaitForEventName.Key()])
	require.Equal(t, "true", attrs[meta.Attrs.StepWaitForEventIf.Key()])
	require.Equal(t, "step-1", attrs[meta.Attrs.StepID.Key()])
	_, hasExpiry := attrs[meta.Attrs.StepWaitExpiry.Key()]
	require.True(t, hasExpiry, "step.wait.expiry should be set from the opcode's own timeout")
}

// TestListenerSetsSignalAttrsOnPauseStartedSpan mirrors the WaitForEvent
// test above for OnWaitForSignal's step.signal.name attribute.
func TestListenerSetsSignalAttrsOnPauseStartedSpan(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	gen := statev1.GeneratorOpcode{
		ID:   "step-1",
		Name: "signal-step",
		Op:   enums.OpcodeWaitForSignal,
		Opts: &statev1.SignalOpts{Signal: "my-signal", Timeout: "1h"},
	}

	attrs := singleSpanAttrs(t, db, func() {
		l.OnWaitForSignal(context.Background(), testMetadata(t), queue.Item{}, gen, statev1.Pause{})
	})

	require.Equal(t, "my-signal", attrs[meta.Attrs.StepSignalName.Key()])
}

// TestListenerSetsInvokeAttrsOnPauseStartedSpan mirrors the WaitForEvent
// test above for OnInvokeFunction's step.invoke.function.id and
// step.invoke.trigger.event.id attributes — the latter read from the
// opcode's own invocation-event payload (opts.Payload.ID), the same field
// pkg/tracing/util.go's generatorAttrs nil-guards (see the OnInvokeFunction
// nil-pointer fix this test would otherwise have caught).
func TestListenerSetsInvokeAttrsOnPauseStartedSpan(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	triggerEventID := ulid.MustNew(ulid.Now(), rand.Reader)
	gen := statev1.GeneratorOpcode{
		ID:   "step-1",
		Name: "invoke-step",
		Op:   enums.OpcodeInvokeFunction,
		Opts: &statev1.InvokeFunctionOpts{
			FunctionID: "my-app-my-fn",
			Timeout:    "1h",
			Payload:    &event.Event{ID: triggerEventID.String()},
		},
	}

	attrs := singleSpanAttrs(t, db, func() {
		l.OnInvokeFunction(context.Background(), testMetadata(t), queue.Item{}, gen, event.Event{})
	})

	require.Equal(t, "my-app-my-fn", attrs[meta.Attrs.StepInvokeFunctionID.Key()])
	require.Equal(t, triggerEventID.String(), attrs[meta.Attrs.StepInvokeTriggerEventID.Key()])
}

// TestListenerOnInvokeFunctionDoesNotPanicWithNilPayload proves the
// pkg/tracing/util.go nil-pointer fix: InvokeFunctionOpts.Validate()
// explicitly treats a nil Payload as valid, so generatorAttrs must not
// crash dereferencing opts.Payload.ID when it's nil — it just gets no
// step.invoke.trigger.event.id attribute.
func TestListenerOnInvokeFunctionDoesNotPanicWithNilPayload(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	gen := statev1.GeneratorOpcode{
		ID:   "step-1",
		Name: "invoke-step",
		Op:   enums.OpcodeInvokeFunction,
		Opts: &statev1.InvokeFunctionOpts{FunctionID: "my-app-my-fn", Timeout: "1h"},
	}

	attrs := singleSpanAttrs(t, db, func() {
		l.OnInvokeFunction(context.Background(), testMetadata(t), queue.Item{}, gen, event.Event{})
	})

	require.Equal(t, "my-app-my-fn", attrs[meta.Attrs.StepInvokeFunctionID.Key()])
	_, hasTriggerEventID := attrs[meta.Attrs.StepInvokeTriggerEventID.Key()]
	require.False(t, hasTriggerEventID)
}

// TestListenerSetsWaitForEventAttrsOnResumedSpan proves resumeAttrs — shared
// by the three *Resumed hooks — pulls the same opcode-specific fields
// directly off statev1.Pause/execution.ResumeRequest (no GeneratorOpcode is
// available at resume time), including step.wait_for_event.matched_id
// (r.EventID) — a field the flat model must set explicitly at resume time
// since, unlike the rollup model, there is no fragment merge to inherit it
// from the pause-started span.
func TestListenerSetsWaitForEventAttrsOnResumedSpan(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	waitForEvent := enums.OpcodeWaitForEvent.String()
	evtName := "app/some.event"
	ifExpr := "true"
	createdAt := time.Now().Add(-time.Minute).Round(0)
	expires := statev1.Time(time.Now().Add(time.Hour))
	pause := statev1.Pause{
		Outgoing:   "step-1",
		Opcode:     &waitForEvent,
		Event:      &evtName,
		Expression: &ifExpr,
		Expires:    expires,
		CreatedAt:  createdAt,
	}
	matchedID := ulid.MustNew(ulid.Now(), rand.Reader)
	resumedAt := time.Now().Round(0)

	attrs := singleSpanAttrs(t, db, func() {
		l.OnWaitForEventResumed(context.Background(), testMetadata(t), pause, execution.ResumeRequest{EventID: &matchedID}, resumedAt)
	})

	require.Equal(t, "step-1", attrs[meta.Attrs.StepID.Key()], "step.id should be set from pause.Outgoing")
	require.Equal(t, enums.OpcodeWaitForEvent.String(), attrs[meta.Attrs.StepOp.Key()])
	require.Equal(t, evtName, attrs[meta.Attrs.StepWaitForEventName.Key()])
	require.Equal(t, "true", attrs[meta.Attrs.StepWaitForEventIf.Key()])
	require.Equal(t, matchedID.String(), attrs[meta.Attrs.StepWaitForEventMatchedID.Key()])
	require.Equal(t, false, attrs[meta.Attrs.StepWaitExpired.Key()])
	_, hasExpiry := attrs[meta.Attrs.StepWaitExpiry.Key()]
	require.True(t, hasExpiry, "step.wait.expiry should be set from pause.Expires")

	// The resumed span's timing attributes: QueuedAt/ScheduledAt fall back
	// to pause.CreatedAt (the only timestamp these hooks otherwise have —
	// see createPauseSpan's own doc comment), and EndedAt is the caller's
	// own resume-time timestamp threaded through the SyncLifecycleListener
	// interface, not a fresh time.Now() read inside this package.
	requireTimeAttrEqual(t, createdAt, attrs[meta.Attrs.QueuedAt.Key()])
	requireTimeAttrEqual(t, createdAt, attrs[meta.Attrs.ScheduledAt.Key()])
	requireTimeAttrEqual(t, resumedAt, attrs[meta.Attrs.EndedAt.Key()])
}

// requireTimeAttrEqual compares a meta.TimeAttr-backed attribute (see
// pkg/tracing/meta/serializers.go's TimeAttr — serialized as an
// attribute.String of the Unix-milliseconds epoch, so it decodes here as a
// plain digit string, not RFC3339) against an expected time.Time to
// millisecond precision.
func requireTimeAttrEqual(t *testing.T, want time.Time, got any) {
	t.Helper()
	gotStr, ok := got.(string)
	require.True(t, ok, "expected a string-encoded timestamp, got %T (%v)", got, got)
	millis, err := strconv.ParseInt(gotStr, 10, 64)
	require.NoError(t, err, "parsing attribute timestamp %q", gotStr)
	require.WithinDuration(t, want, time.UnixMilli(millis), time.Millisecond)
}

// TestListenerSetsInvokeAttrsOnResumedSpan proves resumeAttrs' invoke branch:
// StepInvokeFunctionID/StepInvokeTriggerEventID from the pause itself (the
// latter despite Pause.TriggeringEventID's generic doc comment — see
// resumeAttrs' own comment on why it holds the invocation event's ID for an
// invoke pause specifically), and StepInvokeFinishEventID/StepInvokeRunID
// from the resume request.
func TestListenerSetsInvokeAttrsOnResumedSpan(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	invoke := enums.OpcodeInvokeFunction.String()
	fnID := "my-app-my-fn"
	triggerEventID := ulid.MustNew(ulid.Now(), rand.Reader)
	triggerEventIDStr := triggerEventID.String()
	pause := statev1.Pause{
		Outgoing:          "step-1",
		Opcode:            &invoke,
		InvokeTargetFnID:  &fnID,
		TriggeringEventID: &triggerEventIDStr,
	}
	finishEventID := ulid.MustNew(ulid.Now(), rand.Reader)
	invokedRunID := ulid.MustNew(ulid.Now(), rand.Reader)

	attrs := singleSpanAttrs(t, db, func() {
		l.OnInvokeFunctionResumed(context.Background(), testMetadata(t), pause, execution.ResumeRequest{
			EventID: &finishEventID,
			RunID:   &invokedRunID,
		}, time.Now())
	})

	require.Equal(t, "step-1", attrs[meta.Attrs.StepID.Key()])
	require.Equal(t, enums.OpcodeInvokeFunction.String(), attrs[meta.Attrs.StepOp.Key()])
	require.Equal(t, fnID, attrs[meta.Attrs.StepInvokeFunctionID.Key()])
	require.Equal(t, triggerEventID.String(), attrs[meta.Attrs.StepInvokeTriggerEventID.Key()])
	require.Equal(t, finishEventID.String(), attrs[meta.Attrs.StepInvokeFinishEventID.Key()])
	require.Equal(t, invokedRunID.String(), attrs[meta.Attrs.StepInvokeRunID.Key()])
}

// TestListenerSetsSignalAttrsOnResumedSpan mirrors the WaitForEvent/Invoke
// tests above for OnWaitForSignalResumed's step.signal.name attribute.
func TestListenerSetsSignalAttrsOnResumedSpan(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	signal := enums.OpcodeWaitForSignal.String()
	signalName := "my-signal"
	pause := statev1.Pause{Outgoing: "step-1", Opcode: &signal, SignalID: &signalName}

	attrs := singleSpanAttrs(t, db, func() {
		l.OnWaitForSignalResumed(context.Background(), testMetadata(t), pause, execution.ResumeRequest{}, time.Now())
	})

	require.Equal(t, "step-1", attrs[meta.Attrs.StepID.Key()])
	require.Equal(t, enums.OpcodeWaitForSignal.String(), attrs[meta.Attrs.StepOp.Key()])
	require.Equal(t, signalName, attrs[meta.Attrs.StepSignalName.Key()])
}

// TestListenerDoesNotLeakInvokeInternalFieldsIntoWaitForEventAttrsOnResume
// proves resumeAttrs' opcode gating: an invoke pause's Event/Expression
// fields hold its own internal function-finished-matching plumbing (see
// handleGeneratorInvokeFunction), not a user-facing wait condition, so they
// must never surface as step.wait_for_event.name/if on an invoke resume.
func TestListenerDoesNotLeakInvokeInternalFieldsIntoWaitForEventAttrsOnResume(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	invoke := enums.OpcodeInvokeFunction.String()
	internalEventName := "inngest/function.finished"
	internalExpr := "async.data.correlation_id == \"...\""
	pause := statev1.Pause{
		Outgoing:   "step-1",
		Opcode:     &invoke,
		Event:      &internalEventName,
		Expression: &internalExpr,
	}

	attrs := singleSpanAttrs(t, db, func() {
		l.OnInvokeFunctionResumed(context.Background(), testMetadata(t), pause, execution.ResumeRequest{}, time.Now())
	})

	_, hasName := attrs[meta.Attrs.StepWaitForEventName.Key()]
	_, hasIf := attrs[meta.Attrs.StepWaitForEventIf.Key()]
	require.False(t, hasName, "an invoke pause's internal Event field must not leak as step.wait_for_event.name")
	require.False(t, hasIf, "an invoke pause's internal Expression field must not leak as step.wait_for_event.if")
}

// TestListenerRunSpanUsesRunIDTimestampNotItemEnqueuedAt proves
// OnFunctionFinished's "executor.run" span sets QueuedAt from the run's own
// canonical queued_at (md.ID.RunID's embedded timestamp — the same value
// runCommonFields/OnFunctionScheduled/OnFunctionStarted all treat as
// queued_at), not item.EnqueuedAt: item here is whichever queue.Item
// happened to trigger this specific finish call, which for a multi-step
// function is a later step's item, not the run's original enqueue, and can
// disagree with the run's actual queued_at by any amount. ScheduledAt is
// omitted entirely for the same reason — item.At is that same later step's
// time, not the run's real scheduled_at, which isn't plumbed through the
// state store to this hook (see OnFunctionFinished's own TODO comment).
func TestListenerRunSpanUsesRunIDTimestampNotItemEnqueuedAt(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	md := testMetadata(t)
	runQueuedAt := ulid.Time(md.ID.RunID.Time())
	// A later step's queue.Item — its own EnqueuedAt/At land long after the
	// run was originally queued, simulating a multi-step function's final
	// finish call.
	laterItemTime := runQueuedAt.Add(time.Hour).Round(0)
	item := queue.Item{EnqueuedAt: laterItemTime, At: laterItemTime}

	l.OnFunctionFinished(context.Background(), md, item, nil, statev1.DriverResponse{}, time.Now())

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.run_trace_spans WHERE name = 'executor.run';")
		_ = row.Scan(&count)
		return count == 1
	}, 2*time.Second, 20*time.Millisecond, "executor.run span should land after a batch flush")

	rows := selectRows(t, db, "SELECT attributes FROM inngest.run_trace_spans WHERE name = 'executor.run';")
	require.Len(t, rows, 1)
	attrs, ok := rows[0]["attributes"].(map[string]any)
	require.True(t, ok, "attributes column should decode to a map, got %T", rows[0]["attributes"])

	requireTimeAttrEqual(t, runQueuedAt, attrs[meta.Attrs.QueuedAt.Key()])
	_, hasScheduledAt := attrs[meta.Attrs.ScheduledAt.Key()]
	require.False(t, hasScheduledAt, "scheduled_at should be omitted, not set from the wrong item's time")
}

// TestListenerOnFunctionScheduledOmitsStartedAndEndedAt proves the
// executor.run.queued span never carries StartedAt/EndedAt attributes: the
// run has only been queued at this point, not started or ended, even
// though the physical span's own start_time/end_time columns are both
// pinned to queuedAt (needed for GetSpansByRunID's tree-building — see
// OnFunctionScheduled's own comment). Without the explicit nil-attr
// suppression, tracingv3.CreateSpan would auto-populate both from
// CreateSpanOptions.StartTime/EndTime.
func TestListenerOnFunctionScheduledOmitsStartedAndEndedAt(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	md := testMetadata(t)
	l.OnFunctionScheduled(context.Background(), md, queue.Item{}, nil)

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.run_trace_spans WHERE name = 'executor.run.queued';")
		_ = row.Scan(&count)
		return count == 1
	}, 2*time.Second, 20*time.Millisecond, "executor.run.queued span should land after a batch flush")

	rows := selectRows(t, db, "SELECT attributes, start_time, end_time FROM inngest.run_trace_spans WHERE name = 'executor.run.queued';")
	require.Len(t, rows, 1)
	attrs, ok := rows[0]["attributes"].(map[string]any)
	require.True(t, ok, "attributes column should decode to a map, got %T", rows[0]["attributes"])

	_, hasStartedAt := attrs[meta.Attrs.StartedAt.Key()]
	_, hasEndedAt := attrs[meta.Attrs.EndedAt.Key()]
	require.False(t, hasStartedAt, "started_at should be omitted — the run has only been queued, not started")
	require.False(t, hasEndedAt, "ended_at should be omitted — the run has only been queued, not ended")

	// The physical span row itself must still have real, non-NULL
	// start_time/end_time (both pinned to queuedAt) — only the semantic
	// StartedAt/EndedAt *attributes* are suppressed.
	require.NotNil(t, rows[0]["start_time"])
	require.NotNil(t, rows[0]["end_time"])
}

func TestListenerOnDeferAddWritesDeferSpan(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	md := testMetadata(t)
	d := sv2.Defer{FnSlug: "deferred-fn", HashedID: "hash-1", ScheduleStatus: enums.DeferStatusAfterRun}
	l.OnDeferAdd(context.Background(), md, d, "userland-1", time.Now())

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.run_trace_spans WHERE name = ?;", meta.SpanNameDefer)
		_ = row.Scan(&count)
		return count == 1
	}, 2*time.Second, 20*time.Millisecond, "executor.defer span should land after a batch flush")

	rows := selectRows(t, db, "SELECT run_id, span_id, attributes FROM inngest.run_trace_spans WHERE name = ?;", meta.SpanNameDefer)
	require.Len(t, rows, 1)
	require.Equal(t, md.ID.RunID.String(), rows[0]["run_id"])

	attrs, ok := rows[0]["attributes"].(map[string]any)
	require.True(t, ok, "attributes column should decode to a map, got %T", rows[0]["attributes"])
	require.Equal(t, d.HashedID, attrs[meta.Attrs.DeferHashedID.Key()])
	require.Equal(t, "userland-1", attrs[meta.Attrs.DeferUserlandID.Key()])
	require.Equal(t, d.FnSlug, attrs[meta.Attrs.DeferFnSlug.Key()])
	require.Equal(t, enums.DeferStatusAfterRun.String(), attrs[meta.Attrs.DeferStatus.Key()])

	// The span_id must be the real, deterministic identity — the same one
	// tracing.DeferSpanRef computes for the real system's own span — so
	// later writes to the same defer (abort, schedule-link) land on the
	// same row group.
	wantSpanID := tracing.DeferSpanRef(md.ID.RunID, d.HashedID).DynamicSpanID
	require.Equal(t, wantSpanID, rows[0]["span_id"])
}

// TestListenerOnDeferAbortReusesSameSpanID proves OnDeferAbort's write
// lands on the exact same (run_id, span_id) as OnDeferAdd's — the
// precondition for duckdbquery's read-side collapse to treat them as one
// logical defer rather than two unrelated spans.
func TestListenerOnDeferAbortReusesSameSpanID(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	md := testMetadata(t)
	d := sv2.Defer{FnSlug: "deferred-fn", HashedID: "hash-2", ScheduleStatus: enums.DeferStatusAfterRun}
	l.OnDeferAdd(context.Background(), md, d, "userland-2", time.Now())
	l.OnDeferAbort(context.Background(), md, d.HashedID, time.Now())

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.run_trace_spans WHERE name = ?;", meta.SpanNameDefer)
		_ = row.Scan(&count)
		return count == 2
	}, 2*time.Second, 20*time.Millisecond, "both the add and abort span rows should land after a batch flush")

	rows := selectRows(t, db, "SELECT span_id, attributes FROM inngest.run_trace_spans WHERE name = ?;", meta.SpanNameDefer)
	require.Len(t, rows, 2)
	require.Equal(t, rows[0]["span_id"], rows[1]["span_id"], "add and abort rows must share the same deterministic span_id")

	var sawAborted bool
	for _, row := range rows {
		attrs, ok := row["attributes"].(map[string]any)
		require.True(t, ok)
		if attrs[meta.Attrs.DeferStatus.Key()] == enums.DeferStatusAborted.String() {
			sawAborted = true
			require.Equal(t, d.HashedID, attrs[meta.Attrs.DeferHashedID.Key()])
			// fn_slug/userland_id aren't known at abort time — see
			// OnDeferAbort's own doc comment.
			_, hasFnSlug := attrs[meta.Attrs.DeferFnSlug.Key()]
			require.False(t, hasFnSlug)
		}
	}
	require.True(t, sawAborted, "one of the two rows should carry status=Aborted")
}

// TestListenerOnExtendedTraceSpanWritesSpan proves OnExtendedTraceSpan --
// unlike every other hook in this file, invoked from
// pkg/api/apiv1/traces.go's commitSpan rather than the executor/runner --
// still lands a row in inngest.run_trace_spans via the same
// createSpan/spanExporter path, under the package's own
// tracingv3.SpanNameExtendedTrace name, carrying the caller-supplied
// tenant/run identity and attributes.
func TestListenerOnExtendedTraceSpanWritesSpan(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	accountID, envID, appID, functionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	runID := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)
	md := sv2.Metadata{ID: sv2.ID{
		RunID:      runID,
		FunctionID: functionID,
		Tenant:     sv2.Tenant{AccountID: accountID, EnvID: envID, AppID: appID},
	}}
	sv2.InitConfig(&md.Config)

	span := execution.ExtendedTraceSpan{
		AccountID:  accountID,
		EnvID:      envID,
		AppID:      appID,
		FunctionID: functionID,
		RunID:      runID,
		Parent:     tracing.RunSpanRefFromMetadata(&md),
		SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		Name:       "my.userland.operation",
		StartTime:  time.Now().Add(-time.Second),
		EndTime:    time.Now(),
		Attributes: []attribute.KeyValue{attribute.String("custom.attr", "hello")},
	}
	l.OnExtendedTraceSpan(context.Background(), span)

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.run_trace_spans WHERE name = ?;", tracingv3.SpanNameExtendedTrace)
		_ = row.Scan(&count)
		return count == 1
	}, 2*time.Second, 20*time.Millisecond, "extended-trace span should land after a batch flush")

	rows := selectRows(t, db, "SELECT run_id, span_id, attributes FROM inngest.run_trace_spans WHERE name = ?;", tracingv3.SpanNameExtendedTrace)
	require.Len(t, rows, 1)
	require.Equal(t, runID.String(), rows[0]["run_id"])
	require.Equal(t, span.SpanID.String(), rows[0]["span_id"])

	attrs, ok := rows[0]["attributes"].(map[string]any)
	require.True(t, ok, "attributes column should decode to a map, got %T", rows[0]["attributes"])
	require.Equal(t, "hello", attrs["custom.attr"])
}

// deferScheduleEventJSON builds the raw JSON of an inngest/deferred.schedule
// event, matching what pkg/execution/executor/finalize.go actually
// publishes (minus fields this package's OnFunctionScheduled doesn't read).
func deferScheduleEventJSON(t *testing.T, m event.DeferredScheduleMetadata) json.RawMessage {
	t.Helper()
	evt := event.Event{
		ID:   ulid.MustNew(ulid.Now(), rand.Reader).String(),
		Name: consts.FnDeferScheduleName,
		Data: map[string]any{
			consts.InngestEventDataPrefix: m,
		},
	}
	raw, err := json.Marshal(evt)
	require.NoError(t, err)
	return raw
}

// TestListenerOnFunctionScheduledLinksDeferredChild proves a run scheduled
// via defer() gets is_deferred=true on its own inngest.runs row, its own
// DeferParentRunIDs/DeferParentFnSlug attrs, AND updates the PARENT's
// executor.defer span (a different run_id entirely) with DeferChildRunID —
// the three things GetTraceRunFilter.IsDeferred and GetRunDefers/
// GetRunDeferredFrom need to work on DuckDB.
func TestListenerOnFunctionScheduledLinksDeferredChild(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	childMD := testMetadata(t)
	parentRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	parentFnID := uuid.New()
	parentAppID := uuid.New()
	hashedID := "hash-3"

	deferMeta := event.DeferredScheduleMetadata{
		FnSlug:          "deferred-fn",
		ParentAppID:     parentAppID,
		ParentDeferSpan: tracing.DeferSpanRef(parentRunID, hashedID),
		ParentFnID:      parentFnID,
		ParentFnSlug:    "parent-fn",
		ParentRunID:     parentRunID,
		HashedID:        hashedID,
	}
	raw := deferScheduleEventJSON(t, deferMeta)

	l.OnFunctionScheduled(context.Background(), childMD, queue.Item{}, []json.RawMessage{raw})

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(),
			"SELECT count(*) FROM inngest.run_trace_spans WHERE name = ? AND run_id = ?;",
			meta.SpanNameDefer, parentRunID.String())
		_ = row.Scan(&count)
		return count == 1
	}, 2*time.Second, 20*time.Millisecond, "parent defer span update should land after a batch flush")

	// 1. is_deferred on the child's own run row.
	runRows := selectRows(t, db, "SELECT is_deferred FROM inngest.runs WHERE run_id = ?;", childMD.ID.RunID.String())
	require.Len(t, runRows, 1)
	require.Equal(t, true, runRows[0]["is_deferred"])

	// 2. DeferParentRunIDs/DeferParentFnSlug on the child's own queued span.
	childSpanRows := selectRows(t, db,
		"SELECT attributes FROM inngest.run_trace_spans WHERE name = 'executor.run.queued' AND run_id = ?;",
		childMD.ID.RunID.String())
	require.Len(t, childSpanRows, 1)
	childAttrs, ok := childSpanRows[0]["attributes"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "parent-fn", childAttrs[meta.Attrs.DeferParentFnSlug.Key()])
	parentIDs, ok := childAttrs[meta.Attrs.DeferParentRunIDs.Key()].([]any)
	require.True(t, ok)
	require.Equal(t, []any{parentRunID.String()}, parentIDs)

	// 3. DeferChildRunID stamped onto the PARENT's own defer span row.
	parentSpanRows := selectRows(t, db,
		"SELECT span_id, attributes FROM inngest.run_trace_spans WHERE name = ? AND run_id = ?;",
		meta.SpanNameDefer, parentRunID.String())
	require.Len(t, parentSpanRows, 1)
	parentAttrs, ok := parentSpanRows[0]["attributes"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, hashedID, parentAttrs[meta.Attrs.DeferHashedID.Key()])
	require.Equal(t, "deferred-fn", parentAttrs[meta.Attrs.DeferFnSlug.Key()])
	require.Equal(t, childMD.ID.RunID.String(), parentAttrs[meta.Attrs.DeferChildRunID.Key()])
	require.Equal(t, tracing.DeferSpanRef(parentRunID, hashedID).DynamicSpanID, parentSpanRows[0]["span_id"])
}

// TestListenerOnFunctionScheduledSkipsParentLinkWithoutHashedID proves a
// deferred.schedule event missing HashedID (an older/pre-existing event —
// see DeferredScheduleMetadata's own doc comment) is skipped for the
// parent-span update, rather than writing a wrongly-seeded orphan row, while
// the child's own DeferParentRunIDs/DeferParentFnSlug attrs (which don't
// depend on HashedID) still get set.
func TestListenerOnFunctionScheduledSkipsParentLinkWithoutHashedID(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	childMD := testMetadata(t)
	parentRunID := ulid.MustNew(ulid.Now(), rand.Reader)
	deferMeta := event.DeferredScheduleMetadata{
		FnSlug:      "deferred-fn",
		ParentAppID: uuid.New(),
		// A real ParentDeferSpan is required to pass Validate(), even
		// though HashedID (the field this test omits) is what
		// updateParentDeferSpans actually needs.
		ParentDeferSpan: tracing.DeferSpanRef(parentRunID, "unused-in-this-test"),
		ParentFnID:      uuid.New(),
		ParentFnSlug:    "parent-fn",
		ParentRunID:     parentRunID,
		// HashedID intentionally omitted.
	}
	raw := deferScheduleEventJSON(t, deferMeta)

	l.OnFunctionScheduled(context.Background(), childMD, queue.Item{}, []json.RawMessage{raw})

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.run_trace_spans WHERE name = 'executor.run.queued';")
		_ = row.Scan(&count)
		return count == 1
	}, 2*time.Second, 20*time.Millisecond, "child's own queued span should still land")

	var count int
	row := db.QueryRowContext(context.Background(),
		"SELECT count(*) FROM inngest.run_trace_spans WHERE name = ? AND run_id = ?;",
		meta.SpanNameDefer, parentRunID.String())
	require.NoError(t, row.Scan(&count))
	require.Equal(t, 0, count, "no parent defer span row should be written without a hashed ID")
}

func TestListenerOnFunctionScheduledOmitsIsDeferredForNormalRuns(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	l := NewListener(db, func(o *setupOpts) { o.batchInterval = 20 * time.Millisecond })

	md := testMetadata(t)
	l.OnFunctionScheduled(context.Background(), md, queue.Item{}, nil)

	require.Eventually(t, func() bool {
		var count int
		row := db.QueryRowContext(context.Background(), "SELECT count(*) FROM inngest.runs;")
		_ = row.Scan(&count)
		return count == 1
	}, 2*time.Second, 20*time.Millisecond, "run row should land after a batch flush")

	rows := selectRows(t, db, "SELECT is_deferred FROM inngest.runs WHERE run_id = ?;", md.ID.RunID.String())
	require.Len(t, rows, 1)
	require.Nil(t, rows[0]["is_deferred"], "a normal (non-deferred) run must leave is_deferred NULL")
}
