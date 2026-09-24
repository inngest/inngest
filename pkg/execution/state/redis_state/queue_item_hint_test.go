package redis_state

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestItemHintLeaseLookahead(t *testing.T) {
	for _, backlog := range []bool{false, true} {
		for _, ahead := range []time.Duration{0, 2 * time.Second} {
			t.Run(fmt.Sprintf("backlog=%t/ahead=%s", backlog, ahead), func(t *testing.T) {
				rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{miniredis.RunT(t).Addr()}, DisableCache: true})
				require.NoError(t, err)
				t.Cleanup(rc.Close)
				clock := clockwork.NewFakeClock()
				_, shard := newQueue(t, rc, osqueue.WithClock(clock), osqueue.WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return backlog }))
				fn, acct, env := uuid.New(), uuid.New(), uuid.New()
				at := clock.Now().Add(ahead)
				item, err := shard.EnqueueItem(t.Context(), osqueue.QueueItem{ID: "hint", FunctionID: fn, WorkspaceID: env,
					Data: osqueue.Item{Kind: osqueue.KindStart, WorkspaceID: env, Identifier: state.Identifier{WorkflowID: fn, AccountID: acct, WorkspaceID: env, RunID: ulid.Make()}}}, at, osqueue.EnqueueOpts{})
				require.NoError(t, err)
				lease, err := shard.Lease(t.Context(), item, time.Minute, clock.Now())
				require.NoError(t, err)
				require.Equal(t, ulid.Timestamp(clock.Now().Add(time.Minute)), lease.Time())
				stored, err := shard.LoadQueueItem(t.Context(), item.ID)
				require.NoError(t, err)
				require.Equal(t, at.UnixMilli(), stored.AtMS)
				// Actual now, not the lookahead horizon, decides whether the lease is active.
				_, err = shard.Lease(t.Context(), item, time.Minute, clock.Now())
				require.ErrorIs(t, err, osqueue.ErrQueueItemAlreadyLeased)
			})
		}
	}
}

// Suspend discovery to prove that backlog hints lease directly without refill.
// The ordinary worker and completion path remain real.
type hintOnlyRedisShard struct {
	RedisQueueShard
}

func (s hintOnlyRedisShard) Run(ctx context.Context, _ osqueue.QueueScannerRuntime) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestItemHintBacklogHandoff(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		name := "direct backlog lease"
		if blocked {
			name = "direct lease while refill ownership held"
		}
		t.Run(name, func(t *testing.T) {
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(rc.Close)
			opts := []osqueue.QueueOpt{osqueue.WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return true }),
				osqueue.WithRunMode(osqueue.QueueRunMode{Partition: true}), osqueue.WithPollTick(5 * time.Millisecond), osqueue.WithNumWorkers(2)}
			base := NewQueueShard("ss3", NewQueueClient(rc, "{hints}"), opts...)
			wrapper := hintOnlyRedisShard{RedisQueueShard: base}
			reg, err := osqueue.NewSingleShardRegistry(wrapper)
			require.NoError(t, err)
			offers := make(chan func(osqueue.QueueItem) bool, 1)
			opts = append(opts, osqueue.WithItemHints(osqueue.ItemHintOptions{BufferSize: 2, MaxActive: 1, AttemptTimeout: time.Second, Source: func(ctx context.Context, _ osqueue.QueueShard, offer func(osqueue.QueueItem) bool) error {
				offers <- offer
				<-ctx.Done()
				return nil
			}}))
			proc, err := osqueue.New(t.Context(), "hints", reg, opts...)
			require.NoError(t, err)
			job := "start"
			acct, env, fn := uuid.New(), uuid.New(), uuid.New()
			item := osqueue.Item{Kind: osqueue.KindStart, JobID: &job, WorkspaceID: env, Identifier: state.Identifier{AccountID: acct, WorkspaceID: env, WorkflowID: fn, RunID: ulid.Make()}}
			var stored osqueue.QueueItem
			require.NoError(t, proc.Enqueue(t.Context(), item, time.Now(), osqueue.EnqueueOpts{OnEnqueued: func(qi osqueue.QueueItem, shard string) { stored = qi }}))
			id := stored.ID
			backlog := osqueue.ItemBacklog(t.Context(), stored)
			shadow := osqueue.ItemShadowPartition(t.Context(), stored)
			if blocked {
				_, err = base.ShadowPartitionLease(t.Context(), &shadow, time.Minute)
				require.NoError(t, err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			executed := make(chan struct{}, 1)
			go func() {
				done <- proc.Run(ctx, func(context.Context, osqueue.RunInfo, osqueue.Item) (osqueue.RunResult, error) {
					executed <- struct{}{}
					return osqueue.RunResult{}, nil
				})
			}()
			defer func() {
				cancel()
				select {
				case <-done:
				case <-time.After(2 * time.Second):
					t.Error("processor did not stop")
				}
			}()
			var offer func(osqueue.QueueItem) bool
			select {
			case offer = <-offers:
			case <-time.After(time.Second):
				t.Fatal("hint receiver not ready")
			}
			require.True(t, offer(stored))
			select {
			case <-executed:
			case <-time.After(2 * time.Second):
				t.Fatal("backlog hint did not dispatch")
			}
			require.Eventually(t, func() bool { _, err := base.LoadQueueItem(t.Context(), id); return err == osqueue.ErrQueueItemNotFound }, time.Second, time.Millisecond)
			count, err := base.BacklogSize(t.Context(), backlog.BacklogID)
			require.NoError(t, err)
			require.Zero(t, count)
		})
	}
}

func TestItemHintBacklogPreservesCapacity(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	cm, err := constraintapi.NewRedisCapacityManager(constraintapi.WithClient(rc), constraintapi.WithShardName("hints"))
	require.NoError(t, err)
	opts := []osqueue.QueueOpt{
		osqueue.WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return true }),
		osqueue.WithCapacityManager(cm), osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
		osqueue.WithPartitionConstraintConfigGetter(func(context.Context, osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
			return osqueue.PartitionConstraintConfig{FunctionVersion: 1, Concurrency: osqueue.PartitionConcurrency{AccountConcurrency: 1, FunctionConcurrency: 1}}
		}),
		osqueue.WithRunMode(osqueue.QueueRunMode{Partition: true}), osqueue.WithPollTick(5 * time.Millisecond), osqueue.WithNumWorkers(2),
	}
	base := NewQueueShard("ss3", NewQueueClient(rc, "{capacity-hints}"), opts...)
	reg, err := osqueue.NewSingleShardRegistry(hintOnlyRedisShard{base})
	require.NoError(t, err)
	offers := make(chan func(osqueue.QueueItem) bool, 1)
	opts = append(opts, osqueue.WithItemHints(osqueue.ItemHintOptions{BufferSize: 4, MaxActive: 2, AttemptTimeout: time.Second, Source: func(ctx context.Context, _ osqueue.QueueShard, offer func(osqueue.QueueItem) bool) error {
		offers <- offer
		<-ctx.Done()
		return nil
	}}))
	proc, err := osqueue.New(t.Context(), "hints", reg, opts...)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	entered := make(chan string, 2)
	release := make(chan struct{})
	go func() {
		done <- proc.Run(ctx, func(_ context.Context, _ osqueue.RunInfo, item osqueue.Item) (osqueue.RunResult, error) {
			entered <- *item.JobID
			<-release
			return osqueue.RunResult{}, nil
		})
	}()
	defer func() {
		close(release)
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("processor did not stop")
		}
	}()
	offer := <-offers
	acct, env, fn := uuid.New(), uuid.New(), uuid.New()
	enqueue := func(job string) osqueue.QueueItem {
		item := osqueue.Item{Kind: osqueue.KindStart, JobID: &job, WorkspaceID: env, Identifier: state.Identifier{AccountID: acct, WorkspaceID: env, WorkflowID: fn, RunID: ulid.Make()}}
		var enqueued osqueue.QueueItem
		require.NoError(t, proc.Enqueue(t.Context(), item, time.Now(), osqueue.EnqueueOpts{OnEnqueued: func(qi osqueue.QueueItem, shard string) { enqueued = qi }}))
		return enqueued
	}
	first := enqueue("first")
	require.True(t, offer(first))
	select {
	case id := <-entered:
		require.Equal(t, first.ID, id)
	case <-time.After(2 * time.Second):
		t.Fatal("first hint did not dispatch")
	}
	second := enqueue("second")
	require.True(t, offer(second))
	select {
	case <-entered:
		t.Fatal("hint bypassed function/account concurrency")
	case <-time.After(100 * time.Millisecond):
	}
	stored, err := base.LoadQueueItem(t.Context(), second.ID)
	require.NoError(t, err)
	require.Nil(t, stored.LeaseID)
	require.Empty(t, stored.CapacityLease)
	backlog := osqueue.ItemBacklog(t.Context(), *stored)
	count, err := base.BacklogSize(t.Context(), backlog.BacklogID)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
}

// Ordinary refill can encounter a directly leased backlog item. It must not
// execute it twice, and its extra reservation must expire without renewal.
func TestItemHintBacklogRefillReservationCleanup(t *testing.T) {
	for _, refill := range []bool{false, true} {
		name := "without-refill"
		if refill {
			name = "ordinary-refill-races-direct-lease"
		}
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(rc.Close)
			clock := clockwork.NewFakeClock()
			cm, err := constraintapi.NewRedisCapacityManager(constraintapi.WithClient(rc), constraintapi.WithShardName("hint-refill"), constraintapi.WithClock(clock))
			require.NoError(t, err)
			constraints := osqueue.PartitionConstraintConfig{FunctionVersion: 1, Concurrency: osqueue.PartitionConcurrency{AccountConcurrency: 3, FunctionConcurrency: 3}}
			proc, shard := newQueue(t, rc, osqueue.WithClock(clock), osqueue.WithCapacityManager(cm), osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
				osqueue.WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return true }),
				osqueue.WithPartitionConstraintConfigGetter(func(context.Context, osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					return constraints
				}))
			acct, env, fn := uuid.New(), uuid.New(), uuid.New()
			item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{ID: "direct-backlog", FunctionID: fn, WorkspaceID: env,
				Data: osqueue.Item{Kind: osqueue.KindStart, WorkspaceID: env, Identifier: state.Identifier{AccountID: acct, WorkspaceID: env, WorkflowID: fn, RunID: ulid.Make()}}}, clock.Now(), osqueue.EnqueueOpts{})
			require.NoError(t, err)
			var work osqueue.ProcessItem
			leased, err := proc.(osqueue.QueueItemLeaser).LeaseItem(ctx, osqueue.LeaseItemRequest{Item: &item, StaticTime: clock.Now()}, func(_ context.Context, i osqueue.ProcessItem) (osqueue.DispatchedItem, error) {
				work = i
				return osqueue.NewCompletedDispatchedItem(osqueue.DispatchedItemResult{}), nil
			})
			require.NoError(t, err)
			require.Equal(t, osqueue.LeaseItemStatusDispatched, leased.Status)
			require.NotNil(t, work.CapacityLease)
			defer proc.Semaphore().Release(1)
			capacity := func() int {
				res, userErr, internalErr := cm.Check(ctx, &constraintapi.CapacityCheckRequest{AccountID: acct, EnvID: env, FunctionID: fn, Configuration: osqueue.ConstraintConfigFromConstraints(constraints), Constraints: []constraintapi.ConstraintItem{
					{Kind: constraintapi.ConstraintKindConcurrency, Concurrency: &constraintapi.ConcurrencyConstraint{Scope: enums.ConcurrencyScopeAccount, Mode: enums.ConcurrencyModeStep}},
					{Kind: constraintapi.ConstraintKindConcurrency, Concurrency: &constraintapi.ConcurrencyConstraint{Scope: enums.ConcurrencyScopeFn, Mode: enums.ConcurrencyModeStep}},
				}})
				require.NoError(t, userErr)
				require.NoError(t, internalErr)
				return res.AvailableCapacity
			}
			require.Equal(t, 2, capacity())
			shadow := osqueue.ItemShadowPartition(ctx, item)
			backlog := osqueue.ItemBacklog(ctx, item)
			if refill {
				require.NoError(t, proc.ProcessShadowPartition(ctx, &shadow, 0))
				refilled, err := shard.LoadQueueItem(ctx, item.ID)
				require.NoError(t, err)
				require.NotNil(t, refilled.CapacityLease)
				require.NotEqual(t, work.CapacityLease.LeaseID, refilled.CapacityLease.LeaseID)
				require.Equal(t, work.I.LeaseID, refilled.LeaseID, "refill must preserve the execution lease")
				require.Equal(t, 1, capacity(), "refill temporarily holds one additional slot")
				duplicate, err := proc.(osqueue.QueueItemLeaser).LeaseItem(ctx, osqueue.LeaseItemRequest{Item: refilled, StaticTime: clock.Now()}, func(context.Context, osqueue.ProcessItem) (osqueue.DispatchedItem, error) {
					t.Fatal("refill dispatched the already-leased item")
					return nil, nil
				})
				require.NoError(t, err)
				require.Equal(t, osqueue.LeaseItemStatusAlreadyLeased, duplicate.Status)
			}
			executions := 0
			_, err = proc.ProcessItem(ctx, work, func(context.Context, osqueue.RunInfo, osqueue.Item) (osqueue.RunResult, error) {
				executions++
				return osqueue.RunResult{}, nil
			})
			require.NoError(t, err)
			require.Equal(t, 1, executions)
			_, err = shard.LoadQueueItem(ctx, item.ID)
			require.ErrorIs(t, err, osqueue.ErrQueueItemNotFound)
			count, err := shard.BacklogSize(ctx, backlog.BacklogID)
			require.NoError(t, err)
			require.Zero(t, count)
			readyKey := shadowPartitionReadyQueueKey(shadow, shard.(*queue).RedisClient.kg)
			count, err = rc.Do(ctx, rc.B().Zcard().Key(readyKey).Build()).ToInt64()
			require.NoError(t, err)
			require.Zero(t, count)
			remaining := 3
			if refill {
				remaining = 2
			}
			require.Eventually(t, func() bool { return capacity() == remaining }, time.Second, time.Millisecond, "worker releases its own reservation")
			clock.Advance(osqueue.QueueLeaseDuration + time.Second)
			r.FastForward(osqueue.QueueLeaseDuration + time.Second)
			scavenged, internalErr := cm.Scavenge(ctx)
			require.NoError(t, internalErr)
			if refill {
				require.Equal(t, 1, scavenged.ReclaimedLeases)
			}
			require.Equal(t, 3, capacity(), "expired refill reservation must be reclaimed")
		})
	}
}

// The buffered snapshot is never reloaded. Only the normal atomic lease may
// decide whether an item still exists, is unleased and has the same generation.
type observedHintRedisShard struct {
	hintOnlyRedisShard
	attempts chan error
}

func (s observedHintRedisShard) LoadQueueItem(context.Context, string) (*osqueue.QueueItem, error) {
	panic("hint processor must not reconstruct the item with a fetch")
}
func (s observedHintRedisShard) Lease(ctx context.Context, item osqueue.QueueItem, duration time.Duration, now time.Time, opts ...osqueue.LeaseOptionFn) (*ulid.ULID, error) {
	lease, err := s.RedisQueueShard.Lease(ctx, item, duration, now, opts...)
	s.attempts <- err
	return lease, err
}

func TestItemHintDiscardStaleBufferedSnapshot(t *testing.T) {
	for _, backlog := range []bool{false, true} {
		for _, name := range []string{"missing", "ordinary-leased", "requeued"} {
			t.Run(fmt.Sprintf("backlog=%t/%s", backlog, name), func(t *testing.T) {
				rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{miniredis.RunT(t).Addr()}, DisableCache: true})
				require.NoError(t, err)
				t.Cleanup(rc.Close)
				clock := clockwork.NewFakeClock()
				opts := []osqueue.QueueOpt{osqueue.WithClock(clock), osqueue.WithPollTick(time.Second), osqueue.WithNumWorkers(2), osqueue.WithRunMode(osqueue.QueueRunMode{Partition: true}), osqueue.WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return backlog })}
				base := NewQueueShard("ss3", NewQueueClient(rc, "{stale-hint}"), opts...)
				wrapper := observedHintRedisShard{hintOnlyRedisShard: hintOnlyRedisShard{base}, attempts: make(chan error, 4)}
				reg, err := osqueue.NewSingleShardRegistry(wrapper)
				require.NoError(t, err)
				offers := make(chan func(osqueue.QueueItem) bool, 1)
				opts = append(opts, osqueue.WithItemHints(osqueue.ItemHintOptions{BufferSize: 2, MaxActive: 1, AttemptTimeout: time.Second, Source: func(ctx context.Context, _ osqueue.QueueShard, offer func(osqueue.QueueItem) bool) error {
					offers <- offer
					<-ctx.Done()
					return nil
				}}))
				proc, err := osqueue.New(t.Context(), "stale-hint", reg, opts...)
				require.NoError(t, err)
				acct, env, fn := uuid.New(), uuid.New(), uuid.New()
				job := "hint"
				var stored osqueue.QueueItem
				require.NoError(t, proc.Enqueue(t.Context(), osqueue.Item{JobID: &job, Kind: osqueue.KindStart, WorkspaceID: env, Identifier: state.Identifier{AccountID: acct, WorkspaceID: env, WorkflowID: fn, RunID: ulid.Make()}}, clock.Now(), osqueue.EnqueueOpts{OnEnqueued: func(qi osqueue.QueueItem, _ string) { stored = qi }}))
				require.NotZero(t, stored.GenerationID)
				require.NotEmpty(t, stored.ID)
				ctx, cancel := context.WithCancel(t.Context())
				done := make(chan error, 1)
				var dispatches atomic.Int32
				go func() {
					done <- proc.Run(ctx, func(context.Context, osqueue.RunInfo, osqueue.Item) (osqueue.RunResult, error) {
						dispatches.Add(1)
						return osqueue.RunResult{}, nil
					})
				}()
				defer func() {
					cancel()
					select {
					case <-done:
					case <-time.After(3 * time.Second):
						t.Error("processor did not stop")
					}
				}()
				offer := <-offers
				require.True(t, offer(stored))
				var want error
				switch name {
				case "missing":
					require.NoError(t, base.Dequeue(t.Context(), stored))
					want = osqueue.ErrQueueItemNotFound
				case "ordinary-leased":
					_, err = base.Lease(t.Context(), stored, time.Minute, clock.Now())
					require.NoError(t, err)
					want = osqueue.ErrQueueItemAlreadyLeased
				case "requeued":
					require.NoError(t, base.Requeue(t.Context(), stored, clock.Now()))
					want = osqueue.ErrQueueItemNotFound
				}
				clock.Advance(time.Second)
				select {
				case err := <-wrapper.attempts:
					require.ErrorIs(t, err, want)
				case <-time.After(2 * time.Second):
					t.Fatal("hint lease not attempted")
				}
				clock.Advance(2 * time.Second)
				select {
				case <-wrapper.attempts:
					t.Fatal("hint retried")
				case <-time.After(20 * time.Millisecond):
				}
				require.Zero(t, dispatches.Load())
				if name == "missing" {
					_, err = base.LoadQueueItem(t.Context(), stored.ID)
					require.ErrorIs(t, err, osqueue.ErrQueueItemNotFound)
				}
			})
		}
	}
}
