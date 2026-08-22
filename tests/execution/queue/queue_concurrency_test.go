package queue

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/jonboulle/clockwork"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/service"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestQueuePartitionConcurrency(t *testing.T) {
	r := miniredis.RunT(t)

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	limit_1 := uuid.New()
	limit_10 := uuid.New()
	workflowIDs := []uuid.UUID{limit_1, limit_10}

	// Limit function concurrency by workflow ID.
	pkf := func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
		switch p.FunctionID {
		case limit_1:
			return osqueue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: osqueue.PartitionConcurrency{
					AccountConcurrency:  osqueue.NoConcurrencyLimit,
					FunctionConcurrency: 1,
				},
			}
		case limit_10:
			return osqueue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: osqueue.PartitionConcurrency{
					AccountConcurrency:  osqueue.NoConcurrencyLimit,
					FunctionConcurrency: 10,
				},
			}
		default:
			// No concurrency, which means use the default concurrency limits.
			return osqueue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: osqueue.PartitionConcurrency{
					AccountConcurrency:  osqueue.NoConcurrencyLimit,
					FunctionConcurrency: osqueue.NoConcurrencyLimit,
				},
			}
		}
	}

	// Create a new lifecycle listener.  This should be invoked each time we hit limits.
	ll := newTestLifecycleListener()

	clock := clockwork.NewRealClock()
	opts := []osqueue.QueueOpt{
		osqueue.WithNumWorkers(100),
		osqueue.WithPartitionConstraintConfigGetter(pkf),
		osqueue.WithQueueLifecycles(ll),
		osqueue.WithClock(clock),
	}

	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClient(rc),
		constraintapi.WithShardName("test"),
		constraintapi.WithClock(clock),
		constraintapi.WithEnableDebugLogs(true),
	)
	require.NoError(t, err)

	opts = append(opts, osqueue.WithCapacityManager(cm))
	opts = append(opts,
		osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
	)

	shard1 := redis_state.NewQueueShard(consts.DefaultQueueShardName, redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey), opts...)

	shardRegistry, err := osqueue.NewSingleShardRegistry(shard1)
	require.NoError(t, err)
	q, err := osqueue.New(
		ctx,
		"test-queue",
		shardRegistry,
		opts...,
	)
	require.NoError(t, err)

	var (
		counter_1   int32
		counter_10  int32
		jobDuration = 2 * time.Second
	)

	// Run the queue.
	go func() {
		_ = q.Run(ctx, func(ctx context.Context, _ osqueue.RunInfo, item osqueue.Item) (osqueue.RunResult, error) {
			if item.Identifier.WorkflowID == limit_1 {
				fmt.Println("Single concurrency item hit", time.Now().Truncate(time.Millisecond))
			}

			<-time.After(jobDuration / 2)
			// each job takes 2 seconds to complete.
			switch item.Identifier.WorkflowID {
			case limit_1:
				atomic.AddInt32(&counter_1, 1)
			case limit_10:
				fmt.Println("10 concurrency item hit", time.Now().Truncate(time.Millisecond))
				atomic.AddInt32(&counter_10, 1)
			}

			<-time.After(jobDuration / 2)

			if item.Identifier.WorkflowID == limit_1 {
				fmt.Println("Single concurrency item done", time.Now().Truncate(time.Millisecond))
			}
			return osqueue.RunResult{}, nil
		})
	}()

	at := time.Now().Add(time.Second).Truncate(time.Second)

	accountID := uuid.New()
	envID := uuid.New()

	// Schedule 10 jobs;  it should take 20 seconds for limit_1 to finish,
	// and 2 seconds for limit_10 to finish, given each job takes 2 seconds.
	start := time.Now()
	for i := 0; i < 10; i++ {
		for _, id := range workflowIDs {
			err := q.Enqueue(ctx, osqueue.Item{
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: envID,
					WorkflowID:  id,
					RunID:       ulid.MustNew(ulid.Now(), rand.Reader),
				},
				WorkspaceID: envID,
			}, at, osqueue.EnqueueOpts{})
			require.NoError(t, err)
		}
	}

	<-time.After(jobDuration)

	require.EqualValues(t, 10, atomic.LoadInt32(&counter_10), "Should have hit all 10 items with a concurrency limit of 10")
	require.EqualValues(t, int32(1), atomic.LoadInt32(&counter_1), "Should have only run a single job")

	// TODO: Assert that the counterPartitionConcurrencyLimitReached counter isn't crazy high - we
	// don't want to be churning on the partition.
	for i := 0; i <= 100; i++ {
		<-time.After(500 * time.Millisecond)
		if atomic.LoadInt32(&counter_1) == 10 {
			break
		}
	}

	require.Eventually(t, func() bool {
		ll.lock.Lock()
		defer ll.lock.Unlock()
		return ll.fnConcurrency[limit_1] > 0
	}, 5*time.Second, 50*time.Millisecond)

	diff := time.Since(start).Seconds()
	require.Greater(t, int(diff), 10, "10 jobs should have taken at least 10 seconds")
	require.Less(t, int(diff), 40, "10 jobs should have taken fewer than 40 seconds") // an extra 2x latency due to race checker
}

func TestKeyQueueConcurrencyOneDoesNotOverlapSteps(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))
	r.SetTime(clock.Now())

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
	constraints := osqueue.PartitionConstraintConfig{
		FunctionVersion: 1,
		Concurrency: osqueue.PartitionConcurrency{
			AccountConcurrency:  osqueue.NoConcurrencyLimit,
			FunctionConcurrency: 1,
		},
	}
	options := []osqueue.QueueOpt{
		osqueue.WithClock(clock),
		osqueue.WithNumWorkers(2),
		osqueue.WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool {
			return true
		}),
		osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
		osqueue.WithCapacityLeaseExtendInterval(osqueue.QueueLeaseDuration / 2),
		osqueue.WithPartitionConstraintConfigGetter(func(context.Context, osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
			return constraints
		}),
	}

	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClient(rc),
		constraintapi.WithShardName("test"),
		constraintapi.WithClock(clock),
	)
	require.NoError(t, err)
	options = append(options, osqueue.WithCapacityManager(cm))

	shard := redis_state.NewQueueShard(
		"test",
		redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey),
		options...,
	)
	registry, err := osqueue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := osqueue.New(ctx, "test", registry, options...)
	require.NoError(t, err)

	makeItem := func(jobID string) osqueue.QueueItem {
		return osqueue.QueueItem{
			FunctionID:  fnID,
			WorkspaceID: envID,
			Data: osqueue.Item{
				JobID:       &jobID,
				WorkspaceID: envID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: envID,
					WorkflowID:  fnID,
					RunID:       ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader),
				},
			},
		}
	}

	first, err := shard.EnqueueItem(ctx, makeItem("first"), clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)
	_, err = shard.EnqueueItem(ctx, makeItem("second"), clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	backlog := osqueue.ItemBacklog(ctx, first)
	shadowPartition := osqueue.ItemShadowPartition(ctx, first)
	refillUntil := clock.Now().Add(time.Second)
	refill, _, err := q.ProcessShadowPartitionBacklog(ctx, &shadowPartition, &backlog, refillUntil, constraints)
	require.NoError(t, err)
	require.Len(t, refill.RefilledItems, 1, "concurrency 1 should only refill one item")

	// Leave the refilled capacity lease with less time remaining than one full
	// renewal interval. Without a synchronous refresh before dispatch,
	// ProcessItem would not attempt its first renewal until after expiry.
	clock.Advance(osqueue.QueueLeaseDuration - 3*time.Second)
	r.FastForward(osqueue.QueueLeaseDuration - 3*time.Second)
	r.SetTime(clock.Now())
	refilledItem, err := shard.LoadQueueItem(ctx, refill.RefilledItems[0])
	require.NoError(t, err)
	require.NotNil(t, refilledItem.CapacityLease)
	require.Equal(t, 3*time.Second, refilledItem.CapacityLease.LeaseID.Timestamp().Sub(clock.Now()))

	partition := osqueue.ItemPartition(ctx, first)
	dispatchReady := func() *osqueue.ProcessItem {
		items, err := shard.Peek(ctx, &partition, clock.Now().Add(time.Second), 10)
		require.NoError(t, err)
		if len(items) == 0 {
			return nil
		}

		iterator := osqueue.ProcessorIterator{
			Partition:  &partition,
			Items:      items,
			Queue:      q,
			Dispatch:   dispatchToOSQueueWorkers(q.Workers()),
			StaticTime: clock.Now(),
		}
		require.NoError(t, iterator.Iterate(ctx))

		select {
		case work := <-q.Workers():
			return &work
		default:
			return nil
		}
	}

	firstWork := dispatchReady()
	require.NotNil(t, firstWork)

	var active, maxActive atomic.Int32
	started := make(chan struct{}, 2)
	finish := make(chan struct{})
	var finishOnce sync.Once
	finishWork := func() {
		finishOnce.Do(func() { close(finish) })
	}
	defer finishWork()
	process := func(work osqueue.ProcessItem) <-chan error {
		done := make(chan error, 1)
		go func() {
			_, err := q.ProcessItem(ctx, work, func(context.Context, osqueue.RunInfo, osqueue.Item) (osqueue.RunResult, error) {
				running := active.Add(1)
				for {
					observed := maxActive.Load()
					if running <= observed || maxActive.CompareAndSwap(observed, running) {
						break
					}
				}
				started <- struct{}{}
				<-finish
				active.Add(-1)
				return osqueue.RunResult{}, nil
			})
			q.Semaphore().Release(1)
			done <- err
		}()
		return done
	}

	firstDone := process(*firstWork)
	select {
	case <-started:
	case <-time.After(time.Second):
		require.FailNow(t, "first item did not start")
	}

	// Advance past the original lease's expiry and scavenge it. The refreshed
	// lease must continue holding the concurrency slot while the first callback
	// is running.
	clock.Advance(4 * time.Second)
	r.FastForward(4 * time.Second)
	r.SetTime(clock.Now())
	_, err = cm.Scavenge(ctx)
	require.NoError(t, err)

	_, _, err = q.ProcessShadowPartitionBacklog(ctx, &shadowPartition, &backlog, clock.Now().Add(time.Second), constraints)
	require.NoError(t, err)
	secondWork := dispatchReady()
	var secondDone <-chan error
	if secondWork != nil {
		secondDone = process(*secondWork)
		select {
		case <-started:
		case <-time.After(time.Second):
			require.FailNow(t, "second item was dispatched but did not start")
		}
	}

	finishWork()
	require.NoError(t, <-firstDone)
	if secondDone != nil {
		require.NoError(t, <-secondDone)
	}
	service.Wait()

	require.LessOrEqual(t, maxActive.Load(), int32(1), "steps sharing concurrency 1 must never overlap")
}

type testLifecycleListener struct {
	lock            *sync.Mutex
	fnConcurrency   map[uuid.UUID]int
	acctConcurrency map[uuid.UUID]int
	ckConcurrency   map[string]int
}

func newTestLifecycleListener() testLifecycleListener {
	return testLifecycleListener{
		lock:            &sync.Mutex{},
		fnConcurrency:   map[uuid.UUID]int{},
		acctConcurrency: map[uuid.UUID]int{},
		ckConcurrency:   map[string]int{},
	}
}

func (t testLifecycleListener) OnFnConcurrencyLimitReached(_ context.Context, fnID uuid.UUID) {
	t.lock.Lock()
	defer t.lock.Unlock()

	i := t.fnConcurrency[fnID]
	t.fnConcurrency[fnID] = i + 1
}

func (t testLifecycleListener) OnAccountConcurrencyLimitReached(
	_ context.Context,
	acctID uuid.UUID,
	workspaceID *uuid.UUID,
) {
	t.lock.Lock()
	defer t.lock.Unlock()

	i := t.acctConcurrency[acctID]
	t.acctConcurrency[acctID] = i + 1
}

func (t testLifecycleListener) OnCustomKeyConcurrencyLimitReached(_ context.Context, key string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	i := t.ckConcurrency[key]
	t.ckConcurrency[key] = i + 1
}

func (t testLifecycleListener) OnBacklogRefillConstraintHit(ctx context.Context, p *osqueue.QueueShadowPartition, b *osqueue.QueueBacklog, res *osqueue.BacklogRefillResult) {
	// no-op
}

func (t testLifecycleListener) OnBacklogRefilled(ctx context.Context, p *osqueue.QueueShadowPartition, b *osqueue.QueueBacklog, res *osqueue.BacklogRefillResult) {
	// no-op
}
