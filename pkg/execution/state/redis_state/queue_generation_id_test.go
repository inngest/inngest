package redis_state

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
)

// TestQueueGenerationID asserts the GenerationID lifecycle: Enqueue starts at
// 1 (so 0 is reserved as a "pre-rollout / no value sent" sentinel for the
// validator), Lease and ExtendLease leave it untouched, Requeue bumps it
// monotonically.
func TestQueueGenerationID(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()
	_, shard := newQueue(t, rc, osqueue.WithClock(clock))

	ctx := context.Background()

	enqueueLeasedItem := func(t *testing.T) (osqueue.QueueItem, ulid.ULID) {
		t.Helper()
		fnID, accountID := uuid.New(), uuid.New()
		runID := ulid.MustNew(ulid.Now(), rand.Reader)

		item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Identifier: state.Identifier{
					RunID:      runID,
					WorkflowID: fnID,
					AccountID:  accountID,
				},
			},
		}, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		require.Equal(t, 1, getQueueItem(t, r, item.ID).GenerationID,
			"GenerationID should start at 1 so the first dispatch carries a non-zero value the validator can fence against")

		leaseID, err := shard.Lease(ctx, item, time.Second, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		return item, *leaseID
	}

	t.Run("Lease leaves GenerationID untouched", func(t *testing.T) {
		item, _ := enqueueLeasedItem(t)

		stored := getQueueItem(t, r, item.ID)
		require.Equal(t, 1, stored.GenerationID,
			"Lease must not touch GenerationID; the value carried into the dispatch is whatever was stamped on enqueue/last requeue")
	})

	t.Run("ExtendLease preserves GenerationID", func(t *testing.T) {
		item, leaseID := enqueueLeasedItem(t)

		before := getQueueItem(t, r, item.ID).GenerationID

		clock.Advance(100 * time.Millisecond)
		r.SetTime(clock.Now())

		_, err := shard.ExtendLease(ctx, item, leaseID, time.Second)
		require.NoError(t, err)

		after := getQueueItem(t, r, item.ID).GenerationID
		require.Equal(t, before, after,
			"ExtendLease must not bump GenerationID; a single dispatch (lease + extensions) keeps one ID")
	})

	t.Run("Requeue bumps GenerationID monotonically", func(t *testing.T) {
		item, _ := enqueueLeasedItem(t)

		before := getQueueItem(t, r, item.ID).GenerationID

		clock.Advance(100 * time.Millisecond)
		r.SetTime(clock.Now())

		require.NoError(t, shard.Requeue(ctx, item, clock.Now()))

		after := getQueueItem(t, r, item.ID).GenerationID
		require.Equal(t, before+1, after,
			"Requeue must increment GenerationID so any in-flight SDK from the prior dispatch fails server-side validation")

		// Bump again to confirm monotonicity holds across multiple requeues
		// (e.g. transient errors that don't bump Data.Attempt still bump
		// GenerationID).
		updated := item
		updated.GenerationID = after
		clock.Advance(100 * time.Millisecond)
		r.SetTime(clock.Now())
		require.NoError(t, shard.Requeue(ctx, updated, clock.Now()))
		require.Equal(t, after+1, getQueueItem(t, r, item.ID).GenerationID,
			"successive Requeues must keep bumping GenerationID")
	})
}
