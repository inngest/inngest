package redis_state

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestQueueCapacityReleased(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))
	_, shard := newQueue(t, rc, osqueue.WithClock(clock))
	notifier, ok := shard.(osqueue.CapacityReleaseNotifier)
	require.True(t, ok)

	ctx := context.Background()
	fnID := uuid.New()
	now := clock.Now()
	forced := now.Add(5 * time.Second)

	qi, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
		FunctionID: fnID,
		Data:       osqueue.Item{Identifier: state.Identifier{AccountID: uuid.New(), WorkflowID: fnID}},
	}, now, osqueue.EnqueueOpts{})
	require.NoError(t, err)
	p := osqueue.ItemPartition(ctx, qi)

	lease := func() bool {
		t.Helper()
		_, err := shard.PartitionLease(ctx, &p, time.Minute)
		require.NoError(t, err)
		return p.CapacityReleased
	}

	t.Run("wakes a parked partition and marks only the next lease", func(t *testing.T) {
		require.False(t, lease())
		require.NoError(t, shard.PartitionRequeue(ctx, &p, forced, true))
		requirePartitionScoreEquals(t, r, &fnID, forced)

		notifier.CapacityReleased(ctx, qi)
		requirePartitionScoreEquals(t, r, &fnID, now)
		require.Zero(t, getDefaultPartition(t, r, fnID).ForceAtMS)

		require.True(t, lease())
		require.NoError(t, shard.PartitionRequeue(ctx, &p, forced, true))
		requirePartitionScoreEquals(t, r, &fnID, forced)

		require.False(t, lease(), "the release is consumed by a single lease")
		require.NoError(t, shard.PartitionRequeue(ctx, &p, forced, true))
	})

	t.Run("a release during a scan prevents its forced backoff", func(t *testing.T) {
		require.False(t, lease())
		notifier.CapacityReleased(ctx, qi)

		require.NoError(t, shard.PartitionRequeue(ctx, &p, forced, true))
		requirePartitionScoreEquals(t, r, &fnID, now)
		require.Zero(t, getDefaultPartition(t, r, fnID).ForceAtMS)

		require.True(t, lease(), "the next scan still skips the stale cache")
		require.NoError(t, shard.PartitionRequeue(ctx, &p, forced, true))
	})

	t.Run("unleased pushbacks stay forced", func(t *testing.T) {
		notifier.CapacityReleased(ctx, qi)

		// Paused and migrating partitions are pushed back without a lease.
		require.NoError(t, shard.PartitionRequeue(ctx, &p, forced, true))
		requirePartitionScoreEquals(t, r, &fnID, forced)
		require.NotZero(t, getDefaultPartition(t, r, fnID).ForceAtMS)

		require.True(t, lease())
		require.NoError(t, shard.PartitionRequeue(ctx, &p, forced, true))
	})
}

type countingCapacityManager struct {
	constraintapi.CapacityManager
	acquires atomic.Int64
}

func (m *countingCapacityManager) Acquire(ctx context.Context, req *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	m.acquires.Add(1)
	return m.CapacityManager.Acquire(ctx, req)
}

// TestCapacityReleasedRefillsCachedPartition reproduces SYS-1449: a scan
// fills a custom concurrency key, caching its exhaustion. Every slot is then
// released, but the retry scan is answered from the stale cache and parks the
// partition with nothing left running to wake it.
func TestCapacityReleasedRefillsCachedPartition(t *testing.T) {
	for _, tc := range []struct {
		name string
		// notify calls CapacityReleased directly after each release.
		notify bool
		// processItem completes items via ProcessItem, which notifies itself.
		processItem bool
		// otherExecutor retries from a second executor with its own stale cache.
		otherExecutor bool
		// wakeDisabled turns the account's capacity release wake off.
		wakeDisabled bool
		// fifo leases items sequentially instead of in parallel.
		fifo     bool
		expected int
	}{
		{name: "dequeue wake only parks the partition", expected: 0},
		{name: "confirmed release refills the key", notify: true, expected: 6},
		{name: "fifo honors exhaustion cached by the retry", notify: true, fifo: true, expected: 6},
		{name: "process item notifies the shard", processItem: true, expected: 6},
		{name: "other executor with a stale cache", notify: true, otherExecutor: true, expected: 6},
		{name: "disabled for the account parks the partition", notify: true, wakeDisabled: true, expected: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
			require.NoError(t, err)
			defer rc.Close()

			clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))
			r.SetTime(clock.Now())

			cm, err := constraintapi.NewRedisCapacityManager(
				constraintapi.WithClient(rc),
				constraintapi.WithClock(clock),
				constraintapi.WithShardName("local"),
			)
			require.NoError(t, err)
			counted := &countingCapacityManager{CapacityManager: cm}
			newCache := func() constraintapi.CapacityManager {
				return constraintapi.NewConstraintCache(
					constraintapi.WithConstraintCacheManager(counted),
					constraintapi.WithConstraintCacheClock(clock),
					constraintapi.WithConstraintCacheEnable(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, time.Duration, time.Duration) {
						return true, time.Second, 2 * time.Second
					}),
				)
			}
			cache := newCache()

			accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()
			const keyExpressionHash = "key-expression"
			const evaluatedKeyHash = "key-a"
			config := osqueue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: osqueue.PartitionConcurrency{
					AccountConcurrency:  1000,
					FunctionConcurrency: 1000,
					CustomConcurrencyKeys: []osqueue.CustomConcurrencyLimit{{
						Mode:                enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeFn,
						HashedKeyExpression: keyExpressionHash,
						Limit:               6,
					}},
				},
			}
			opts := []osqueue.QueueOpt{
				osqueue.WithClock(clock),
				osqueue.WithCapacityManager(cache),
				osqueue.WithPeekSizeRange(100, 100),
				osqueue.WithRunMode(osqueue.QueueRunMode{}),
				osqueue.WithPartitionConstraintConfigGetter(func(context.Context, osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
					return config
				}),
				osqueue.WithCapacityReleaseWakeEnabled(func(_ context.Context, id uuid.UUID) bool {
					require.Equal(t, accountID, id)
					return !tc.wakeDisabled
				}),
			}
			if !tc.fifo {
				opts = append(opts, osqueue.WithDisableFifoForFunctions(map[string]struct{}{fnID.String(): {}}))
			}
			shard := NewQueueShard("local", NewQueueClient(rc, "{sys-1449}"), opts...)
			registry, err := osqueue.NewSingleShardRegistry(shard)
			require.NoError(t, err)
			processor, err := osqueue.New(ctx, "local", registry, opts...)
			require.NoError(t, err)

			var p osqueue.QueuePartition
			for i := 0; i < 100; i++ {
				item, err := shard.EnqueueItem(ctx, osqueue.QueueItem{
					FunctionID:  fnID,
					WorkspaceID: envID,
					Data: osqueue.Item{
						Kind:       osqueue.KindStart,
						Identifier: state.Identifier{AccountID: accountID, WorkspaceID: envID, WorkflowID: fnID, RunID: ulid.Make()},
						CustomConcurrencyKeys: []state.CustomConcurrency{{
							Key:   fmt.Sprintf("f:%s:%s", fnID, evaluatedKeyHash),
							Hash:  keyExpressionHash,
							Limit: 6,
						}},
					},
				}, clock.Now().Add(-time.Second), osqueue.EnqueueOpts{})
				require.NoError(t, err)
				p = osqueue.ItemPartition(ctx, item)
			}

			var mu sync.Mutex
			var dispatched []osqueue.ProcessItem
			dispatch := func(_ context.Context, item osqueue.ProcessItem) (osqueue.DispatchedItem, error) {
				mu.Lock()
				defer mu.Unlock()
				dispatched = append(dispatched, item)
				return osqueue.NewCompletedDispatchedItem(osqueue.DispatchedItemResult{}), nil
			}

			// Fill all six slots; the response caches the key as exhausted.
			require.NoError(t, processor.ProcessPartition(ctx, &p, 0, false, dispatch))
			require.Len(t, dispatched, 6)
			coldAcquires := counted.acquires.Load()

			if tc.otherExecutor {
				// A second executor caches its own denial while the key is full,
				// then performs the retry scan.
				other, err := osqueue.New(ctx, "other", registry, append(opts, osqueue.WithCapacityManager(newCache()))...)
				require.NoError(t, err)

				var denied []osqueue.ProcessItem
				require.NoError(t, other.ProcessPartition(ctx, &p, 0, false, func(_ context.Context, item osqueue.ProcessItem) (osqueue.DispatchedItem, error) {
					denied = append(denied, item)
					return osqueue.NewCompletedDispatchedItem(osqueue.DispatchedItemResult{}), nil
				}))
				require.Empty(t, denied)
				processor = other
			}

			// Complete and release every running item before retrying.
			for _, item := range dispatched {
				require.NotNil(t, item.CapacityLease)
				if tc.processItem {
					_, err := processor.ProcessItem(ctx, item, func(context.Context, osqueue.RunInfo, osqueue.Item) (osqueue.RunResult, error) {
						return osqueue.RunResult{}, nil
					})
					require.NoError(t, err)
					continue
				}

				require.NoError(t, shard.Dequeue(ctx, item.I, osqueue.DequeueCapacityLeased()))
				_, err := cache.Release(ctx, &constraintapi.CapacityReleaseRequest{
					AccountID:      accountID,
					IdempotencyKey: item.I.ID,
					LeaseID:        item.CapacityLease.LeaseID,
					Source: constraintapi.LeaseSource{
						Service:           constraintapi.ServiceExecutor,
						Location:          constraintapi.CallerLocationItemLease,
						RunProcessingMode: constraintapi.RunProcessingModeBackground,
					},
				})
				require.NoError(t, err)
				if tc.notify {
					shard.(osqueue.CapacityReleaseNotifier).CapacityReleased(ctx, item.I)
				}
			}

			kg := shard.Client().KeyGenerator()
			if tc.processItem {
				// ProcessItem notifies asynchronously after the release.
				require.Eventually(t, func() bool {
					return strings.Contains(r.HGet(kg.PartitionItem(), p.ID), "capacityReleased")
				}, 5*time.Second, time.Millisecond)
			}

			// Separate the retry's cache entries from those cached before the release.
			clock.Advance(time.Millisecond)

			before := counted.acquires.Load()
			dispatched = nil
			require.NoError(t, processor.ProcessPartition(ctx, &p, 0, false, dispatch))
			require.Len(t, dispatched, tc.expected)
			acquires := counted.acquires.Load() - before
			t.Logf("retry dispatched=%d authoritative_acquires=%d cold_acquires=%d", len(dispatched), acquires, coldAcquires)
			// Skipping stale exhaustion must not cost more than scanning with an
			// empty cache: exhaustion cached during the retry is still honored.
			require.LessOrEqual(t, acquires, coldAcquires)

			if tc.expected == 0 {
				require.Equal(t, before, counted.acquires.Load(), "the retry is answered entirely by the stale cache")
				score, err := r.ZScore(kg.GlobalPartitionIndex(), p.ID)
				require.NoError(t, err)
				require.Equal(t, float64(clock.Now().Add(osqueue.PartitionConcurrencyLimitRequeueExtension).Unix()), score)
			}
		})
	}
}
