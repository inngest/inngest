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

// TestQueueDispatchID asserts the DispatchID lifecycle that EXE-1552's fix
// relies on: Lease mints, ExtendLease preserves, Requeue rotates.
func TestQueueDispatchID(t *testing.T) {
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

	enqueueLeasedItem := func(t *testing.T) (osqueue.QueueItem, ulid.ULID, ulid.ULID) {
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

		require.Nil(t, getQueueItem(t, r, item.ID).DispatchID,
			"DispatchID should be nil before first lease")

		leaseID, dispatchID, err := shard.Lease(ctx, item, time.Second, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		require.NotNil(t, dispatchID)

		return item, *leaseID, *dispatchID
	}

	t.Run("Lease mints a DispatchID", func(t *testing.T) {
		item, _, dispatchID := enqueueLeasedItem(t)

		stored := getQueueItem(t, r, item.ID)
		require.NotNil(t, stored.DispatchID, "Lease should mint a DispatchID")
		require.Equal(t, dispatchID.String(), stored.DispatchID.String(),
			"Lease should return the same DispatchID stored on the queue item")
	})

	t.Run("ExtendLease preserves the DispatchID", func(t *testing.T) {
		item, leaseID, _ := enqueueLeasedItem(t)

		before := getQueueItem(t, r, item.ID).DispatchID
		require.NotNil(t, before)

		clock.Advance(100 * time.Millisecond)
		r.SetTime(clock.Now())

		_, err := shard.ExtendLease(ctx, item, leaseID, time.Second)
		require.NoError(t, err)

		after := getQueueItem(t, r, item.ID).DispatchID
		require.NotNil(t, after)
		require.Equal(t, before.String(), after.String(),
			"ExtendLease must not rotate DispatchID; a single dispatch (lease + extensions) shares one ID")
	})

	t.Run("Requeue rotates the DispatchID", func(t *testing.T) {
		item, _, _ := enqueueLeasedItem(t)

		before := getQueueItem(t, r, item.ID).DispatchID
		require.NotNil(t, before)

		clock.Advance(100 * time.Millisecond)
		r.SetTime(clock.Now())

		require.NoError(t, shard.Requeue(ctx, item, clock.Now()))

		after := getQueueItem(t, r, item.ID).DispatchID
		require.NotNil(t, after, "Requeue must mint a new DispatchID")
		require.NotEqual(t, before.String(), after.String(),
			"Requeue must rotate DispatchID so any in-flight SDK from the prior dispatch fails server-side validation")
	})
}
