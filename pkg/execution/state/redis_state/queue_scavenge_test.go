package redis_state

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestQueueScavenge(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	t.Run("in-progress items must be added to scavenger index", func(t *testing.T) {
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		accountID := uuid.New()
		fnID := uuid.New()

		qi := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					AccountID:  accountID,
					WorkflowID: fnID,
				},
			},
		}

		start := time.Now().Truncate(time.Second)

		item, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		leaseExpiry := clock.Now().Add(5 * time.Second)

		leaseID, _, err := shard.Lease(ctx, item, 5*time.Second, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		// Check that partition scavenger index + concurrency index are not populated
		require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item.ID)))
		require.True(t, r.Exists(kg.ConcurrencyIndex()))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

		require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))

		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())

		// Expire lease and expect scores to represent new expiry
		leaseExpiry = clock.Now().Add(5 * time.Second)
		leaseID, err = shard.ExtendLease(ctx, item, *leaseID, 5*time.Second)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item.ID)))
		require.True(t, r.Exists(kg.ConcurrencyIndex()))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

		require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))

		// Dequeue item and check scavenger index was cleaned up
		err = q.Dequeue(ctx, shard, item)
		require.NoError(t, err)

		require.False(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.False(t, r.Exists(kg.ConcurrencyIndex()))
		require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
	})

	t.Run("in-progress items must be added to scavenger index - requeue", func(t *testing.T) {
		r.FlushAll()

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		accountID := uuid.New()
		fnID := uuid.New()

		qi := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					AccountID:  accountID,
					WorkflowID: fnID,
				},
			},
		}

		start := time.Now().Truncate(time.Second)

		item, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		// Lease item in legacy/fallback mode (do not disable lease checks)
		leaseID, _, err := shard.Lease(ctx, item, 5*time.Second, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())

		require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.True(t, r.Exists(kg.ConcurrencyIndex()))

		// Requeue item and check scavenger index was cleaned up
		err = shard.Requeue(ctx, item, clock.Now().Add(5*time.Second))
		require.NoError(t, err)

		require.False(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.False(t, r.Exists(kg.ConcurrencyIndex()))
		// Legacy: Since we did not disable lease checks,
		require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
	})

	t.Run("enqueueing multiple items should lead to earliest lease to expire to be pointer score", func(t *testing.T) {
		r.FlushAll()

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		accountID := uuid.New()
		fnID := uuid.New()

		qi := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					AccountID:  accountID,
					WorkflowID: fnID,
				},
			},
		}

		start := time.Now().Truncate(time.Second)

		item1, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		item2, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		leaseExpiry := clock.Now().Add(5 * time.Second)
		leaseExpiry2 := clock.Now().Add(3 * time.Second)
		require.NotEqual(t, leaseExpiry, leaseExpiry2)

		leaseID1, _, err := shard.Lease(ctx, item1, 5*time.Second, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, leaseID1)

		leaseID2, _, err := shard.Lease(ctx, item2, 3*time.Second, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, leaseID2)

		// Ensure both items are in scavenger index
		require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item1.ID)))
		require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item2.ID)))

		// The earliest expiring lease should become the pointer score
		require.True(t, r.Exists(kg.ConcurrencyIndex()))
		require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))
	})

	t.Run("earlier item in scavenger index", func(t *testing.T) {
		r.FlushAll()

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		accountID := uuid.New()
		fnID := uuid.New()

		qi := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					AccountID:  accountID,
					WorkflowID: fnID,
				},
			},
		}

		start := time.Now().Truncate(time.Second)

		item1, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		item2, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		// Perform initial lease only to the new index
		leaseID, _, err := shard.Lease(ctx, item1, 3*time.Second, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		leaseID2, _, err := shard.Lease(ctx, item2, 5*time.Second, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, leaseID2)

		leaseExpiry1 := clock.Now().Add(3 * time.Second)
		leaseExpiry2 := clock.Now().Add(5 * time.Second)

		// Both items must exist in scavenger index
		require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item1.ID)))
		require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item2.ID)))

		// Global index must have earlier timestamp
		require.True(t, r.Exists(kg.ConcurrencyIndex()))
		require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

		// Concurrency index must only have second item
		require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
	})

	t.Run("extend does not checks for existing leases and only updates if theres no earlier lease", func(t *testing.T) {
		t.Run("earlier item in scavenger index", func(t *testing.T) {
			r.FlushAll()

			_, shard := newQueue(
				t, rc,
				osqueue.WithClock(clock),
				osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
					return true
				}),
			)
			kg := shard.Client().kg
			ctx := context.Background()

			accountID := uuid.New()
			fnID := uuid.New()

			qi := osqueue.QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					Payload: json.RawMessage("{\"test\":\"payload\"}"),
					Identifier: state.Identifier{
						AccountID:  accountID,
						WorkflowID: fnID,
					},
				},
			}

			start := time.Now().Truncate(time.Second)

			item1, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			item2, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			// Perform initial lease only to the new index
			leaseID, _, err := shard.Lease(ctx, item1, 3*time.Second, clock.Now())
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			leaseID2, _, err := shard.Lease(ctx, item2, 2*time.Second, clock.Now())
			require.NoError(t, err)
			require.NotNil(t, leaseID2)

			leaseID2, err = shard.ExtendLease(ctx, item2, *leaseID2, 5*time.Second)
			require.NoError(t, err)
			require.NotNil(t, leaseID2)

			leaseExpiry1 := clock.Now().Add(3 * time.Second)
			leaseExpiry2 := clock.Now().Add(5 * time.Second)

			// Both items must exist in scavenger index
			require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
			require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item1.ID)))
			require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item2.ID)))

			// Global index must have earlier timestamp
			require.True(t, r.Exists(kg.ConcurrencyIndex()))
			require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

			require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
		})
	})

	t.Run("requeue checks for existing leases and updates to earliest lease", func(t *testing.T) {
		t.Run("update to next earliest item in scavenger index", func(t *testing.T) {
			r.FlushAll()

			q, shard := newQueue(
				t, rc,
				osqueue.WithClock(clock),
				osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
					return true
				}),
			)
			kg := shard.Client().kg
			ctx := context.Background()

			accountID := uuid.New()
			fnID := uuid.New()

			qi := osqueue.QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					Payload: json.RawMessage("{\"test\":\"payload\"}"),
					Identifier: state.Identifier{
						AccountID:  accountID,
						WorkflowID: fnID,
					},
				},
			}

			start := time.Now().Truncate(time.Second)

			item1, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			item2, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			// Perform initial lease only to the new index
			leaseID, _, err := shard.Lease(ctx, item1, 2*time.Second, clock.Now())
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			leaseID2, _, err := shard.Lease(ctx, item2, 5*time.Second, clock.Now())
			require.NoError(t, err)
			require.NotNil(t, leaseID2)

			leaseExpiry1 := clock.Now().Add(2 * time.Second)
			leaseExpiry2 := clock.Now().Add(5 * time.Second)

			// Both items must exist in scavenger index
			require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
			require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item1.ID)))
			require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item2.ID)))

			// Global index must have earlier timestamp
			require.True(t, r.Exists(kg.ConcurrencyIndex()))
			require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

			require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))

			err = q.Requeue(ctx, shard, item1, clock.Now().Add(time.Minute))
			require.NoError(t, err)
			require.NotNil(t, leaseID2)

			// Item 1 must be removed
			require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
			require.False(t, hasMember(t, r, kg.PartitionScavengerIndex(fnID.String()), item1.ID))
			require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item2.ID)))

			// Global index must have next timestamp
			require.True(t, r.Exists(kg.ConcurrencyIndex()))
			require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

			require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
		})
	})

	t.Run("dequeue checks for existing leases and updates to earliest lease", func(t *testing.T) {
		t.Run("update to next earliest item in scavenger index", func(t *testing.T) {
			r.FlushAll()

			q, shard := newQueue(
				t, rc,
				osqueue.WithClock(clock),
				osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
					return true
				}),
			)
			kg := shard.Client().kg
			ctx := context.Background()

			accountID := uuid.New()
			fnID := uuid.New()

			qi := osqueue.QueueItem{
				FunctionID: fnID,
				Data: osqueue.Item{
					Payload: json.RawMessage("{\"test\":\"payload\"}"),
					Identifier: state.Identifier{
						AccountID:  accountID,
						WorkflowID: fnID,
					},
				},
			}

			start := time.Now().Truncate(time.Second)

			item1, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			item2, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)

			// Perform initial lease only to the new index
			leaseID, _, err := shard.Lease(ctx, item1, 2*time.Second, clock.Now())
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			leaseID2, _, err := shard.Lease(ctx, item2, 5*time.Second, clock.Now())
			require.NoError(t, err)
			require.NotNil(t, leaseID2)

			leaseExpiry1 := clock.Now().Add(2 * time.Second)
			leaseExpiry2 := clock.Now().Add(5 * time.Second)

			// Both items must exist in scavenger index
			require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
			require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item1.ID)))
			require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item2.ID)))

			// Global index must have earlier timestamp
			require.True(t, r.Exists(kg.ConcurrencyIndex()))
			require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

			require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))

			err = q.Dequeue(ctx, shard, item1)
			require.NoError(t, err)
			require.NotNil(t, leaseID2)

			// Item 1 must be removed
			require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
			require.False(t, hasMember(t, r, kg.PartitionScavengerIndex(fnID.String()), item1.ID))
			require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item2.ID)))

			// Global index must have next timestamp
			require.True(t, r.Exists(kg.ConcurrencyIndex()))
			require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

			require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
		})
	})

	t.Run("scavenger must clean up expired leases", func(t *testing.T) {
		r.FlushAll()

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		accountID := uuid.New()
		fnID := uuid.New()

		qi := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					AccountID:  accountID,
					WorkflowID: fnID,
				},
			},
		}

		start := time.Now().Truncate(time.Second)

		item1, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		// Simulate existing lease valid for another second
		leaseExpiry := clock.Now().Add(5 * time.Second)
		leaseID, _, err := shard.Lease(ctx, item1, 5*time.Second, clock.Now())
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		// First run should not find any items since lease is still valid

		scavenged, err := shard.Scavenge(ctx, 100)
		require.NoError(t, err)
		require.Equal(t, 0, scavenged)

		require.True(t, r.Exists(kg.ConcurrencyIndex()))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item1.ID)))
		require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))

		clock.Advance(6 * time.Second)
		r.FastForward(6 * time.Second)
		r.SetTime(clock.Now())

		require.True(t, clock.Now().After(leaseExpiry))

		scavenged, err = shard.Scavenge(ctx, 100)
		require.NoError(t, err)
		require.Equal(t, 1, scavenged)

		require.False(t, r.Exists(kg.ConcurrencyIndex()))
		require.False(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
	})

	t.Run("scavenger increments ScavengeCount on requeue", func(t *testing.T) {
		r.FlushAll()

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
		)
		ctx := context.Background()

		accountID := uuid.New()
		fnID := uuid.New()

		qi := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					AccountID:  accountID,
					WorkflowID: fnID,
				},
			},
		}

		start := time.Now().Truncate(time.Second)

		item, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		require.Equal(t, 0, item.ScavengeCount)

		// Lease the item and let it expire
		_, _, err = shard.Lease(ctx, item, 5*time.Second, clock.Now())
		require.NoError(t, err)

		clock.Advance(6 * time.Second)
		r.FastForward(6 * time.Second)
		r.SetTime(clock.Now())

		// First scavenge should increment ScavengeCount to 1
		scavenged, err := shard.Scavenge(ctx, 100)
		require.NoError(t, err)
		require.Equal(t, 1, scavenged)

		requeued, err := shard.LoadQueueItem(ctx, item.ID)
		require.NoError(t, err)
		require.Equal(t, 1, requeued.ScavengeCount)

		// Lease and expire again to verify accumulation
		_, _, err = shard.Lease(ctx, *requeued, 5*time.Second, clock.Now())
		require.NoError(t, err)

		clock.Advance(6 * time.Second)
		r.FastForward(6 * time.Second)
		r.SetTime(clock.Now())

		scavenged, err = shard.Scavenge(ctx, 100)
		require.NoError(t, err)
		require.Equal(t, 1, scavenged)

		requeued2, err := shard.LoadQueueItem(ctx, item.ID)
		require.NoError(t, err)
		require.Equal(t, 2, requeued2.ScavengeCount)
	})
}
