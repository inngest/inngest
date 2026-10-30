package redis_state_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// A dispatch outage can leave expired refill reservations ahead of fresh ones.
// Refill then consumes all capacity while FIFO stops before their owners. Use
// real queue/constraint Lua with controlled scanner ordering and a healthy worker.
func TestProcessPartitionRecoversExpiredRefillCapacity(t *testing.T) {
	for _, tc := range []struct {
		name                                 string
		jobs, limit, expiredBatches, seconds int
	}{
		{name: "expired_prefix", jobs: 300, limit: 25, expiredBatches: 1, seconds: 180},
		{name: "beyond_peek", jobs: 1000, limit: 25, expiredBatches: 16, seconds: 600},
		{name: "limit_one", jobs: 30, limit: 1, expiredBatches: 1, seconds: 240},
		{name: "healthy", jobs: 100, limit: 25, seconds: 120},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCapacityRecoveryFixture(t, tc.limit)
			queue.WithPeekSizeRange(32, 32)(f.processor.Options())
			f.enqueue(tc.jobs)
			for batch := 0; batch < tc.expiredBatches; batch++ {
				before := f.ready()
				for attempt := 0; attempt < 3 && f.ready()-before < tc.limit; attempt++ {
					f.refill()
					f.advance(time.Millisecond)
				}
				require.Equal(t, tc.limit, f.ready()-before)
				f.advance(31 * time.Second)
			}
			f.completionDelay = 600 * time.Millisecond
			for second := 0; second < tc.seconds; second++ {
				res, err := f.cm.Scavenge(t.Context())
				require.NoError(t, err)
				f.scavenged += res.ReclaimedLeases
				f.refill()
				// Two refill passes reserve 20 + 5 slots before FIFO scans.
				if second%5 == 2 {
					f.processPartition()
				}
				f.advance(time.Second)
			}
			t.Logf("dispatched=%d ready=%d backlog=%d scavenged=%d", f.dispatched, f.ready(), f.backlogSize(), f.scavenged)
			require.Equal(t, tc.jobs, f.dispatched, "healthy dispatch must recover without disabling key queues")
			require.Zero(t, f.ready())
			require.Zero(t, f.backlogSize())
			require.Zero(t, f.functionUsage())
			require.Empty(t, f.pending)
		})
	}
}

type reproCapacityManager interface {
	constraintapi.CapacityManager
	constraintapi.CapacityLeaseScavenger
}

type capacityRecoveryFixture struct {
	t                *testing.T
	redis            *miniredis.Miniredis
	clock            *clockwork.FakeClock
	shard            redis_state.RedisQueueShard
	processor        queue.QueueProcessor
	cm               reproCapacityManager
	constraints      queue.PartitionConstraintConfig
	shadow           queue.QueueShadowPartition
	backlog          queue.QueueBacklog
	refilled         int
	scavenged        int
	dispatched       int
	dispatchMu       sync.Mutex
	keyQueuesEnabled bool
	dispatchIDs      map[string]bool
	enqueued         int
	completionDelay  time.Duration
	pending          []pendingCompletion
}

type pendingCompletion struct {
	at   time.Time
	item queue.ProcessItem
}

func newCapacityRecoveryFixture(t *testing.T, limit int) *capacityRecoveryFixture {
	t.Helper()
	f := &capacityRecoveryFixture{t: t, redis: miniredis.RunT(t), clock: clockwork.NewFakeClockAt(time.Now().Truncate(time.Second)), keyQueuesEnabled: true}
	rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{f.redis.Addr()}, DisableCache: true})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	f.cm, err = constraintapi.NewRedisCapacityManager(
		constraintapi.WithShardName("repro"), constraintapi.WithClient(rc), constraintapi.WithClock(f.clock),
	)
	require.NoError(t, err)
	f.constraints = queue.PartitionConstraintConfig{
		FunctionVersion: 1,
		Concurrency:     queue.PartitionConcurrency{AccountConcurrency: 10000, FunctionConcurrency: limit},
	}
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	f.shadow = queue.QueueShadowPartition{PartitionID: fnID.String(), AccountID: &accountID, EnvID: &envID, FunctionID: &fnID}
	f.backlog = *f.shadow.DefaultBacklog(f.constraints, true)
	opts := []queue.QueueOpt{
		queue.WithClock(f.clock), queue.WithNumWorkers(100), queue.WithPeekSizeRange(250, 1500),
		queue.WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return f.keyQueuesEnabled }),
		queue.WithCapacityManager(f.cm), queue.WithAcquireCapacityLeaseOnBacklogRefill(true),
		queue.WithPartitionConstraintConfigGetter(func(context.Context, queue.PartitionIdentifier) queue.PartitionConstraintConfig { return f.constraints }),
	}
	f.shard = redis_state.NewQueueShard("repro", redis_state.NewQueueClient(rc, "{q:v1}"), opts...)
	registry, err := queue.NewSingleShardRegistry(f.shard)
	require.NoError(t, err)
	f.processor, err = queue.New(t.Context(), "repro", registry, opts...)
	require.NoError(t, err)
	// Item admission releases expired reservations in background goroutines.
	// Join those before closing this fixture's Redis client.
	t.Cleanup(service.Wait)
	return f
}

func (f *capacityRecoveryFixture) enqueue(n int) {
	f.t.Helper()
	for i := range n {
		at := f.clock.Now().Add(time.Duration(i) * time.Millisecond)
		_, err := f.shard.EnqueueItem(f.t.Context(), queue.QueueItem{
			ID: fmt.Sprintf("job-%04d", f.enqueued+i), FunctionID: *f.shadow.FunctionID, WorkspaceID: *f.shadow.EnvID,
			Data: queue.Item{Kind: queue.KindStart, WorkspaceID: *f.shadow.EnvID, Identifier: state.Identifier{
				AccountID: *f.shadow.AccountID, WorkspaceID: *f.shadow.EnvID, WorkflowID: *f.shadow.FunctionID,
				RunID: ulid.MustNew(ulid.Timestamp(at), nil), WorkflowVersion: 1,
			}},
		}, at, queue.EnqueueOpts{PassthroughJobId: true})
		require.NoError(f.t, err)
	}
	f.enqueued += n
	f.advance(time.Second)
}

func (f *capacityRecoveryFixture) advance(d time.Duration) {
	f.clock.Advance(d)
	f.redis.FastForward(d)
	kept := f.pending[:0]
	for _, completion := range f.pending {
		if !completion.at.After(f.clock.Now()) {
			f.complete(completion.item)
		} else {
			kept = append(kept, completion)
		}
	}
	f.pending = kept
}

func (f *capacityRecoveryFixture) refill() {
	f.t.Helper()
	res, _, err := f.processor.ProcessShadowPartitionBacklog(f.t.Context(), &f.shadow, &f.backlog, f.clock.Now().Add(2*time.Second), f.constraints)
	require.NoError(f.t, err)
	if res != nil {
		f.refilled += len(res.RefilledItems)
	}
}

func (f *capacityRecoveryFixture) processPartition() {
	f.t.Helper()
	if f.ready() == 0 {
		return
	}
	p := &queue.QueuePartition{ID: f.shadow.PartitionID, FunctionID: f.shadow.FunctionID, EnvID: f.shadow.EnvID, AccountID: *f.shadow.AccountID}
	err := f.processor.ProcessPartition(f.t.Context(), p, 0, false, func(ctx context.Context, item queue.ProcessItem) (queue.DispatchedItem, error) {
		f.dispatchMu.Lock()
		defer f.dispatchMu.Unlock()
		if f.dispatchIDs == nil {
			f.dispatchIDs = map[string]bool{}
		}
		require.False(f.t, f.dispatchIDs[item.I.ID], "an item must not dispatch twice")
		f.dispatchIDs[item.I.ID] = true
		require.LessOrEqual(f.t, f.functionUsage(), f.constraints.Concurrency.FunctionConcurrency)
		f.dispatched++
		if f.completionDelay == 0 {
			f.complete(item)
		} else {
			f.pending = append(f.pending, pendingCompletion{at: f.clock.Now().Add(f.completionDelay), item: item})
		}
		// The harness consumes the dispatch acknowledgement synchronously;
		// actual item/capacity leases and semaphore occupancy persist until complete.
		return queue.NewCompletedDispatchedItem(queue.DispatchedItemResult{}), nil
	})
	if errors.Is(err, queue.ErrPartitionGarbageCollected) {
		// Synchronous completion can remove the last ready item before the
		// iterator requeues its partition pointer.
		require.Zero(f.t, f.ready())
		return
	}
	require.NoError(f.t, err)
}

func (f *capacityRecoveryFixture) complete(item queue.ProcessItem) {
	f.t.Helper()
	defer f.processor.Semaphore().Release(1)
	require.NoError(f.t, f.shard.Dequeue(f.t.Context(), item.I, queue.DequeueCapacityLeased()))
	require.NotNil(f.t, item.CapacityLease)
	_, err := f.cm.Release(f.t.Context(), &constraintapi.CapacityReleaseRequest{
		AccountID: *f.shadow.AccountID, LeaseID: item.CapacityLease.LeaseID, IdempotencyKey: "complete-" + item.I.ID,
		Source: constraintapi.LeaseSource{Service: constraintapi.ServiceExecutor, Location: constraintapi.CallerLocationItemLease},
	})
	require.NoError(f.t, err)
}

func (f *capacityRecoveryFixture) ready() int {
	f.t.Helper()
	kg := f.shard.Client().KeyGenerator()
	key := kg.PartitionQueueSet(enums.PartitionTypeDefault, f.shadow.PartitionID, "")
	n, err := f.shard.Client().Client().Do(f.t.Context(), f.shard.Client().Client().B().Zcard().Key(key).Build()).AsInt64()
	require.NoError(f.t, err)
	return int(n)
}

func (f *capacityRecoveryFixture) backlogSize() int64 {
	f.t.Helper()
	n, err := f.shard.BacklogSize(f.t.Context(), f.backlog.BacklogID)
	require.NoError(f.t, err)
	return n
}

func (f *capacityRecoveryFixture) functionUsage() int {
	f.t.Helper()
	res, userErr, err := f.cm.Check(f.t.Context(), &constraintapi.CapacityCheckRequest{
		AccountID: *f.shadow.AccountID, EnvID: *f.shadow.EnvID, FunctionID: *f.shadow.FunctionID,
		Configuration: queue.ConstraintConfigFromConstraints(f.constraints),
		Constraints: []constraintapi.ConstraintItem{{Kind: constraintapi.ConstraintKindConcurrency,
			Concurrency: &constraintapi.ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn}}},
	})
	require.NoError(f.t, err)
	require.Nil(f.t, userErr)
	require.Len(f.t, res.Usage, 1)
	return res.Usage[0].Used
}

// Requeue eligibility must be checked on current Redis state, after the peek.
func TestRequeueIfExpiredCapacityLeaseConditions(t *testing.T) {
	for _, tc := range []struct {
		name        string
		change      func(*capacityRecoveryFixture, *queue.QueueItem, *ulid.ULID)
		wantRequeue bool
		wantErr     error
	}{
		{name: "expired", wantRequeue: true},
		{name: "legacy_zero_generation", wantRequeue: true, change: func(f *capacityRecoveryFixture, item *queue.QueueItem, _ *ulid.ULID) {
			item.GenerationID = 0
			f.store(*item)
		}},
		{name: "item_lease_wins", change: func(f *capacityRecoveryFixture, item *queue.QueueItem, _ *ulid.ULID) {
			_, err := f.shard.Lease(f.t.Context(), *item, queue.QueueLeaseDuration, f.clock.Now())
			require.NoError(f.t, err)
		}},
		{name: "valid_capacity_snapshot", change: func(f *capacityRecoveryFixture, item *queue.QueueItem, _ *ulid.ULID) {
			item.CapacityLease = &queue.CapacityLease{LeaseID: ulid.MustNew(ulid.Timestamp(f.clock.Now().Add(time.Minute)), nil)}
			f.store(*item)
		}},
		{name: "capacity_refreshed_after_peek", change: func(f *capacityRecoveryFixture, item *queue.QueueItem, _ *ulid.ULID) {
			current := *item
			current.CapacityLease = &queue.CapacityLease{LeaseID: ulid.MustNew(ulid.Timestamp(f.clock.Now().Add(time.Minute)), nil)}
			f.store(current)
		}},
		{name: "different_expired_capacity_epoch", change: func(f *capacityRecoveryFixture, item *queue.QueueItem, _ *ulid.ULID) {
			current := *item
			current.CapacityLease = &queue.CapacityLease{LeaseID: ulid.MustNew(ulid.Timestamp(f.clock.Now().Add(-time.Second)), nil)}
			f.store(current)
		}},
		{name: "generation_changed", change: func(f *capacityRecoveryFixture, item *queue.QueueItem, _ *ulid.ULID) {
			current := *item
			current.GenerationID++
			f.store(current)
		}},
		{name: "backlog_changed", change: func(f *capacityRecoveryFixture, item *queue.QueueItem, _ *ulid.ULID) {
			current := *item
			current.RefilledFrom += ":changed"
			f.store(current)
		}},
		{name: "not_ready", change: func(f *capacityRecoveryFixture, item *queue.QueueItem, _ *ulid.ULID) {
			client := f.shard.Client().Client()
			key := f.shard.Client().KeyGenerator().PartitionQueueSet(enums.PartitionTypeDefault, f.shadow.PartitionID, "")
			require.NoError(f.t, client.Do(f.t.Context(), client.B().Zrem().Key(key).Member(item.ID).Build()).Error())
		}},
		{name: "not_due", change: func(f *capacityRecoveryFixture, item *queue.QueueItem, _ *ulid.ULID) {
			item.AtMS = f.clock.Now().Add(time.Minute).UnixMilli()
			f.store(*item)
		}},
		{name: "not_refilled", change: func(f *capacityRecoveryFixture, item *queue.QueueItem, _ *ulid.ULID) {
			item.RefilledFrom = ""
			f.store(*item)
		}},
		{name: "key_queues_disabled", change: func(f *capacityRecoveryFixture, _ *queue.QueueItem, _ *ulid.ULID) {
			f.keyQueuesEnabled = false
		}},
		{name: "expired_partition_lease", wantErr: queue.ErrPartitionLeaseMismatch, change: func(f *capacityRecoveryFixture, _ *queue.QueueItem, _ *ulid.ULID) {
			f.advance(queue.PartitionLeaseDuration + time.Second)
		}},
		{name: "partition_lease_replaced", wantErr: queue.ErrPartitionLeaseMismatch, change: func(f *capacityRecoveryFixture, _ *queue.QueueItem, _ *ulid.ULID) {
			f.advance(queue.PartitionLeaseDuration + time.Second)
			_, err := f.shard.PartitionLease(f.t.Context(), f.partition(), queue.PartitionLeaseDuration)
			require.NoError(f.t, err)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCapacityRecoveryFixture(t, 25)
			f.enqueue(1)
			f.refill()
			f.advance(31 * time.Second)
			items, err := f.shard.Peek(t.Context(), f.partition(), f.clock.Now(), 1)
			require.NoError(t, err)
			require.Len(t, items, 1)
			snapshot := *items[0]
			partitionLease, err := f.shard.PartitionLease(t.Context(), f.partition(), queue.PartitionLeaseDuration)
			require.NoError(t, err)
			if tc.change != nil {
				tc.change(f, &snapshot, partitionLease)
			}
			before, err := f.shard.LoadQueueItem(t.Context(), snapshot.ID)
			require.NoError(t, err)
			ready, backlog := f.ready(), f.backlogSize()
			err = f.shard.Requeue(t.Context(), snapshot, f.clock.Now(), queue.RequeueIfExpiredCapacityLease(*partitionLease))
			require.ErrorIs(t, err, tc.wantErr)
			after, err := f.shard.LoadQueueItem(t.Context(), snapshot.ID)
			require.NoError(t, err)
			if !tc.wantRequeue {
				require.Equal(t, *before, *after, "rejected cleanup must not alter the current item")
				require.Equal(t, ready, f.ready())
				require.Equal(t, backlog, f.backlogSize())
				return
			}
			require.Zero(t, f.ready())
			require.EqualValues(t, 1, f.backlogSize())
			require.Equal(t, before.GenerationID+1, after.GenerationID)
			require.Nil(t, after.CapacityLease)
			require.Equal(t, before.EnqueuedAt, after.EnqueuedAt, "repair preserves original work age")
			require.Equal(t, before.Data.Attempt, after.Data.Attempt)
			// Cleanup wins before a buffered processor's item lease. This also
			// needs to fence explicit generation zero on legacy queued items.
			_, err = f.shard.Lease(t.Context(), snapshot, queue.QueueLeaseDuration, f.clock.Now())
			require.ErrorIs(t, err, queue.ErrQueueItemNotFound)
		})
	}
}

func (f *capacityRecoveryFixture) partition() *queue.QueuePartition {
	return &queue.QueuePartition{ID: f.shadow.PartitionID, FunctionID: f.shadow.FunctionID, EnvID: f.shadow.EnvID, AccountID: *f.shadow.AccountID}
}

func (f *capacityRecoveryFixture) store(item queue.QueueItem) {
	f.t.Helper()
	data, err := json.Marshal(item)
	require.NoError(f.t, err)
	client := f.shard.Client().Client()
	require.NoError(f.t, client.Do(f.t.Context(), client.B().Hset().Key(f.shard.Client().KeyGenerator().QueueItem()).FieldValue().FieldValue(item.ID, string(data)).Build()).Error())
}

type capacityAcquireHook struct {
	constraintapi.CapacityManager
	afterAcquire func()
}

func (m *capacityAcquireHook) Acquire(ctx context.Context, req *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	res, err := m.CapacityManager.Acquire(ctx, req)
	if err == nil && len(res.Leases) > 0 && req.Source.Location == constraintapi.CallerLocationItemLease {
		m.afterAcquire()
	}
	return res, err
}

func TestExpiredCapacityRepairDuringItemAdmission(t *testing.T) {
	f := newCapacityRecoveryFixture(t, 25)
	f.enqueue(1)
	f.refill()
	f.advance(31 * time.Second)
	items, err := f.shard.Peek(t.Context(), f.partition(), f.clock.Now(), 1)
	require.NoError(t, err)
	require.Len(t, items, 1)
	snapshot := *items[0]
	p := f.partition()
	p.LeaseID, err = f.shard.PartitionLease(t.Context(), p, queue.PartitionLeaseDuration)
	require.NoError(t, err)
	var once sync.Once
	f.processor.Options().CapacityManager = &capacityAcquireHook{CapacityManager: f.cm, afterAcquire: func() {
		once.Do(func() {
			err := f.shard.Requeue(t.Context(), snapshot, f.clock.Now(), queue.RequeueIfExpiredCapacityLease(*p.LeaseID))
			require.NoError(t, err)
			f.advance(time.Millisecond)
			// A refill can reserve the repaired generation before the original
			// admission resumes. Its stale item lease must still be fenced.
			f.refill()
		})
	}}
	dispatched := false
	result, err := f.processor.(queue.QueueItemLeaser).LeaseItem(t.Context(), queue.LeaseItemRequest{Item: &snapshot, StaticTime: f.clock.Now()}, func(context.Context, queue.ProcessItem) (queue.DispatchedItem, error) {
		dispatched = true
		return queue.NewCompletedDispatchedItem(queue.DispatchedItemResult{}), nil
	})
	require.NoError(t, err)
	require.Equal(t, queue.LeaseItemStatusNotFound, result.Status)
	require.False(t, dispatched)
	require.Equal(t, 1, f.ready())
	require.Equal(t, 2, f.functionUsage())
	require.Zero(t, f.processor.Semaphore().Count())
	require.NoError(t, f.shard.PartitionRequeue(t.Context(), p, f.clock.Now(), true))
	f.processPartition()
	require.Equal(t, 1, f.dispatched)
	require.Zero(t, f.ready())
	// Capacity acquisition precedes the atomic item lease. As in ordinary
	// lease contention, a losing admission can retain an unused grant until
	// expiry; it must not hold capacity permanently or dispatch stale work.
	require.Equal(t, 1, f.functionUsage())
	f.advance(31 * time.Second)
	_, err = f.cm.Scavenge(t.Context())
	require.NoError(t, err)
	require.Zero(t, f.functionUsage())
}

func TestExpiredCapacityRepairBoundsWorkAndReachesReservations(t *testing.T) {
	f := newCapacityRecoveryFixture(t, 25)
	queue.WithPeekSizeRange(250, 250)(f.processor.Options())
	f.enqueue(300)
	// Put more expired items in one peek than a single pass may repair.
	for range 5 {
		for range 2 {
			f.refill()
			f.advance(time.Millisecond)
		}
		f.advance(31 * time.Second)
	}
	require.Equal(t, 125, f.ready())
	for range 2 {
		f.refill()
		f.advance(time.Millisecond)
	}
	require.Equal(t, 150, f.ready())
	f.processPartition()
	// Repair 100 expired items, skip the remaining expired snapshots, then
	// dispatch the 25 fresh reservation owners in the same bounded peek.
	require.Equal(t, 25, f.dispatched)
	require.Equal(t, 25, f.ready())
	require.EqualValues(t, 250, f.backlogSize())
	require.Zero(t, f.functionUsage())
	key := f.shard.Client().KeyGenerator().GlobalPartitionIndex()
	score, err := f.redis.ZScore(key, f.shadow.PartitionID)
	require.NoError(t, err)
	require.LessOrEqual(t, int64(score), f.clock.Now().Unix(), "remaining repair must be promptly rescanned")
}

func TestExpiredCapacityRepairRespectsKeyQueueDisable(t *testing.T) {
	for _, disableDuringCleanup := range []bool{false, true} {
		name := "disabled_before_scan"
		if disableDuringCleanup {
			name = "disabled_after_processor_check"
		}
		t.Run(name, func(t *testing.T) {
			f := newCapacityRecoveryFixture(t, 25)
			f.enqueue(50)
			for range 2 {
				f.refill()
				f.advance(time.Millisecond)
			}
			require.Equal(t, 25, f.ready())
			require.EqualValues(t, 25, f.backlogSize())
			f.advance(31 * time.Second)

			f.keyQueuesEnabled = disableDuringCleanup
			checks := 0
			queue.WithAllowKeyQueues(func(_ context.Context, accountID, envID, functionID uuid.UUID) bool {
				require.Equal(t, *f.shadow.AccountID, accountID)
				require.Equal(t, *f.shadow.EnvID, envID)
				require.Equal(t, *f.shadow.FunctionID, functionID)
				checks++
				if disableDuringCleanup && checks == 1 {
					// The processor observed enablement, but the shard observes
					// the emergency disable before constructing the Redis mutation.
					f.keyQueuesEnabled = false
					return true
				}
				return f.keyQueuesEnabled
			})(f.processor.Options())

			f.processPartition()
			require.Positive(t, checks)
			require.False(t, f.keyQueuesEnabled)
			require.EqualValues(t, 25, f.backlogSize(), "expired ready items must not be returned to disabled backlogs")
			if disableDuringCleanup {
				require.Equal(t, 24, f.dispatched)
				require.Equal(t, 1, f.ready(), "a rejected cleanup snapshot is retried on the next scan")
			} else {
				require.Equal(t, 25, f.dispatched)
				require.Zero(t, f.ready())
			}

			// Emergency refill still dumps the backlog into ready in bounded
			// batches without capacity reservations, for ordinary admission.
			for range 2 {
				f.refill()
				f.advance(time.Millisecond)
			}
			require.Zero(t, f.backlogSize())
			for i := 25; i < 50; i++ {
				item, err := f.shard.LoadQueueItem(t.Context(), fmt.Sprintf("job-%04d", i))
				require.NoError(t, err)
				require.Nil(t, item.CapacityLease)
			}
			f.processPartition()
			require.Equal(t, 50, f.dispatched)
			require.Zero(t, f.ready())
			require.Zero(t, f.backlogSize())
			require.Zero(t, f.functionUsage())
		})
	}
}
