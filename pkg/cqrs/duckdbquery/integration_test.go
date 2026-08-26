package duckdbquery

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/dualwrite"
	"github.com/inngest/inngest/pkg/execution/queue"
	statev1 "github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// TestDualWriteThenDuckDBQueryRoundTrip drives the real dual-write listener
// (the same one pkg/devserver wires up in production) through a full
// scheduled->started->finished run and one event, then reads them back
// through duckdbquery.Manager — proving the two ends of this plan's spec
// (docs/plans/007-duckdb-gql-resolvers.md) actually agree on the schema.
func TestDualWriteThenDuckDBQueryRoundTrip(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()

	ctx := context.Background()
	queuedAt := time.Now().UTC()
	// Explicitly well-separated (10ms apart) rather than relying on
	// time.Now() calls made microseconds apart in a fast synchronous test:
	// inngest.runs' TIMESTAMP_MS columns truncate to millisecond precision,
	// so queued_at/started_at/ended_at landing in the same millisecond
	// bucket would make GetTraceRun's "latest row" ordering
	// (COALESCE(ended_at, started_at, queued_at) DESC) genuinely tied —
	// exactly the scenario production's real scheduling/execution latency
	// never produces, but a synchronous unit test can.
	startedAt := queuedAt.Add(10 * time.Millisecond)
	endedAt := queuedAt.Add(20 * time.Millisecond)

	evt := event.NewBaseTrackedEvent(event.Event{Name: "e2e/test"}, nil)

	md := sv2.Metadata{
		ID: sv2.ID{
			RunID:      ulid.MustNew(ulid.Timestamp(queuedAt), rand.Reader),
			FunctionID: uuid.New(),
			Tenant: sv2.Tenant{
				AccountID: uuid.New(),
				EnvID:     uuid.New(),
			},
		},
		Config: sv2.Config{
			StartedAt: startedAt,
			EventIDs:  []ulid.ULID{evt.GetInternalID()},
		},
	}

	l := dualwrite.NewListener(db)
	l.OnFunctionScheduled(ctx, md, queue.Item{}, nil)
	l.OnFunctionStarted(ctx, md, queue.Item{}, nil)
	l.OnFunctionFinished(ctx, md, queue.Item{}, nil, statev1.DriverResponse{}, endedAt)

	l.OnEventReceived(ctx, evt)

	m := Wrap(nil, db).(*Manager)

	require.Eventually(t, func() bool {
		run, err := m.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: md.ID.RunID})
		return err == nil && run.Status.String() == "Completed"
	}, 5*time.Second, 50*time.Millisecond, "run should be queryable as Completed after dual-write flushes")

	run, err := m.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: md.ID.RunID})
	require.NoError(t, err)
	require.Equal(t, []string{evt.GetInternalID().String()}, run.TriggerIDs,
		"TriggerIDs should round-trip through the event_ids column")

	require.Eventually(t, func() bool {
		events, err := m.GetEventsByInternalIDs(ctx, []ulid.ULID{evt.GetInternalID()})
		return err == nil && len(events) == 1
	}, 5*time.Second, 50*time.Millisecond, "event should be queryable after dual-write flushes")

	runs, err := m.GetTraceRuns(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: md.ID.Tenant.AccountID, WorkspaceID: md.ID.Tenant.EnvID,
			TimeField: enums.TraceRunTimeQueuedAt,
			From:      time.Now().Add(-time.Hour), Until: time.Now().Add(time.Hour),
		},
		Items: 40,
	})
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, md.ID.RunID.String(), runs[0].RunID)

	count, err := m.GetTraceRunsCount(ctx, cqrs.GetTraceRunOpt{
		Filter: cqrs.GetTraceRunFilter{
			AccountID: md.ID.Tenant.AccountID, WorkspaceID: md.ID.Tenant.EnvID,
			TimeField: enums.TraceRunTimeQueuedAt,
			From:      time.Now().Add(-time.Hour), Until: time.Now().Add(time.Hour),
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, count, "3 lifecycle rows for the one run must count as 1")
}
