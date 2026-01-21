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
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
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
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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

		// Lease item in legacy/fallback mode (do not disable lease checks)
		leaseID, err := shard.Lease(ctx, item, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		// Check that partition scavenger index + concurrency index are populated
		require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item.ID)))
		require.True(t, r.Exists(kg.ConcurrencyIndex()))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

		// Legacy: Since we did not disable lease checks,
		require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item.ID)))

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

		// Legacy: Since we did not disable lease checks,
		require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
		require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item.ID)))

		// Dequeue item and check scavenger index was cleaned up
		err = q.Dequeue(ctx, shard, item, osqueue.DequeueOptionDisableConstraintUpdates(false))
		require.NoError(t, err)

		require.False(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.False(t, r.Exists(kg.ConcurrencyIndex()))
		// Legacy: Since we did not disable lease checks,
		require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
	})

	t.Run("in-progress items must be added to scavenger index - requeue", func(t *testing.T) {
		r.FlushAll()

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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
		leaseID, err := shard.Lease(ctx, item, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())

		require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		require.True(t, r.Exists(kg.ConcurrencyIndex()))

		// Requeue item and check scavenger index was cleaned up
		err = shard.Requeue(ctx, item, clock.Now().Add(5*time.Second), osqueue.RequeueOptionDisableConstraintUpdates(false))
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
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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

		// Lease item in legacy/fallback mode (do not disable lease checks)
		leaseID1, err := shard.Lease(ctx, item1, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
		require.NoError(t, err)
		require.NotNil(t, leaseID1)

		leaseID2, err := shard.Lease(ctx, item2, 3*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
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

	// NOTE: This test validates scavenging logic continues to work when we progressively switch to the Constraint API.
	// The idea here is that initially we will always update constraint state and add items to the in progress set,
	// when rolling out the new scavenger code we will always add to the scavenger partition index,
	// and when using valid capacity leases we will stop updating constraint state while still updating the new scavenger partition index
	// to ensure in-progress items are requeued in case a worker dies.
	t.Run("when mixing items with and without constraint checks, the scavenger index remains consistent", func(t *testing.T) {
		t.Run("old executors must not break new items", func(t *testing.T) {
			// Scenario:
			// - New executors will start populating scavenger index while still updating the in progress (concurrency) sets
			// - Old executors will finish processing the existing runs while only considering the concurrency sets
			//
			// Not actually true:
			// - If we roll out without another flag, the old executor may remove pointers to the scavenger index because they do not read from the new index
			// - We should only start populating the new index once all executors have rolled out to consume from both old + new
			//
			// What happens:
			// - Since we keep writing to the in progress/concurrency set consumed by the existing operations on old executor pods,
			// we will not drop pointers to partitions from the global concurrency index
			// - The only thing we _cannot_ do until all old executor pods have terminated is to stop constraint checks/updates,
			// as that would mean we _only_ write to the new index which is not read by _all_ executor pods.

			r.FlushAll()

			q, shard := newQueue(
				t, rc,
				osqueue.WithClock(clock),
				osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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

			// Perform initial lease without writing to the new index
			leaseID, err := shard.Lease(ctx, item, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
			require.NoError(t, err)
			require.NotNil(t, leaseID)

			leaseExpiry := clock.Now().Add(5 * time.Second)
			require.True(t, r.Exists(kg.ConcurrencyIndex()))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

			// It's critical that we write to the existing concurrency set which is checked by old Lease, Extend, Requeue, Dequeue scripts
			require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item.ID)))

			// This will not be read by the old executors!
			require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item.ID)))

			clock.Advance(time.Second)
			r.FastForward(time.Second)
			r.SetTime(clock.Now())

			leaseID, err = shard.ExtendLease(ctx, item, *leaseID, 5*time.Second, osqueue.ExtendLeaseOptionDisableConstraintUpdates(false))
			require.NoError(t, err)
			require.NotNil(t, leaseID)
			leaseExpiry = clock.Now().Add(5 * time.Second)

			require.True(t, r.Exists(kg.ConcurrencyIndex()))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

			// It's critical that we extend the item in the existing concurrency set which is checked by old Lease, Extend, Requeue, Dequeue scripts
			require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item.ID)))

			require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
			require.Equal(t, leaseExpiry.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item.ID)))

			err = q.Requeue(ctx, shard, item, clock.Now(), osqueue.RequeueOptionDisableConstraintUpdates(false))
			require.NoError(t, err)

			require.False(t, r.Exists(kg.ConcurrencyIndex()))
			require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
			require.False(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
		})

		t.Run("lease does not check for existing leases and only updates if theres no earlier lease", func(t *testing.T) {
			t.Run("earlier in progress item exists", func(t *testing.T) {
				r.FlushAll()

				_, shard := newQueue(
					t, rc,
					osqueue.WithClock(clock),
					osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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

				// Simulate existing lease valid for another second
				leaseExpiry1 := clock.Now().Add(time.Second)
				_, err = r.ZAdd(kg.Concurrency("p", fnID.String()), float64(leaseExpiry1.UnixMilli()), item1.ID)
				require.NoError(t, err)

				_, err = r.ZAdd(kg.ConcurrencyIndex(), float64(leaseExpiry1.UnixMilli()), fnID.String())
				require.NoError(t, err)

				// Perform initial lease without writing to the new index
				leaseExpiry2 := clock.Now().Add(5 * time.Second)
				leaseID, err := shard.Lease(ctx, item2, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
				require.NoError(t, err)
				require.NotNil(t, leaseID)

				// New item must exist in partition scavenger index
				require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item2.ID)))

				// Global index must NOT have earlier timestamp after removing concurrency key
				require.True(t, r.Exists(kg.ConcurrencyIndex()))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

				// Concurrency index must have both items
				require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
				require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item1.ID)))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item2.ID)))
			})

			t.Run("earlier item in scavenger index", func(t *testing.T) {
				r.FlushAll()

				_, shard := newQueue(
					t, rc,
					osqueue.WithClock(clock),
					osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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
				leaseID, err := shard.Lease(ctx, item1, 3*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(true)) // NOTE: Do not update concurrency index!
				require.NoError(t, err)
				require.NotNil(t, leaseID)

				leaseID2, err := shard.Lease(ctx, item2, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
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
				require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
				require.False(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), item1.ID))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item2.ID)))
			})
		})

		t.Run("extend does not checks for existing leases and only updates if theres no earlier lease", func(t *testing.T) {
			t.Run("earlier in progress item exists", func(t *testing.T) {
				r.FlushAll()

				_, shard := newQueue(
					t, rc,
					osqueue.WithClock(clock),
					osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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

				// Simulate existing lease valid for another second
				leaseExpiry1 := clock.Now().Add(time.Second)
				_, err = r.ZAdd(kg.Concurrency("p", fnID.String()), float64(leaseExpiry1.UnixMilli()), item1.ID)
				require.NoError(t, err)

				_, err = r.ZAdd(kg.ConcurrencyIndex(), float64(leaseExpiry1.UnixMilli()), fnID.String())
				require.NoError(t, err)

				// Lease item first
				leaseID, err := shard.Lease(ctx, item2, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
				require.NoError(t, err)
				require.NotNil(t, leaseID)

				// Then push back lease with expiry which is still later than earliest leased item
				leaseExpiry2 := clock.Now().Add(3 * time.Second)
				leaseID, err = shard.ExtendLease(ctx, item2, *leaseID, 3*time.Second, osqueue.ExtendLeaseOptionDisableConstraintUpdates(false))
				require.NoError(t, err)
				require.NotNil(t, leaseID)

				// New item must exist in partition scavenger index
				require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item2.ID)))

				// Global index must NOT have earlier timestamp
				require.True(t, r.Exists(kg.ConcurrencyIndex()))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

				// Concurrency index must have both items
				require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
				require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item1.ID)))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item2.ID)))
			})

			t.Run("earlier item in scavenger index", func(t *testing.T) {
				r.FlushAll()

				_, shard := newQueue(
					t, rc,
					osqueue.WithClock(clock),
					osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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
				leaseID, err := shard.Lease(ctx, item1, 3*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(true)) // NOTE: Do not update concurrency index!
				require.NoError(t, err)
				require.NotNil(t, leaseID)

				leaseID2, err := shard.Lease(ctx, item2, 2*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
				require.NoError(t, err)
				require.NotNil(t, leaseID2)

				leaseID2, err = shard.ExtendLease(ctx, item2, *leaseID2, 5*time.Second, osqueue.ExtendLeaseOptionDisableConstraintUpdates(false))
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
				require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
				require.False(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), item1.ID))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item2.ID)))
			})
		})

		t.Run("requeue checks for existing leases and updates to earliest lease", func(t *testing.T) {
			t.Run("no more leases in either should drop pointer to function", func(t *testing.T) {
				r.FlushAll()

				q, shard := newQueue(
					t, rc,
					osqueue.WithClock(clock),
					osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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

				// Simulate existing item in the future
				leaseExpiry1 := clock.Now().Add(5 * time.Second)
				_, err = r.ZAdd(kg.Concurrency("p", fnID.String()), float64(leaseExpiry1.UnixMilli()), item1.ID)
				require.NoError(t, err)

				_, err = r.ZAdd(kg.ConcurrencyIndex(), float64(leaseExpiry1.UnixMilli()), fnID.String())
				require.NoError(t, err)

				// Lease item closer to now
				leaseID, err := shard.Lease(ctx, item2, 3*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
				require.NoError(t, err)
				require.NotNil(t, leaseID)

				// Then push back lease with expiry which is still later than earliest leased item
				leaseExpiry2 := clock.Now().Add(3 * time.Second)

				// New item must exist in partition scavenger index
				require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item2.ID)))

				// Global index must have earlier timestamp
				require.True(t, r.Exists(kg.ConcurrencyIndex()))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

				// Concurrency index must have both items
				require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
				require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item1.ID)))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item2.ID)))

				err = q.Requeue(ctx, shard, item1, clock.Now().Add(time.Minute))
				require.NoError(t, err)

				err = q.Requeue(ctx, shard, item2, clock.Now().Add(time.Minute))
				require.NoError(t, err)

				// Must be fully cleaned up
				require.False(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
				require.False(t, r.Exists(kg.ConcurrencyIndex()))
				require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
			})

			t.Run("does not update to next earliest old lease", func(t *testing.T) {
				r.FlushAll()

				q, shard := newQueue(
					t, rc,
					osqueue.WithClock(clock),
					osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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

				// Simulate existing item in the future
				leaseExpiry1 := clock.Now().Add(5 * time.Second)
				_, err = r.ZAdd(kg.Concurrency("p", fnID.String()), float64(leaseExpiry1.UnixMilli()), item1.ID)
				require.NoError(t, err)

				_, err = r.ZAdd(kg.ConcurrencyIndex(), float64(leaseExpiry1.UnixMilli()), fnID.String())
				require.NoError(t, err)

				// Lease item closer to now
				leaseID, err := shard.Lease(ctx, item2, 3*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
				require.NoError(t, err)
				require.NotNil(t, leaseID)

				// Then push back lease with expiry which is still later than earliest leased item
				leaseExpiry2 := clock.Now().Add(3 * time.Second)

				// New item must exist in partition scavenger index
				require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item2.ID)))

				// Global index must have earlier timestamp
				require.True(t, r.Exists(kg.ConcurrencyIndex()))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

				// Concurrency index must have both items
				require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
				require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item1.ID)))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item2.ID)))

				err = q.Requeue(ctx, shard, item2, clock.Now().Add(time.Minute))
				require.NoError(t, err)

				// Scavenger index must be empty (since only old item exists now)
				require.False(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))

				// Global index must be empty since no more in index
				require.False(t, r.Exists(kg.ConcurrencyIndex()))

				// Concurrency index must have only one item
				require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
				require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item1.ID)))
				require.False(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), item2.ID))
			})

			t.Run("update to next earliest item in scavenger index", func(t *testing.T) {
				r.FlushAll()

				q, shard := newQueue(
					t, rc,
					osqueue.WithClock(clock),
					osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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
				leaseID, err := shard.Lease(ctx, item1, 2*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(true)) // NOTE: Do not update concurrency index!
				require.NoError(t, err)
				require.NotNil(t, leaseID)

				leaseID2, err := shard.Lease(ctx, item2, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
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

				// Concurrency index must only have second item
				require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
				require.False(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), item1.ID))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item2.ID)))

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

				// Concurrency index must only have second item
				require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
				require.False(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), item1.ID))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item2.ID)))
			})
		})

		t.Run("dequeue checks for existing leases and updates to earliest lease", func(t *testing.T) {
			t.Run("no more leases in either should drop pointer to function", func(t *testing.T) {
				r.FlushAll()

				q, shard := newQueue(
					t, rc,
					osqueue.WithClock(clock),
					osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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

				// Simulate existing item in the future
				leaseExpiry1 := clock.Now().Add(5 * time.Second)
				_, err = r.ZAdd(kg.Concurrency("p", fnID.String()), float64(leaseExpiry1.UnixMilli()), item1.ID)
				require.NoError(t, err)

				_, err = r.ZAdd(kg.ConcurrencyIndex(), float64(leaseExpiry1.UnixMilli()), fnID.String())
				require.NoError(t, err)

				// Lease item closer to now
				leaseID, err := shard.Lease(ctx, item2, 3*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
				require.NoError(t, err)
				require.NotNil(t, leaseID)

				// Then push back lease with expiry which is still later than earliest leased item
				leaseExpiry2 := clock.Now().Add(3 * time.Second)

				// New item must exist in partition scavenger index
				require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item2.ID)))

				// Global index must have earlier timestamp
				require.True(t, r.Exists(kg.ConcurrencyIndex()))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

				// Concurrency index must have both items
				require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
				require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item1.ID)))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item2.ID)))

				err = q.Dequeue(ctx, shard, item1)
				require.NoError(t, err)

				err = q.Dequeue(ctx, shard, item2)
				require.NoError(t, err)

				// Must be fully cleaned up
				require.False(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
				require.False(t, r.Exists(kg.ConcurrencyIndex()))
				require.False(t, r.Exists(kg.Concurrency("p", fnID.String())))
			})

			t.Run("does not update to next earliest old lease", func(t *testing.T) {
				r.FlushAll()

				q, shard := newQueue(
					t, rc,
					osqueue.WithClock(clock),
					osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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

				// Simulate existing item in the future
				leaseExpiry1 := clock.Now().Add(5 * time.Second)
				_, err = r.ZAdd(kg.Concurrency("p", fnID.String()), float64(leaseExpiry1.UnixMilli()), item1.ID)
				require.NoError(t, err)

				_, err = r.ZAdd(kg.ConcurrencyIndex(), float64(leaseExpiry1.UnixMilli()), fnID.String())
				require.NoError(t, err)

				// Lease item closer to now
				leaseID, err := shard.Lease(ctx, item2, 3*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
				require.NoError(t, err)
				require.NotNil(t, leaseID)

				// Then push back lease with expiry which is still later than earliest leased item
				leaseExpiry2 := clock.Now().Add(3 * time.Second)

				// New item must exist in partition scavenger index
				require.True(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.PartitionScavengerIndex(fnID.String()), item2.ID)))

				// Global index must have earlier timestamp
				require.True(t, r.Exists(kg.ConcurrencyIndex()))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.ConcurrencyIndex(), fnID.String())))

				// Concurrency index must have both items
				require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
				require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item1.ID)))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item2.ID)))

				err = q.Dequeue(ctx, shard, item2)
				require.NoError(t, err)

				// Scavenger index must be empty (since only old item exists now)
				require.False(t, r.Exists(kg.PartitionScavengerIndex(fnID.String())))

				// Global index must be empty since no more items in index
				require.False(t, r.Exists(kg.ConcurrencyIndex()))

				// Concurrency index must have only one item
				require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
				require.Equal(t, leaseExpiry1.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item1.ID)))
				require.False(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), item2.ID))
			})

			t.Run("update to next earliest item in scavenger index", func(t *testing.T) {
				r.FlushAll()

				q, shard := newQueue(
					t, rc,
					osqueue.WithClock(clock),
					osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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
				leaseID, err := shard.Lease(ctx, item1, 2*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(true)) // NOTE: Do not update concurrency index!
				require.NoError(t, err)
				require.NotNil(t, leaseID)

				leaseID2, err := shard.Lease(ctx, item2, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
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

				// Concurrency index must only have second item
				require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
				require.False(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), item1.ID))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item2.ID)))

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

				// Concurrency index must only have second item
				require.True(t, r.Exists(kg.Concurrency("p", fnID.String())))
				require.False(t, hasMember(t, r, kg.Concurrency("p", fnID.String()), item1.ID))
				require.Equal(t, leaseExpiry2.UnixMilli(), int64(score(t, r, kg.Concurrency("p", fnID.String()), item2.ID)))
			})
		})
	})

	t.Run("scavenger must clean up expired leases", func(t *testing.T) {
		r.FlushAll()

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
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
		leaseID, err := shard.Lease(ctx, item1, 5*time.Second, clock.Now(), nil, osqueue.LeaseOptionDisableConstraintChecks(false))
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

	t.Run("scavenging removes leftover traces of key queues", func(t *testing.T) {
		r.FlushAll()

		_, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg
		ctx := context.Background()

		id := uuid.New()

		qi := osqueue.QueueItem{
			FunctionID: id,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
			},
		}

		start := clock.Now().Truncate(time.Second)

		item, err := shard.EnqueueItem(ctx, qi, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		require.NotEqual(t, item.ID, ulid.Zero)
		require.Equal(t, time.UnixMilli(item.WallTimeMS).Truncate(time.Second), start)

		qp := getDefaultPartition(t, r, id)

		leaseStart := clock.Now()
		leaseExpires := clock.Now().Add(time.Second)

		itemCountMatches := func(num int) {
			zsetKey := partitionZsetKey(qp, kg)
			items, err := rc.Do(ctx, rc.B().
				Zrangebyscore().
				Key(zsetKey).
				Min("-inf").
				Max("+inf").
				Build()).AsStrSlice()
			require.NoError(t, err)
			assert.Equal(t, num, len(items), "expected %d items in the queue %q", num, zsetKey, r.Dump())
		}

		concurrencyItemCountMatches := func(num int) {
			items, err := rc.Do(ctx, rc.B().
				Zrangebyscore().
				Key(partitionConcurrencyKey(qp, kg)).
				Min("-inf").
				Max("+inf").
				Build()).AsStrSlice()
			require.NoError(t, err)
			assert.Equal(t, num, len(items), "expected %d items in the concurrency queue", num, r.Dump())
		}

		itemCountMatches(1)
		concurrencyItemCountMatches(0)

		leaseId, err := shard.Lease(ctx, item, time.Second, leaseStart, nil)
		require.NoError(t, err)
		require.NotNil(t, leaseId)

		itemCountMatches(0)
		concurrencyItemCountMatches(1)

		// wait til leases are expired
		clock.Advance(2 * time.Second)
		r.FastForward(2 * time.Second)
		r.SetTime(clock.Now())
		require.True(t, clock.Now().After(leaseExpires))

		incompatibleConcurrencyIndexItem := kg.Concurrency("p", id.String())
		compatibleConcurrencyIndexItem := id.String()

		indexMembers, err := r.ZMembers(kg.ConcurrencyIndex())
		require.NoError(t, err)
		require.Equal(t, 1, len(indexMembers))
		require.Contains(t, indexMembers, compatibleConcurrencyIndexItem)

		leftoverData := []string{
			kg.Concurrency("p", id.String()),
			"{queue}:concurrency:p:0ffd4629-317c-4f65-8b8f-b30fccfde46f",
			"{queue}:concurrency:custom:f:0ffd4629-317c-4f65-8b8f-b30fccfde46f:1nt4mu0skse4a",
		}
		score := float64(leaseStart.Add(time.Second).UnixMilli())
		for _, leftover := range leftoverData {
			_, err = r.ZAdd(kg.ConcurrencyIndex(), score, leftover)
			require.NoError(t, err)
		}
		indexMembers, err = r.ZMembers(kg.ConcurrencyIndex())
		require.NoError(t, err)
		require.Equal(t, 4, len(indexMembers))
		for _, datum := range leftoverData {
			require.Contains(t, indexMembers, datum)
		}

		requeued, err := shard.Scavenge(ctx, osqueue.ScavengePeekSize)
		require.NoError(t, err)
		assert.Equal(t, 1, requeued, "expected one item with expired leases to be requeued by scavenge", r.Dump())

		itemCountMatches(1)
		concurrencyItemCountMatches(0)

		_, err = r.ZMembers(kg.ConcurrencyIndex())
		require.Error(t, err, r.Dump())
		require.ErrorIs(t, err, miniredis.ErrKeyNotFound)

		newConcurrencyQueueItems, err := rc.Do(ctx, rc.B().Zcard().Key(incompatibleConcurrencyIndexItem).Build()).AsInt64()
		require.NoError(t, err)
		assert.Equal(t, 0, int(newConcurrencyQueueItems), "expected no items in the new concurrency queue", r.Dump())

		oldConcurrencyQueueItems, err := rc.Do(ctx, rc.B().Zcard().Key(compatibleConcurrencyIndexItem).Build()).AsInt64()
		require.NoError(t, err)
		assert.Equal(t, 0, int(oldConcurrencyQueueItems), "expected no items in the old concurrency queue", r.Dump())
	})
}
