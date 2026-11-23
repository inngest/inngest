package queue

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type repairBudgetTestShard struct {
	*mockShardForIterator
	clock  *clockwork.FakeClock
	items  []*QueueItem
	calls  atomic.Int32
	repair func(context.Context) error
}

func (s *repairBudgetTestShard) Requeue(ctx context.Context, _ QueueItem, _ time.Time, _ ...RequeueOptionFn) error {
	s.calls.Add(1)
	if s.repair != nil {
		return s.repair(ctx)
	}
	return nil
}

func (s *repairBudgetTestShard) PartitionLease(_ context.Context, _ *QueuePartition, duration time.Duration, _ ...PartitionLeaseOpt) (*ulid.ULID, error) {
	atomic.AddInt32(&s.partitionLeaseCount, 1)
	id := ulid.MustNew(ulid.Timestamp(s.clock.Now().Add(duration)), nil)
	return &id, nil
}

func (s *repairBudgetTestShard) Peek(context.Context, *QueuePartition, time.Time, int64) ([]*QueueItem, error) {
	return s.items, nil
}

func repairBudgetProcessor(t *testing.T, clock *clockwork.FakeClock, shardName string, budget *KeyQueueCapacityLeaseRepairBudget) (*queueProcessor, *repairBudgetTestShard, *QueuePartition) {
	t.Helper()
	account, env, fn := uuid.New(), uuid.New(), uuid.New()
	expired := ulid.MustNew(ulid.Timestamp(clock.Now().Add(-time.Second)), nil)
	item := QueueItem{ID: "expired", AtMS: clock.Now().UnixMilli(), FunctionID: fn, WorkspaceID: env,
		RefilledFrom: "backlog", CapacityLease: &CapacityLease{LeaseID: expired},
		Data: Item{Kind: KindStart, Identifier: state.Identifier{AccountID: account, WorkspaceID: env, WorkflowID: fn}},
	}
	valid := item
	valid.ID = "valid"
	valid.CapacityLease = &CapacityLease{LeaseID: ulid.MustNew(ulid.Timestamp(clock.Now().Add(time.Hour)), nil)}
	second, third := item, item
	second.ID, third.ID = "expired-2", "expired-3"
	shard := &repairBudgetTestShard{mockShardForIterator: &mockShardForIterator{name: shardName}, clock: clock, items: []*QueueItem{&item, &second, &third, &valid}}
	registry, err := NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := New(t.Context(), "test", registry, WithClock(clock), WithNumWorkers(10),
		WithCapacityManager(&refillPauseCapacityManager{}), WithKeyQueueCapacityLeaseRepairBudget(budget),
		WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return true }),
		WithAllowKeyQueueCapacityLeaseRepair(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return true }),
		WithRunMode(QueueRunMode{Continuations: true}),
	)
	require.NoError(t, err)
	p := &QueuePartition{ID: fn.String(), FunctionID: &fn, EnvID: &env, AccountID: account}
	p.LeaseID = ptrULID(clock.Now().Add(PartitionLeaseDuration))
	return q, shard, p
}

func ptrULID(at time.Time) *ulid.ULID {
	id := ulid.MustNew(ulid.Timestamp(at), nil)
	return &id
}

func TestKeyQueueCapacityLeaseRepairLimits(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*KeyQueueCapacityLeaseRepairLimits)
	}{
		{"concurrency", func(l *KeyQueueCapacityLeaseRepairLimits) { l.Concurrency = 0 }},
		{"rate", func(l *KeyQueueCapacityLeaseRepairLimits) { l.Rate = -1 }},
		{"nan", func(l *KeyQueueCapacityLeaseRepairLimits) { l.Rate = math.NaN() }},
		{"infinity", func(l *KeyQueueCapacityLeaseRepairLimits) { l.Rate = math.Inf(1) }},
		{"burst", func(l *KeyQueueCapacityLeaseRepairLimits) { l.Burst = 0 }},
		{"duration", func(l *KeyQueueCapacityLeaseRepairLimits) { l.MaxDuration = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limits := DefaultKeyQueueCapacityLeaseRepairLimits()
			tc.change(&limits)
			_, err := NewKeyQueueCapacityLeaseRepairBudget(limits)
			require.Error(t, err)
		})
	}
}

func TestCapacityLeaseRepairBudgetSharedAcrossProcessors(t *testing.T) {
	clock := clockwork.NewFakeClock()
	limits := DefaultKeyQueueCapacityLeaseRepairLimits()
	limits.Rate, limits.Burst = 2, 2
	budget, err := NewKeyQueueCapacityLeaseRepairBudget(limits)
	require.NoError(t, err)
	q1, s1, p1 := repairBudgetProcessor(t, clock, "shared", budget)
	q2, s2, p2 := repairBudgetProcessor(t, clock, "shared", budget)
	q3, s3, p3 := repairBudgetProcessor(t, clock, "other", budget)
	for i, tc := range []struct {
		q    *queueProcessor
		s    *repairBudgetTestShard
		p    *QueuePartition
		want int32
	}{{q1, s1, p1, 2}, {q2, s2, p2, 0}, {q3, s3, p3, 2}} {
		kept, result, err := tc.q.requeueExpiredCapacityLeases(t.Context(), tc.p, tc.s.items)
		require.NoError(t, err)
		require.Equal(t, tc.want, tc.s.calls.Load(), "processor %d", i)
		require.Len(t, kept, 1, "expired snapshots cannot fall through to ordinary admission")
		require.Equal(t, "valid", kept[0].ID)
		require.Equal(t, "rate", result.reason)
	}
	clock.Advance(time.Second)
	_, _, err = q2.requeueExpiredCapacityLeases(t.Context(), p2, s2.items)
	require.NoError(t, err)
	require.EqualValues(t, 2, s2.calls.Load(), "rate budget replenishes with time")
}

func TestCapacityLeaseRepairSemaphoreDoesNotWait(t *testing.T) {
	clock := clockwork.NewFakeClock()
	limits := DefaultKeyQueueCapacityLeaseRepairLimits()
	limits.Concurrency = 1
	limits.MaxDuration = time.Second
	budget, err := NewKeyQueueCapacityLeaseRepairBudget(limits)
	require.NoError(t, err)
	q1, s1, p1 := repairBudgetProcessor(t, clock, "shared", budget)
	q2, s2, p2 := repairBudgetProcessor(t, clock, "shared", budget)
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	s1.items = s1.items[:1]
	s1.repair = func(ctx context.Context) error {
		close(entered)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	done := make(chan error, 1)
	go func() { _, _, err := q1.requeueExpiredCapacityLeases(t.Context(), p1, s1.items); done <- err }()
	<-entered
	require.Zero(t, q1.Semaphore().Count(), "repair must not occupy customer execution slots")
	kept, result, err := q2.requeueExpiredCapacityLeases(t.Context(), p2, s2.items)
	require.NoError(t, err)
	require.Equal(t, "concurrency", result.reason)
	require.Zero(t, s2.calls.Load())
	require.Len(t, kept, 1)
	once.Do(func() { close(release) })
	require.NoError(t, <-done)
	_, _, err = q2.requeueExpiredCapacityLeases(t.Context(), p2, s2.items)
	require.NoError(t, err)
	require.EqualValues(t, 3, s2.calls.Load(), "repair slot is released after the batch")
}

func TestCapacityLeaseRepairDefaultBudgetIsShared(t *testing.T) {
	clock := clockwork.NewFakeClock()
	shardName := uuid.NewString()
	q1, s1, p1 := repairBudgetProcessor(t, clock, shardName, nil)
	q2, s2, p2 := repairBudgetProcessor(t, clock, shardName, nil)
	pool := defaultKeyQueueCapacityLeaseRepairBudget.shard(shardName)
	require.True(t, pool.limiter.AllowN(clock.Now(), pool.limiter.Burst()-2))
	_, _, err := q1.requeueExpiredCapacityLeases(t.Context(), p1, s1.items)
	require.NoError(t, err)
	_, result, err := q2.requeueExpiredCapacityLeases(t.Context(), p2, s2.items)
	require.NoError(t, err)
	require.EqualValues(t, 2, s1.calls.Load())
	require.Zero(t, s2.calls.Load())
	require.Equal(t, "rate", result.reason)
}

func TestCapacityLeaseRepairLowRateBoundsBackoff(t *testing.T) {
	clock := clockwork.NewFakeClock()
	limits := DefaultKeyQueueCapacityLeaseRepairLimits()
	limits.Rate, limits.Burst = 0.01, 1
	budget, err := NewKeyQueueCapacityLeaseRepairBudget(limits)
	require.NoError(t, err)
	q, s, p := repairBudgetProcessor(t, clock, "test", budget)
	require.True(t, budget.shard("test").limiter.AllowN(clock.Now(), 1))
	kept, result, err := q.requeueExpiredCapacityLeases(t.Context(), p, s.items)
	require.NoError(t, err)
	require.Len(t, kept, 1)
	require.Zero(t, s.calls.Load())
	require.Equal(t, "rate", result.reason)
	require.True(t, result.retryAt.Before(clock.Now().Add(6*time.Second)), "normal work cannot wait for the 100-second token refill")
	require.True(t, result.retryAt.After(clock.Now().Add(PartitionLookahead)))
}

func TestCapacityLeaseRepairTimeBudget(t *testing.T) {
	for _, tc := range []struct {
		name               string
		maxDuration, lease time.Duration
	}{
		{"configured budget", 100 * time.Millisecond, PartitionLeaseDuration},
		{"remaining lease", time.Second, 400 * time.Millisecond},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clock := clockwork.NewFakeClock()
			limits := DefaultKeyQueueCapacityLeaseRepairLimits()
			limits.MaxDuration = tc.maxDuration
			budget, err := NewKeyQueueCapacityLeaseRepairBudget(limits)
			require.NoError(t, err)
			q, s, p := repairBudgetProcessor(t, clock, "test", budget)
			p.LeaseID = ptrULID(clock.Now().Add(tc.lease))
			s.repair = func(context.Context) error { clock.Advance(75 * time.Millisecond); return nil }
			kept, result, err := q.requeueExpiredCapacityLeases(t.Context(), p, s.items)
			require.NoError(t, err)
			require.EqualValues(t, 2, s.calls.Load())
			require.Equal(t, "time", result.reason)
			require.Len(t, kept, 1)
		})
	}
}

func TestCapacityLeaseRepairTimeoutKeepsAdmissionContext(t *testing.T) {
	clock := clockwork.NewFakeClock()
	limits := DefaultKeyQueueCapacityLeaseRepairLimits()
	limits.MaxDuration = 5 * time.Millisecond
	budget, err := NewKeyQueueCapacityLeaseRepairBudget(limits)
	require.NoError(t, err)
	q, s, p := repairBudgetProcessor(t, clock, "test", budget)
	s.repair = func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }
	kept, result, err := q.requeueExpiredCapacityLeases(t.Context(), p, s.items)
	require.NoError(t, err)
	require.NoError(t, t.Context().Err())
	require.EqualValues(t, 1, s.calls.Load())
	require.Equal(t, "time", result.reason)
	require.Len(t, kept, 1)
	require.True(t, budget.shard("test").sem.TryAcquire(2))
	budget.shard("test").sem.Release(2)
	s.repair = func(context.Context) error { return errors.New("redis unavailable") }
	_, _, err = q.requeueExpiredCapacityLeases(t.Context(), p, s.items)
	require.ErrorContains(t, err, "redis unavailable")
	require.True(t, budget.shard("test").sem.TryAcquire(2))
	budget.shard("test").sem.Release(2)
}

func TestProcessPartitionDefersRepairButDispatchesReservations(t *testing.T) {
	clock := clockwork.NewFakeClock()
	budget, err := NewKeyQueueCapacityLeaseRepairBudget(DefaultKeyQueueCapacityLeaseRepairLimits())
	require.NoError(t, err)
	q, s, p := repairBudgetProcessor(t, clock, "test", budget)
	require.True(t, budget.shard("test").limiter.AllowN(clock.Now(), 100))
	q.addContinue(t.Context(), p, 1)
	dispatched := 0
	err = q.ProcessPartition(t.Context(), p, 0, false, func(_ context.Context, item ProcessItem) (DispatchedItem, error) {
		require.Equal(t, "valid", item.I.ID)
		dispatched++
		q.Semaphore().Release(1)
		return NewCompletedDispatchedItem(DispatchedItemResult{}), nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, dispatched)
	require.Zero(t, s.calls.Load())
	require.True(t, s.partitionRequeueForceAt)
	require.True(t, s.partitionRequeueAt.After(clock.Now().Add(PartitionLookahead)))
	require.Empty(t, q.continues)
	q.addContinue(t.Context(), p, 2)
	require.Empty(t, q.continues, "completion callbacks cannot bypass repair backoff")
	leases := atomic.LoadInt32(&s.partitionLeaseCount)
	require.NoError(t, q.ProcessPartition(t.Context(), p, 1, false, nil))
	require.Equal(t, leases, atomic.LoadInt32(&s.partitionLeaseCount))
	clock.Advance(s.partitionRequeueAt.Sub(clock.Now()) + time.Millisecond)
	q.addContinue(t.Context(), p, 2)
	require.Len(t, q.continues, 1, "continuations resume when the repair backoff expires")
}
