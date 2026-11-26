package queue

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// the run func gets the item's due-to-start latency, so that lifecycle
// listeners can report per-function queue delays.
func TestProcessItemExposesStartLatency(t *testing.T) {
	ctx := context.Background()
	// queue timestamps are in milliseconds.
	clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Millisecond))

	registry, err := NewSingleShardRegistry(&mockShardForIterator{name: "test"})
	require.NoError(t, err)

	q, err := New(ctx, "test", registry, WithClock(clock))
	require.NoError(t, err)

	now := clock.Now()
	itemLeaseID := ulid.Make()
	item := QueueItem{
		ID:         "start-latency-item",
		FunctionID: uuid.New(),
		LeaseID:    &itemLeaseID,
		// Enqueued 5s ago to run 2s later, so the item has waited 3s since
		// it was due.  The scheduled 2s is intentional and not a delay.
		EnqueuedAt: now.Add(-5 * time.Second).UnixMilli(),
		AtMS:       now.Add(-3 * time.Second).UnixMilli(),
		WallTimeMS: now.Add(-3 * time.Second).UnixMilli(),
		Data: Item{
			Kind: KindEdge,
			Identifier: state.Identifier{
				AccountID:   uuid.New(),
				WorkspaceID: uuid.New(),
				WorkflowID:  uuid.New(),
				RunID:       ulid.Make(),
			},
		},
	}

	var (
		latency time.Duration
		ok      bool
	)
	_, err = q.ProcessItem(ctx, ProcessItem{I: item}, func(ctx context.Context, _ RunInfo, _ Item) (RunResult, error) {
		latency, ok = GetItemStartLatency(ctx)
		return RunResult{}, nil
	})
	require.NoError(t, err)
	require.True(t, ok, "the start latency must be set on the run context")
	require.Equal(t, 3*time.Second, latency)
}

func TestGetItemStartLatency(t *testing.T) {
	_, ok := GetItemStartLatency(context.Background())
	require.False(t, ok)

	latency, ok := GetItemStartLatency(WithItemStartLatency(context.Background(), time.Second))
	require.True(t, ok)
	require.Equal(t, time.Second, latency)
}
