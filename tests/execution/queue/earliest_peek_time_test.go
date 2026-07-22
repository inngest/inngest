package queue

import (
	"context"
	"crypto/rand"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func dispatchToOSQueueWorkers(workers chan osqueue.ProcessItem) osqueue.DispatchFunc {
	return func(_ context.Context, item osqueue.ProcessItem) error {
		workers <- item
		return nil
	}
}

func TestProcessorIteratorSetsEarliestPeekTimeBeforeConstraintLimit(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClockAt(time.Date(2026, 6, 11, 10, 0, 0, 123_000_000, time.UTC))
	r.SetTime(clock.Now())

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	cmLifecycles := constraintapi.NewConstraintAPIDebugLifecycles()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClient(rc),
		constraintapi.WithShardName("test"),
		constraintapi.WithClock(clock),
		constraintapi.WithLifecycles(cmLifecycles),
	)
	require.NoError(t, err)

	options := []osqueue.QueueOpt{
		osqueue.WithClock(clock),
		osqueue.WithCapacityManager(cm),
		osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
		osqueue.WithQueueItemEarliestPeekTimeEnabled(func(ctx context.Context, shardName string, acctID, gotEnvID, gotFnID uuid.UUID) osqueue.QueueItemEarliestPeekTimeConfig {
			if shardName == "test" && acctID == accountID && gotEnvID == envID && gotFnID == fnID {
				return osqueue.QueueItemEarliestPeekTimeConfig{
					Enabled:        true,
					BulkStampLimit: 100,
				}
			}
			return osqueue.QueueItemEarliestPeekTimeConfig{}
		}),
		osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
			return osqueue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: osqueue.PartitionConcurrency{
					AccountConcurrency:  1,
					FunctionConcurrency: 1,
				},
			}
		}),
	}

	queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shard := redis_state.NewQueueShard("test", queueClient, options...)

	shardRegistry, err := osqueue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := osqueue.New(ctx, "test", shardRegistry, options...)
	require.NoError(t, err)

	makeItem := func() osqueue.QueueItem {
		return osqueue.QueueItem{
			FunctionID:  fnID,
			WorkspaceID: envID,
			Data: osqueue.Item{
				Kind: osqueue.KindStart,
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: envID,
					WorkflowID:  fnID,
					RunID:       ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader),
				},
				WorkspaceID: envID,
			},
		}
	}

	qi1, err := shard.EnqueueItem(ctx, makeItem(), clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)
	qi2, err := shard.EnqueueItem(ctx, makeItem(), clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	partition := osqueue.ItemPartition(ctx, qi1)
	peekTime := clock.Now().Add(250 * time.Millisecond)
	iter := osqueue.ProcessorIterator{
		Partition:  &partition,
		Items:      []*osqueue.QueueItem{&qi1, &qi2},
		Queue:      q,
		Dispatch:   dispatchToOSQueueWorkers(q.Workers()),
		StaticTime: peekTime,
	}

	err = iter.Iterate(ctx)
	require.NoError(t, err)

	require.Len(t, cmLifecycles.AcquireCalls, 2)
	require.Len(t, cmLifecycles.AcquireCalls[0].GrantedLeases, 1)
	require.Len(t, cmLifecycles.AcquireCalls[1].GrantedLeases, 0)
	require.Len(t, cmLifecycles.AcquireCalls[1].LimitingConstraints, 1)

	require.Equal(t, peekTime.UnixMilli(), qi1.EarliestPeekTime)
	require.Equal(t, peekTime.UnixMilli(), qi2.EarliestPeekTime)

	key := queueClient.KeyGenerator().QueueItemEarliestPeekTime(qi2.ID)
	val, err := r.Get(key)
	require.NoError(t, err)
	require.Equal(t, strconv.FormatInt(peekTime.UnixMilli(), 10), val)

	loaded, err := shard.LoadQueueItem(ctx, qi1.ID)
	require.NoError(t, err)
	require.Equal(t, peekTime.UnixMilli(), loaded.EarliestPeekTime, "leased item should persist stamped earliest peek time")

	loaded, err = shard.LoadQueueItem(ctx, qi2.ID)
	require.NoError(t, err)
	require.Zero(t, loaded.EarliestPeekTime, "unleased stamped item should still live in the side key only")
}

func TestLeasePersistsStampedEarliestPeekTimeWhenFeatureFlagEnabled(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClockAt(time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC))
	r.SetTime(clock.Now())

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shard := redis_state.NewQueueShard(
		"test",
		queueClient,
		osqueue.WithClock(clock),
		osqueue.WithQueueItemEarliestPeekTimeEnabled(func(ctx context.Context, shardName string, acctID, gotEnvID, gotFnID uuid.UUID) osqueue.QueueItemEarliestPeekTimeConfig {
			if shardName == "test" && acctID == accountID && gotEnvID == envID && gotFnID == fnID {
				return osqueue.QueueItemEarliestPeekTimeConfig{Enabled: true}
			}
			return osqueue.QueueItemEarliestPeekTimeConfig{}
		}),
	)

	qi, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: envID,
		Data: osqueue.Item{
			Kind: osqueue.KindStart,
			Identifier: state.Identifier{
				AccountID:   accountID,
				WorkspaceID: envID,
				WorkflowID:  fnID,
				RunID:       ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader),
			},
			WorkspaceID: envID,
		},
	}, clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)
	require.Zero(t, qi.EarliestPeekTime)

	peekTime := clock.Now().Add(250 * time.Millisecond)
	stampedAt, err := shard.SetEarliestPeekTime(ctx, qi, peekTime)
	require.NoError(t, err)
	require.Equal(t, peekTime.UnixMilli(), stampedAt.UnixMilli())

	loaded, err := shard.LoadQueueItem(ctx, qi.ID)
	require.NoError(t, err)
	require.Zero(t, loaded.EarliestPeekTime)

	qi.EarliestPeekTime = stampedAt.UnixMilli()
	_, err = shard.Lease(ctx, qi, osqueue.QueueLeaseDuration, clock.Now().Add(time.Second))
	require.NoError(t, err)

	loaded, err = shard.LoadQueueItem(ctx, qi.ID)
	require.NoError(t, err)
	require.Equal(t, peekTime.UnixMilli(), loaded.EarliestPeekTime)

	key := queueClient.KeyGenerator().QueueItemEarliestPeekTime(qi.ID)
	val, err := r.Get(key)
	require.NoError(t, err)
	require.Equal(t, strconv.FormatInt(peekTime.UnixMilli(), 10), val)
}

func TestLeaseIgnoresStampedEarliestPeekTimeWhenFeatureFlagDisabled(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClockAt(time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC))
	r.SetTime(clock.Now())

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shard := redis_state.NewQueueShard("test", queueClient, osqueue.WithClock(clock))

	qi, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: envID,
		Data: osqueue.Item{
			Kind: osqueue.KindStart,
			Identifier: state.Identifier{
				AccountID:   accountID,
				WorkspaceID: envID,
				WorkflowID:  fnID,
				RunID:       ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader),
			},
			WorkspaceID: envID,
		},
	}, clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	stampedPeekTime := clock.Now().Add(-time.Minute)
	qi.EarliestPeekTime = stampedPeekTime.UnixMilli()

	leaseTime := clock.Now().Add(250 * time.Millisecond)
	_, err = shard.Lease(ctx, qi, osqueue.QueueLeaseDuration, leaseTime)
	require.NoError(t, err)

	loaded, err := shard.LoadQueueItem(ctx, qi.ID)
	require.NoError(t, err)
	require.Equal(t, leaseTime.UnixMilli(), loaded.EarliestPeekTime)
	require.NotEqual(t, stampedPeekTime.UnixMilli(), loaded.EarliestPeekTime)

	key := queueClient.KeyGenerator().QueueItemEarliestPeekTime(qi.ID)
	require.False(t, r.Exists(key), "legacy path should not create the side key")
}

func TestProcessorIteratorSetsEarliestPeekTimeForProcessedItemsBeforeConcurrencyStop(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClockAt(time.Date(2026, 6, 11, 10, 0, 0, 123_000_000, time.UTC))
	r.SetTime(clock.Now())

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	cmLifecycles := constraintapi.NewConstraintAPIDebugLifecycles()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClient(rc),
		constraintapi.WithShardName("test"),
		constraintapi.WithClock(clock),
		constraintapi.WithLifecycles(cmLifecycles),
	)
	require.NoError(t, err)

	const bulkStampLimit = 5
	options := []osqueue.QueueOpt{
		osqueue.WithClock(clock),
		osqueue.WithCapacityManager(cm),
		osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
		osqueue.WithQueueItemEarliestPeekTimeEnabled(func(ctx context.Context, shardName string, acctID, gotEnvID, gotFnID uuid.UUID) osqueue.QueueItemEarliestPeekTimeConfig {
			if shardName == "test" && acctID == accountID && gotEnvID == envID && gotFnID == fnID {
				return osqueue.QueueItemEarliestPeekTimeConfig{
					Enabled:        true,
					BulkStampLimit: bulkStampLimit,
				}
			}
			return osqueue.QueueItemEarliestPeekTimeConfig{}
		}),
		osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
			return osqueue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: osqueue.PartitionConcurrency{
					AccountConcurrency:  1,
					FunctionConcurrency: 1,
				},
			}
		}),
	}

	queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shard := redis_state.NewQueueShard("test", queueClient, options...)

	shardRegistry, err := osqueue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := osqueue.New(ctx, "test", shardRegistry, options...)
	require.NoError(t, err)

	makeItem := func() osqueue.QueueItem {
		return osqueue.QueueItem{
			FunctionID:  fnID,
			WorkspaceID: envID,
			Data: osqueue.Item{
				Kind: osqueue.KindStart,
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: envID,
					WorkflowID:  fnID,
					RunID:       ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader),
				},
				WorkspaceID: envID,
			},
		}
	}

	items := make([]*osqueue.QueueItem, 10)
	for i := range items {
		qi, err := shard.EnqueueItem(ctx, makeItem(), clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)
		items[i] = &qi
	}

	partition := osqueue.ItemPartition(ctx, *items[0])
	peekTime := clock.Now().Add(250 * time.Millisecond)
	iter := osqueue.ProcessorIterator{
		Partition:  &partition,
		Items:      items,
		Queue:      q,
		Dispatch:   dispatchToOSQueueWorkers(q.Workers()),
		StaticTime: peekTime,
	}

	err = iter.Iterate(ctx)
	require.NoError(t, err)

	require.Len(t, cmLifecycles.AcquireCalls, 2)
	require.Len(t, cmLifecycles.AcquireCalls[0].GrantedLeases, 1)
	require.Len(t, cmLifecycles.AcquireCalls[1].GrantedLeases, 0)
	require.Len(t, cmLifecycles.AcquireCalls[1].LimitingConstraints, 1)

	stampedItems := len(cmLifecycles.AcquireCalls) + bulkStampLimit
	if stampedItems > len(items) {
		stampedItems = len(items)
	}
	for i, qi := range items {
		key := queueClient.KeyGenerator().QueueItemEarliestPeekTime(qi.ID)
		if i < stampedItems {
			require.Equal(t, peekTime.UnixMilli(), qi.EarliestPeekTime, "item %d should be stamped", i)
			val, err := r.Get(key)
			require.NoError(t, err)
			require.Equal(t, strconv.FormatInt(peekTime.UnixMilli(), 10), val)
			continue
		}

		require.Zero(t, qi.EarliestPeekTime, "item %d should be beyond the cutoff", i)
		require.False(t, r.Exists(key), "item %d should not have a side key", i)
	}
}

func TestProcessorIteratorDoesNotStampRemainingItemsWhenDisabled(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClockAt(time.Date(2026, 6, 11, 10, 0, 0, 123_000_000, time.UTC))
	r.SetTime(clock.Now())

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	cmLifecycles := constraintapi.NewConstraintAPIDebugLifecycles()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClient(rc),
		constraintapi.WithShardName("test"),
		constraintapi.WithClock(clock),
		constraintapi.WithLifecycles(cmLifecycles),
	)
	require.NoError(t, err)

	const bulkStampLimit = 5
	options := []osqueue.QueueOpt{
		osqueue.WithClock(clock),
		osqueue.WithCapacityManager(cm),
		osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
		osqueue.WithQueueItemEarliestPeekTimeEnabled(func(ctx context.Context, shardName string, acctID, gotEnvID, gotFnID uuid.UUID) osqueue.QueueItemEarliestPeekTimeConfig {
			if shardName == "test" && acctID == accountID && gotEnvID == envID && gotFnID == fnID {
				return osqueue.QueueItemEarliestPeekTimeConfig{
					Enabled:        false,
					BulkStampLimit: bulkStampLimit,
				}
			}
			return osqueue.QueueItemEarliestPeekTimeConfig{}
		}),
		osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
			return osqueue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: osqueue.PartitionConcurrency{
					AccountConcurrency:  1,
					FunctionConcurrency: 1,
				},
			}
		}),
	}

	queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shard := redis_state.NewQueueShard("test", queueClient, options...)

	shardRegistry, err := osqueue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := osqueue.New(ctx, "test", shardRegistry, options...)
	require.NoError(t, err)

	makeItem := func() osqueue.QueueItem {
		return osqueue.QueueItem{
			FunctionID:  fnID,
			WorkspaceID: envID,
			Data: osqueue.Item{
				Kind: osqueue.KindStart,
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: envID,
					WorkflowID:  fnID,
					RunID:       ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader),
				},
				WorkspaceID: envID,
			},
		}
	}

	items := make([]*osqueue.QueueItem, 10)
	for i := range items {
		qi, err := shard.EnqueueItem(ctx, makeItem(), clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)
		items[i] = &qi
	}

	partition := osqueue.ItemPartition(ctx, *items[0])
	iter := osqueue.ProcessorIterator{
		Partition:  &partition,
		Items:      items,
		Queue:      q,
		Dispatch:   dispatchToOSQueueWorkers(q.Workers()),
		StaticTime: clock.Now().Add(250 * time.Millisecond),
	}

	err = iter.Iterate(ctx)
	require.NoError(t, err)

	require.Len(t, cmLifecycles.AcquireCalls, 2)
	require.Len(t, cmLifecycles.AcquireCalls[0].GrantedLeases, 1)
	require.Len(t, cmLifecycles.AcquireCalls[1].GrantedLeases, 0)

	for i, qi := range items {
		key := queueClient.KeyGenerator().QueueItemEarliestPeekTime(qi.ID)
		require.False(t, r.Exists(key), "item %d should not have a side key", i)
	}
}

func TestQueueLatencySeparatesSystemAndUserLatencyAfterEarliestPeekStamp(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	enqueuedAt := time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC)
	clock := clockwork.NewFakeClockAt(enqueuedAt)
	r.SetTime(clock.Now())

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClient(rc),
		constraintapi.WithShardName("test"),
		constraintapi.WithClock(clock),
	)
	require.NoError(t, err)

	functionVersion := 1
	concurrencyLimit := 1
	options := []osqueue.QueueOpt{
		osqueue.WithClock(clock),
		osqueue.WithCapacityManager(cm),
		osqueue.WithAcquireCapacityLeaseOnBacklogRefill(true),
		osqueue.WithQueueItemEarliestPeekTimeEnabled(func(ctx context.Context, shardName string, acctID, gotEnvID, gotFnID uuid.UUID) osqueue.QueueItemEarliestPeekTimeConfig {
			if shardName == "test" && acctID == accountID && gotEnvID == envID && gotFnID == fnID {
				return osqueue.QueueItemEarliestPeekTimeConfig{
					Enabled:        true,
					BulkStampLimit: 10,
				}
			}
			return osqueue.QueueItemEarliestPeekTimeConfig{}
		}),
		osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
			return osqueue.PartitionConstraintConfig{
				FunctionVersion: functionVersion,
				Concurrency: osqueue.PartitionConcurrency{
					AccountConcurrency:  concurrencyLimit,
					FunctionConcurrency: concurrencyLimit,
				},
			}
		}),
	}

	queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shard := redis_state.NewQueueShard("test", queueClient, options...)

	shardRegistry, err := osqueue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := osqueue.New(ctx, "test", shardRegistry, options...)
	require.NoError(t, err)

	makeItem := func() osqueue.QueueItem {
		return osqueue.QueueItem{
			FunctionID:  fnID,
			WorkspaceID: envID,
			Data: osqueue.Item{
				Kind: osqueue.KindStart,
				Identifier: state.Identifier{
					AccountID:   accountID,
					WorkspaceID: envID,
					WorkflowID:  fnID,
					RunID:       ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader),
				},
				WorkspaceID: envID,
			},
		}
	}

	qi1, err := shard.EnqueueItem(ctx, makeItem(), enqueuedAt, osqueue.EnqueueOpts{})
	require.NoError(t, err)
	qi2, err := shard.EnqueueItem(ctx, makeItem(), enqueuedAt, osqueue.EnqueueOpts{})
	require.NoError(t, err)

	partition := osqueue.ItemPartition(ctx, qi1)
	peekTime := enqueuedAt.Add(5 * time.Second)
	iter := osqueue.ProcessorIterator{
		Partition:  &partition,
		Items:      []*osqueue.QueueItem{&qi1, &qi2},
		Queue:      q,
		Dispatch:   dispatchToOSQueueWorkers(q.Workers()),
		StaticTime: peekTime,
	}

	err = iter.Iterate(ctx)
	require.NoError(t, err)
	require.Equal(t, peekTime.UnixMilli(), qi2.EarliestPeekTime)

	firstWork := <-q.Workers()
	err = q.ProcessItem(ctx, firstWork, func(ctx context.Context, info osqueue.RunInfo, item osqueue.Item) (osqueue.RunResult, error) {
		return osqueue.RunResult{}, nil
	})
	require.NoError(t, err)
	q.Semaphore().Release(1)
	require.NotNil(t, firstWork.CapacityLease)
	require.NoError(t, firstWork.CapacityLease.Release())

	functionVersion = 2
	concurrencyLimit = 2

	processTime := peekTime.Add(7 * time.Second)
	clock.Advance(processTime.Sub(clock.Now()))
	r.SetTime(clock.Now())

	iter = osqueue.ProcessorIterator{
		Partition:  &partition,
		Items:      []*osqueue.QueueItem{&qi2},
		Queue:      q,
		Dispatch:   dispatchToOSQueueWorkers(q.Workers()),
		StaticTime: processTime,
	}
	err = iter.Iterate(ctx)
	require.NoError(t, err)
	require.Len(t, q.Workers(), 1)

	infoCh := make(chan osqueue.RunInfo, 1)
	secondWork := <-q.Workers()
	err = q.ProcessItem(ctx, secondWork, func(ctx context.Context, info osqueue.RunInfo, item osqueue.Item) (osqueue.RunResult, error) {
		infoCh <- info
		return osqueue.RunResult{}, nil
	})
	require.NoError(t, err)
	q.Semaphore().Release(1)
	require.NotNil(t, secondWork.CapacityLease)
	require.NoError(t, secondWork.CapacityLease.Release())

	info := <-infoCh
	require.Equal(t, peekTime.Sub(enqueuedAt), info.Latency)
	require.Equal(t, processTime.Sub(peekTime), info.SojournDelay)
}

func TestSetEarliestPeekTimeOnlyStoresFirstTimestamp(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClockAt(time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC))
	r.SetTime(clock.Now())

	queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shard := redis_state.NewQueueShard("test", queueClient, osqueue.WithClock(clock))

	qi, err := shard.EnqueueItem(ctx, osqueue.QueueItem{}, clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	first := clock.Now()
	second := first.Add(time.Hour)

	got, err := shard.SetEarliestPeekTime(ctx, qi, first)
	require.NoError(t, err)
	require.Equal(t, first.UnixMilli(), got.UnixMilli())

	got, err = shard.SetEarliestPeekTime(ctx, qi, second)
	require.NoError(t, err)
	require.Equal(t, first.UnixMilli(), got.UnixMilli())

	key := queueClient.KeyGenerator().QueueItemEarliestPeekTime(qi.ID)
	val, err := r.Get(key)
	require.NoError(t, err)
	require.Equal(t, strconv.FormatInt(first.UnixMilli(), 10), val)

	ttl := r.TTL(key)
	require.Greater(t, ttl, time.Duration(0))
	require.LessOrEqual(t, ttl, osqueue.QueueItemEarliestPeekTimeTTL)
}

func TestEarliestPeekTimeKeyDeletedOnRequeueAndDequeue(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClockAt(time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC))
	r.SetTime(clock.Now())

	queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
	shard := redis_state.NewQueueShard("test", queueClient, osqueue.WithClock(clock))

	qi, err := shard.EnqueueItem(ctx, osqueue.QueueItem{}, clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	key := queueClient.KeyGenerator().QueueItemEarliestPeekTime(qi.ID)
	_, err = shard.SetEarliestPeekTime(ctx, qi, clock.Now())
	require.NoError(t, err)
	require.True(t, r.Exists(key))

	qi.EarliestPeekTime = clock.Now().UnixMilli()
	err = shard.Requeue(ctx, qi, clock.Now().Add(time.Minute))
	require.NoError(t, err)
	require.False(t, r.Exists(key))

	loaded, err := shard.LoadQueueItem(ctx, qi.ID)
	require.NoError(t, err)
	require.Zero(t, loaded.EarliestPeekTime)

	_, err = shard.SetEarliestPeekTime(ctx, *loaded, clock.Now().Add(time.Second))
	require.NoError(t, err)
	require.True(t, r.Exists(key))

	err = shard.Dequeue(ctx, *loaded)
	require.NoError(t, err)
	require.False(t, r.Exists(key))
}
