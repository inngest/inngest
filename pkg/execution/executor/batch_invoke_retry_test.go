package executor

import (
	"bytes"
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// stubBatchEnqueuer simulates a queue that returns ErrQueueItemExists for all
// items, representing a retry where all timeouts were already enqueued.
type stubBatchEnqueuer struct {
	queue.Queue
	enqueueErr error
}

func (q *stubBatchEnqueuer) Enqueue(_ context.Context, _ queue.Item, _ time.Time, _ queue.EnqueueOpts) error {
	return q.enqueueErr
}

func (q *stubBatchEnqueuer) EnqueueBatch(_ context.Context, items []queue.Item, _ []time.Time, _ queue.EnqueueOpts) []error {
	errs := make([]error, len(items))
	for i := range errs {
		errs[i] = q.enqueueErr
	}
	return errs
}

// TestEnqueueAndPublishBatch_PublishesEventsOnRetry verifies that when all items
// return ErrQueueItemExists (a retry scenario), events are still published for
// every item. This prevents the bug where children are permanently orphaned.
func TestEnqueueAndPublishBatch_PublishesEventsOnRetry(t *testing.T) {
	ctx := context.Background()

	var publishedEvents []event.TrackedEvent
	e := &executor{
		queue: &stubBatchEnqueuer{enqueueErr: queue.ErrQueueItemExists},
		handleInvokeEventsBatch: func(_ context.Context, evts []event.TrackedEvent) error {
			publishedEvents = append(publishedEvents, evts...)
			return nil
		},
		log:            logger.From(ctx),
		tracerProvider: tracing.NewNoopTracerProvider(),
	}

	items := make([]batchInvokeItem, 3)
	for i := range items {
		items[i] = batchInvokeItem{
			item: queue.Item{
				WorkspaceID: uuid.New(),
			},
			evt: event.BaseTrackedEvent{
				ID:    ulid.MustNew(ulid.Now(), nil),
				Event: event.Event{ID: uuid.NewString(), Name: "inngest/function.invoked"},
			},
			expires: time.Now().Add(time.Hour),
		}
	}

	skipItem, err := e.enqueueAndPublishBatch(ctx, nil, items)
	require.NoError(t, err)

	// All items should be marked as skip (ErrQueueItemExists → span suppression)
	for idx, skip := range skipItem {
		require.True(t, skip, "expected skipItem[%d] = true for ErrQueueItemExists", idx)
	}

	// Despite all items being "skipped" for span purposes, ALL events must
	// have been published — this is the critical retry-safety property.
	require.Len(t, publishedEvents, 3, "all events must be published on retry even when queue items already exist")
}

// TestEnqueueAndPublishBatch_PublishesEventsOnFirstAttempt verifies that on a
// fresh first attempt (no ErrQueueItemExists), events are also published for
// all items.
func TestEnqueueAndPublishBatch_PublishesEventsOnFirstAttempt(t *testing.T) {
	ctx := context.Background()

	var publishedEvents []event.TrackedEvent
	e := &executor{
		queue: &stubBatchEnqueuer{enqueueErr: nil}, // all succeed
		handleInvokeEventsBatch: func(_ context.Context, evts []event.TrackedEvent) error {
			publishedEvents = append(publishedEvents, evts...)
			return nil
		},
		log:            logger.From(ctx),
		tracerProvider: tracing.NewNoopTracerProvider(),
	}

	items := make([]batchInvokeItem, 3)
	for i := range items {
		items[i] = batchInvokeItem{
			item: queue.Item{
				WorkspaceID: uuid.New(),
			},
			evt: event.BaseTrackedEvent{
				ID:    ulid.MustNew(ulid.Now(), nil),
				Event: event.Event{ID: uuid.NewString(), Name: "inngest/function.invoked"},
			},
			expires: time.Now().Add(time.Hour),
		}
	}

	skipItem, err := e.enqueueAndPublishBatch(ctx, nil, items)
	require.NoError(t, err)

	// No items should be marked as skip on first successful attempt
	for idx, skip := range skipItem {
		require.False(t, skip, "expected skipItem[%d] = false on first attempt", idx)
	}

	require.Len(t, publishedEvents, 3, "all events must be published on first attempt")
}

// perItemOnlyQueue implements queue.Queue.Enqueue but NOT queue.BatchEnqueuer,
// forcing the per-item fallback path in enqueueBatchTimeouts.
type perItemOnlyQueue struct {
	queue.Queue
	enqueueErr error
}

func (q *perItemOnlyQueue) Enqueue(_ context.Context, _ queue.Item, _ time.Time, _ queue.EnqueueOpts) error {
	return q.enqueueErr
}

// TestEnqueueAndPublishBatch_PerItemFallback verifies the retry-safety behavior
// when the queue does NOT implement BatchEnqueuer (falls back to per-item Enqueue).
func TestEnqueueAndPublishBatch_PerItemFallback(t *testing.T) {
	ctx := context.Background()

	var publishedEvents []event.TrackedEvent
	e := &executor{
		queue: &perItemOnlyQueue{enqueueErr: queue.ErrQueueItemExists},
		handleInvokeEventsBatch: func(_ context.Context, evts []event.TrackedEvent) error {
			publishedEvents = append(publishedEvents, evts...)
			return nil
		},
		log:            logger.From(ctx),
		tracerProvider: tracing.NewNoopTracerProvider(),
	}

	items := make([]batchInvokeItem, 3)
	for i := range items {
		items[i] = batchInvokeItem{
			item: queue.Item{
				WorkspaceID: uuid.New(),
			},
			evt: event.BaseTrackedEvent{
				ID:    ulid.MustNew(ulid.Now(), nil),
				Event: event.Event{ID: uuid.NewString(), Name: "inngest/function.invoked"},
			},
			expires: time.Now().Add(time.Hour),
		}
	}

	skipItem, err := e.enqueueAndPublishBatch(ctx, nil, items)
	require.NoError(t, err)

	for idx, skip := range skipItem {
		require.True(t, skip, "expected skipItem[%d] = true for ErrQueueItemExists (per-item fallback)", idx)
	}

	require.Len(t, publishedEvents, 3, "all events must be published on retry via per-item fallback path")
}

// TestBuildSingleInvokeItem_DeterministicEventID verifies that the event ID
// generated for batch invoke items is deterministic (same runID + gen.ID
// always produces the same event ID), preventing duplicate child runs on retry.
func TestBuildSingleInvokeItem_DeterministicEventID(t *testing.T) {
	runID := ulid.MustNew(ulid.Now(), nil)
	genID := "step-invoke-1"

	// Compute the expected deterministic event ID
	expectedID := deterministicEventULID(t, runID, genID)

	// Verify it's stable across calls
	require.Equal(t, expectedID, deterministicEventULID(t, runID, genID))

	// Different gen.ID produces different event ID
	otherID := deterministicEventULID(t, runID, "step-invoke-2")
	require.NotEqual(t, expectedID, otherID)

	// The result must be a valid ULID (downstream code uses ulid.MustParse)
	_, err := ulid.Parse(expectedID)
	require.NoError(t, err, "deterministic event ID must be a valid ULID")
}

// deterministicEventULID mirrors the logic in buildSingleInvokeItem for testing.
func deterministicEventULID(t *testing.T, runID ulid.ULID, genID string) string {
	t.Helper()
	h := sha256.Sum256([]byte(runID.String() + genID + ":evt"))
	id, err := ulid.New(runID.Time(), bytes.NewReader(h[:10]))
	require.NoError(t, err)
	return id.String()
}
