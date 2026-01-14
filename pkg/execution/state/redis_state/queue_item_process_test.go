package redis_state

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestQueueItemProcessWithConstraintChecks(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	ctx := context.Background()
	l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelTrace))
	ctx = logger.WithStdlib(ctx, l)

	cmLifecycles := constraintapi.NewConstraintAPIDebugLifecycles()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClock(clock),
		constraintapi.WithEnableDebugLogs(true),
		constraintapi.WithLifecycles(cmLifecycles),
		constraintapi.WithNumScavengerShards(1),
		constraintapi.WithQueueShards(map[string]rueidis.Client{
			consts.DefaultQueueShardName: rc,
		}),
		constraintapi.WithQueueStateKeyPrefix("queue"),
		constraintapi.WithRateLimitClient(rc),
		constraintapi.WithRateLimitKeyPrefix("rl"),
	)
	require.NoError(t, err)

	reset := func() {
		r.FlushAll()
		r.SetTime(clock.Now())
		cmLifecycles.Reset()
	}

	accountID := uuid.New()
	envID := uuid.New()
	fnID := uuid.New()

	item := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: envID,
		Data: osqueue.Item{
			Payload: json.RawMessage("{\"test\":\"payload\"}"),
			Identifier: state.Identifier{
				AccountID:   accountID,
				WorkspaceID: envID,
				WorkflowID:  fnID,
			},
		},
	}

	start := clock.Now()

	t.Run("without constraint api", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := osqueue.ItemPartition(ctx, qi)

		var counter int64

		err = q.ProcessItem(ctx, osqueue.ProcessItem{
			I:                        qi,
			P:                        p,
			DisableConstraintUpdates: false,
			CapacityLease:            nil,
		}, func(ctx context.Context, ri osqueue.RunInfo, i osqueue.Item) (osqueue.RunResult, error) {
			atomic.AddInt64(&counter, 1)
			return osqueue.RunResult{}, nil
		})
		require.NoError(t, err)

		require.Equal(t, 1, int(counter))
	})

	t.Run("with constraint api but no valid lease", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := osqueue.ItemPartition(ctx, qi)

		var counter int64

		err = q.ProcessItem(ctx, osqueue.ProcessItem{
			I:                        qi,
			P:                        p,
			DisableConstraintUpdates: false,
			CapacityLease:            nil,
		}, func(ctx context.Context, ri osqueue.RunInfo, i osqueue.Item) (osqueue.RunResult, error) {
			<-time.After(3 * time.Second)
			atomic.AddInt64(&counter, 1)
			return osqueue.RunResult{}, nil
		})
		require.NoError(t, err)

		// No extend calls should be fired
		require.Equal(t, 1, int(counter))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
	})

	t.Run("with constraint api and valid lease", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
		)
		kg := shard.Client().kg

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := osqueue.ItemPartition(ctx, qi)

		// Acquire a lease
		resp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			AccountID:            accountID,
			EnvID:                envID,
			IdempotencyKey:       qi.ID,
			FunctionID:           fnID,
			LeaseIdempotencyKeys: []string{qi.ID},
			Amount:               1,
			Configuration: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:  5,
					FunctionConcurrency: 2,
				},
			},
			Constraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: kg.Concurrency("account", accountID.String()),
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeFn,
						InProgressItemKey: kg.Concurrency("p", fnID.String()),
					},
				},
			},
			CurrentTime:     clock.Now(),
			Duration:        10 * time.Second,
			MaximumLifetime: time.Minute,
			Source: constraintapi.LeaseSource{
				Service:           constraintapi.ServiceExecutor,
				Location:          constraintapi.CallerLocationItemLease,
				RunProcessingMode: constraintapi.RunProcessingModeBackground,
			},
			Migration: constraintapi.MigrationIdentifier{
				QueueShard: shard.Name(),
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)

		require.Len(t, cmLifecycles.AcquireCalls, 1)

		var counter int64

		err = q.ProcessItem(ctx, osqueue.ProcessItem{
			I:                        qi,
			P:                        p,
			DisableConstraintUpdates: true,
			CapacityLease: &osqueue.CapacityLease{
				LeaseID: resp.Leases[0].LeaseID,
			},
		}, func(ctx context.Context, ri osqueue.RunInfo, i osqueue.Item) (osqueue.RunResult, error) {
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
						// Ensure we tick the extend at least once
						clock.Advance(time.Second)
					}
				}
			}()

			<-time.After(3 * time.Second)
			atomic.AddInt64(&counter, 1)
			return osqueue.RunResult{}, nil
		})
		require.NoError(t, err)

		require.Equal(t, 1, int(counter))

		service.Wait()

		// Expect at least 1 extend call
		require.Greater(t, len(cmLifecycles.ExtendCalls), 0)

		// Expect exactly 1 release call
		require.Equal(t, len(cmLifecycles.ReleaseCalls), 1)
	})
}

func TestQueueProcessorPreLeaseWithConstraintAPI(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	ctx := context.Background()
	l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelTrace))
	ctx = logger.WithStdlib(ctx, l)

	cmLifecycles := constraintapi.NewConstraintAPIDebugLifecycles()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClock(clock),
		constraintapi.WithEnableDebugLogs(true),
		constraintapi.WithLifecycles(cmLifecycles),
		constraintapi.WithNumScavengerShards(1),
		constraintapi.WithQueueShards(map[string]rueidis.Client{
			consts.DefaultQueueShardName: rc,
		}),
		constraintapi.WithQueueStateKeyPrefix("queue"),
		constraintapi.WithRateLimitClient(rc),
		constraintapi.WithRateLimitKeyPrefix("rl"),
	)
	require.NoError(t, err)

	reset := func() {
		r.FlushAll()
		r.SetTime(clock.Now())
		cmLifecycles.Reset()
	}

	accountID := uuid.New()
	envID := uuid.New()
	fnID := uuid.New()

	item := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: envID,
		Data: osqueue.Item{
			Payload: json.RawMessage("{\"test\":\"payload\"}"),
			Identifier: state.Identifier{
				AccountID:   accountID,
				WorkspaceID: envID,
				WorkflowID:  fnID,
			},
		},
	}

	start := clock.Now()

	t.Run("without constraint api", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				return false, false
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := osqueue.ItemPartition(ctx, qi)

		iter := osqueue.ProcessorIterator{
			Partition:            &p,
			Items:                []*osqueue.QueueItem{&qi},
			PartitionContinueCtr: 0,
			Queue:                q,
			Denies:               osqueue.NewLeaseDenyList(),
			StaticTime:           clock.Now(),
			Parallel:             false,
		}

		err = iter.Process(ctx, &qi)
		require.NoError(t, err)

		require.Equal(t, 0, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("with constraint api and no active lease", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					FunctionVersion: 1,
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  10,
						FunctionConcurrency: 5,
					},
				}
			}),
		)

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		p := osqueue.ItemPartition(ctx, qi)

		iter := osqueue.ProcessorIterator{
			Partition:            &p,
			Items:                []*osqueue.QueueItem{&qi},
			PartitionContinueCtr: 0,
			Queue:                q,
			Denies:               osqueue.NewLeaseDenyList(),
			StaticTime:           clock.Now(),
			Parallel:             false,
		}

		err = iter.Process(ctx, &qi)
		require.NoError(t, err)

		require.Equal(t, 1, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))
	})

	t.Run("with constraint api and active capacity lease", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			osqueue.WithCapacityManager(cm),
			// make lease extensions more frequent
			osqueue.WithCapacityLeaseExtendInterval(time.Second),
			osqueue.WithLogger(l),
		)
		kg := shard.Client().kg

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		// Acquire a lease
		resp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			AccountID:            accountID,
			EnvID:                envID,
			IdempotencyKey:       qi.ID,
			FunctionID:           fnID,
			LeaseIdempotencyKeys: []string{qi.ID},
			Amount:               1,
			Configuration: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:  5,
					FunctionConcurrency: 2,
				},
			},
			Constraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: kg.Concurrency("account", accountID.String()),
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeFn,
						InProgressItemKey: kg.Concurrency("p", fnID.String()),
					},
				},
			},
			CurrentTime:     clock.Now(),
			Duration:        10 * time.Second,
			MaximumLifetime: time.Minute,
			Source: constraintapi.LeaseSource{
				Service:           constraintapi.ServiceExecutor,
				Location:          constraintapi.CallerLocationItemLease,
				RunProcessingMode: constraintapi.RunProcessingModeBackground,
			},
			Migration: constraintapi.MigrationIdentifier{
				QueueShard: shard.Name(),
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)

		require.Len(t, cmLifecycles.AcquireCalls, 1)

		cmLifecycles.Reset()

		// Set capacity lease ID
		qi.CapacityLease = &osqueue.CapacityLease{
			LeaseID: resp.Leases[0].LeaseID,
		}

		p := osqueue.ItemPartition(ctx, qi)

		iter := osqueue.ProcessorIterator{
			Partition:            &p,
			Items:                []*osqueue.QueueItem{&qi},
			PartitionContinueCtr: 0,
			Queue:                q,
			Denies:               osqueue.NewLeaseDenyList(),
			StaticTime:           clock.Now(),
			Parallel:             false,
		}

		err = iter.Process(ctx, &qi)
		require.NoError(t, err)

		// No further Constraint API calls should be made
		require.Equal(t, 0, len(cmLifecycles.AcquireCalls))
		require.Equal(t, 0, len(cmLifecycles.ExtendCalls))
		require.Equal(t, 0, len(cmLifecycles.ReleaseCalls))

		// Expect item to be sent to worker with capacity lease + request to disable constraint updates
		item := <-q.Workers()
		require.Equal(t, qi, item.I)
		require.Equal(t, qi.CapacityLease, item.CapacityLease)
		require.True(t, item.DisableConstraintUpdates)
	})
}

func TestPartitionProcessRequeueAfterLimitedWithConstraintAPI(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	ctx := context.Background()
	l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelTrace))
	ctx = logger.WithStdlib(ctx, l)

	cmLifecycles := constraintapi.NewConstraintAPIDebugLifecycles()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClock(clock),
		constraintapi.WithEnableDebugLogs(true),
		constraintapi.WithLifecycles(cmLifecycles),
		constraintapi.WithNumScavengerShards(1),
		constraintapi.WithQueueShards(map[string]rueidis.Client{
			consts.DefaultQueueShardName: rc,
		}),
		constraintapi.WithQueueStateKeyPrefix("queue"),
		constraintapi.WithRateLimitClient(rc),
		constraintapi.WithRateLimitKeyPrefix("rl"),
	)
	require.NoError(t, err)

	reset := func() {
		r.FlushAll()
		r.SetTime(clock.Now())
		cmLifecycles.Reset()
	}

	accountID := uuid.New()
	envID := uuid.New()
	fnID := uuid.New()

	item := osqueue.QueueItem{
		FunctionID:  fnID,
		WorkspaceID: envID,
		Data: osqueue.Item{
			Payload: json.RawMessage("{\"test\":\"payload\"}"),
			Identifier: state.Identifier{
				AccountID:   accountID,
				WorkspaceID: envID,
				WorkflowID:  fnID,
			},
		},
	}

	start := clock.Now()

	t.Run("without constraintapi and no leases", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				return false, false
			}),
			osqueue.WithCapacityManager(cm),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					FunctionVersion: 1,
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  5,
						FunctionConcurrency: 2,
					},
				}
			}),
		)
		kg := shard.Client().kg

		items := []*osqueue.QueueItem{}

		amount := 10

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		items = append(items, &qi)

		for range amount - 1 {
			qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			items = append(items, &qi)
		}

		p := osqueue.ItemPartition(ctx, qi)
		require.True(t, r.Exists(partitionZsetKey(p, kg)))
		require.Equal(t, 10, zcard(t, rc, partitionZsetKey(p, kg)))

		iter := osqueue.ProcessorIterator{
			Partition: &p,
			// Pass in all items
			Items:                items,
			PartitionContinueCtr: 0,
			Queue:                q,
			Denies:               osqueue.NewLeaseDenyList(),
			StaticTime:           clock.Now(),
			Parallel:             false,
		}

		require.False(t, iter.IsRequeuable())

		// Iterate over all items
		err = iter.Iterate(ctx)
		require.NoError(t, err)

		// first two items were successfully leased
		require.Equal(t, 2, int(iter.CtrSuccess))

		// third item was concurrency limited, we stopped
		require.Equal(t, 1, int(iter.CtrConcurrency), r.Dump())

		// we should requeue the item
		require.True(t, iter.IsRequeuable())

		// the 2 items are in progress
		require.Equal(t, 2, zcard(t, rc, partitionConcurrencyKey(p, kg)))
		require.Equal(t, 2, zcard(t, rc, kg.PartitionScavengerIndex(p.ID))) // item should also be added to scavenger index
		require.Equal(t, 8, zcard(t, rc, partitionZsetKey(p, kg)))

		// expect no calls to constraintapi
		require.Len(t, cmLifecycles.AcquireCalls, 0)
	})

	t.Run("without constraintapi and no leases using processPartition", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				return false, false
			}),
			osqueue.WithCapacityManager(cm),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					FunctionVersion: 1,
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  5,
						FunctionConcurrency: 2,
					},
				}
			}),
		)
		kg := shard.Client().kg

		amount := 10

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		for range amount - 1 {
			_, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
		}

		p := osqueue.ItemPartition(ctx, qi)
		require.True(t, r.Exists(partitionZsetKey(p, kg)))
		require.Equal(t, 10, zcard(t, rc, partitionZsetKey(p, kg)))

		// score in global set is at earliest item
		require.Equal(t, start.Unix(), int64(score(t, r, kg.GlobalPartitionIndex(), p.ID)))

		err = q.ProcessPartition(ctx, &p, 0, false)
		require.NoError(t, err)

		// first two items were successfully leased
		require.Equal(t, 2, zcard(t, rc, partitionConcurrencyKey(p, kg)))
		require.Equal(t, 2, zcard(t, rc, kg.PartitionScavengerIndex(p.ID))) // item should also be added to scavenger index

		// remaining items are still in partition
		require.Equal(t, 8, zcard(t, rc, partitionZsetKey(p, kg)))

		// expect no calls to constraintapi
		require.Len(t, cmLifecycles.AcquireCalls, 0)

		// partition was requeued
		require.Equal(t, start.Add(osqueue.PartitionConcurrencyLimitRequeueExtension).Unix(), int64(score(t, r, kg.GlobalPartitionIndex(), p.ID)))
	})

	t.Run("with constraintapi and no valid leases", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				return true, true // acquire leases
			}),
			osqueue.WithCapacityManager(cm),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					FunctionVersion: 1,
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  5,
						FunctionConcurrency: 2,
					},
				}
			}),
		)
		kg := shard.Client().kg

		items := []*osqueue.QueueItem{}

		amount := 10

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)
		items = append(items, &qi)

		for range amount - 1 {
			qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
			items = append(items, &qi)
		}

		p := osqueue.ItemPartition(ctx, qi)
		require.True(t, r.Exists(partitionZsetKey(p, kg)))
		require.Equal(t, 10, zcard(t, rc, partitionZsetKey(p, kg)))

		iter := osqueue.ProcessorIterator{
			Partition: &p,
			// Pass in all items
			Items:                items,
			PartitionContinueCtr: 0,
			Queue:                q,
			Denies:               osqueue.NewLeaseDenyList(),
			StaticTime:           clock.Now(),
			Parallel:             false,
		}

		require.False(t, iter.IsRequeuable())

		// Iterate over all items
		err = iter.Iterate(ctx)
		require.NoError(t, err)

		// first two items were successfully leased
		require.Equal(t, 2, int(iter.CtrSuccess))

		// third item was concurrency limited, we stopped
		require.Equal(t, 1, int(iter.CtrConcurrency), r.Dump())

		// we should requeue the item
		require.True(t, iter.IsRequeuable())

		// the 2 items are in progress
		require.Equal(t, 0, zcard(t, rc, partitionConcurrencyKey(p, kg)))   // since we used constraint API, item should not be added to in progress items set
		require.Equal(t, 2, zcard(t, rc, kg.PartitionScavengerIndex(p.ID))) // but instead to partition scavenger index
		require.Equal(t, 8, zcard(t, rc, partitionZsetKey(p, kg)))

		// expect 2 successful and 1 failed calls to constraintapi
		require.Len(t, cmLifecycles.AcquireCalls, 3)
		require.Len(t, cmLifecycles.AcquireCalls[0].GrantedLeases, 1)
		require.Len(t, cmLifecycles.AcquireCalls[1].GrantedLeases, 1)
		require.Len(t, cmLifecycles.AcquireCalls[2].GrantedLeases, 0)
		require.Equal(t, cmLifecycles.AcquireCalls[2].LimitingConstraints[0].Kind, constraintapi.ConstraintKindConcurrency)
	})

	t.Run("with constraintapi and no leases using processPartition", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			osqueue.WithCapacityManager(cm),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					FunctionVersion: 1,
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  5,
						FunctionConcurrency: 2,
					},
				}
			}),
		)
		kg := shard.Client().kg

		amount := 10

		qi, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		for range amount - 1 {
			_, err := shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{})
			require.NoError(t, err)
		}

		p := osqueue.ItemPartition(ctx, qi)
		require.True(t, r.Exists(partitionZsetKey(p, kg)))
		require.Equal(t, 10, zcard(t, rc, partitionZsetKey(p, kg)))

		// score in global set is at earliest item
		require.Equal(t, start.Unix(), int64(score(t, r, kg.GlobalPartitionIndex(), p.ID)))

		err = q.ProcessPartition(ctx, &p, 0, false)
		require.NoError(t, err)

		// first two items were successfully leased
		require.Equal(t, 0, zcard(t, rc, partitionConcurrencyKey(p, kg)))   // items should not be in old concurrency zset
		require.Equal(t, 2, zcard(t, rc, kg.PartitionScavengerIndex(p.ID))) // but in scavenger set

		// remaining items are still in partition
		require.Equal(t, 8, zcard(t, rc, partitionZsetKey(p, kg)))

		// expect 2 successful and 1 failed calls to constraintapi
		require.Len(t, cmLifecycles.AcquireCalls, 3)
		require.Len(t, cmLifecycles.AcquireCalls[0].GrantedLeases, 1)
		require.Len(t, cmLifecycles.AcquireCalls[1].GrantedLeases, 1)
		require.Len(t, cmLifecycles.AcquireCalls[2].GrantedLeases, 0)
		require.Equal(t, cmLifecycles.AcquireCalls[2].LimitingConstraints[0].Kind, constraintapi.ConstraintKindConcurrency)

		// partition was requeued
		require.Equal(t, start.Add(osqueue.PartitionConcurrencyLimitRequeueExtension).Unix(), int64(score(t, r, kg.GlobalPartitionIndex(), p.ID)))
	})

	t.Run("with constraintapi and valid leases", func(t *testing.T) {
		reset()

		q, shard := newQueue(
			t, rc,
			osqueue.WithClock(clock),
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithLogger(l),
			osqueue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			osqueue.WithCapacityManager(cm),
			osqueue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p osqueue.PartitionIdentifier) osqueue.PartitionConstraintConfig {
				return osqueue.PartitionConstraintConfig{
					FunctionVersion: 1,
					Concurrency: osqueue.PartitionConcurrency{
						AccountConcurrency:  5,
						FunctionConcurrency: 2,
					},
				}
			}),
		)
		kg := shard.Client().kg

		amount := 10

		/*
		* - Acquire lease for first item
		* - Enqueue item with lease details
		* - Enqueue following items (with later timestamps)
		* - Process partition should peek all items
		* - First item with active lease should be allowed
		* - Second item should be Acquire-checked
		* - Second item should be limited and stop processing
		* - Partition should be requeued
		 */

		var qi osqueue.QueueItem
		{
			var err error
			firstItemID := util.XXHash("item0")

			// Acquire a lease
			resp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
				AccountID:            accountID,
				EnvID:                envID,
				IdempotencyKey:       firstItemID,
				FunctionID:           fnID,
				LeaseIdempotencyKeys: []string{firstItemID},
				Amount:               1,
				Configuration: constraintapi.ConstraintConfig{
					FunctionVersion: 1,
					Concurrency: constraintapi.ConcurrencyConfig{
						AccountConcurrency:  5,
						FunctionConcurrency: 2,
					},
				},
				Constraints: []constraintapi.ConstraintItem{
					{
						Kind: constraintapi.ConstraintKindConcurrency,
						Concurrency: &constraintapi.ConcurrencyConstraint{
							Scope:             enums.ConcurrencyScopeAccount,
							InProgressItemKey: kg.Concurrency("account", accountID.String()),
						},
					},
					{
						Kind: constraintapi.ConstraintKindConcurrency,
						Concurrency: &constraintapi.ConcurrencyConstraint{
							Scope:             enums.ConcurrencyScopeFn,
							InProgressItemKey: kg.Concurrency("p", fnID.String()),
						},
					},
				},
				CurrentTime:     clock.Now(),
				Duration:        10 * time.Second,
				MaximumLifetime: time.Minute,
				Source: constraintapi.LeaseSource{
					Service:           constraintapi.ServiceExecutor,
					Location:          constraintapi.CallerLocationItemLease,
					RunProcessingMode: constraintapi.RunProcessingModeBackground,
				},
				Migration: constraintapi.MigrationIdentifier{
					QueueShard: shard.Name(),
				},
			})
			require.NoError(t, err)
			require.Len(t, resp.Leases, 1)

			require.Len(t, cmLifecycles.AcquireCalls, 1)

			cmLifecycles.Reset()

			// Set capacity lease ID on first item
			item.CapacityLease = &osqueue.CapacityLease{
				LeaseID: resp.Leases[0].LeaseID,
			}
			// Manually set ID for first item
			item.ID = util.XXHash("item0")

			qi, err = shard.EnqueueItem(ctx, item, start, osqueue.EnqueueOpts{
				PassthroughJobId: true,
			})
			require.NoError(t, err)

			// reset for following items
			item.CapacityLease = nil
			item.ID = ""
		}

		for i := range amount - 1 {
			_, err := shard.EnqueueItem(ctx, item, start.Add(time.Millisecond*time.Duration(i+1)), osqueue.EnqueueOpts{})
			require.NoError(t, err)
		}

		p := osqueue.ItemPartition(ctx, qi)
		require.True(t, r.Exists(partitionZsetKey(p, kg)))
		require.Equal(t, 10, zcard(t, rc, partitionZsetKey(p, kg)))

		// score in global set is at earliest item
		require.Equal(t, start.Unix(), int64(score(t, r, kg.GlobalPartitionIndex(), p.ID)))

		err = q.ProcessPartition(logger.WithStdlib(ctx, l), &p, 0, false)
		require.NoError(t, err)

		// first two items were successfully leased
		require.Equal(t, 0, zcard(t, rc, partitionConcurrencyKey(p, kg)))   // items should not be in old concurrency zset
		require.Equal(t, 2, zcard(t, rc, kg.PartitionScavengerIndex(p.ID))) // but in scavenger set

		// remaining items are still in partition
		require.Equal(t, 8, zcard(t, rc, partitionZsetKey(p, kg)))

		// expect 1 successful and 1 failed calls to constraintapi
		require.Len(t, cmLifecycles.AcquireCalls, 2)
		require.Len(t, cmLifecycles.AcquireCalls[0].GrantedLeases, 1)
		require.Len(t, cmLifecycles.AcquireCalls[1].GrantedLeases, 0)
		require.Equal(t, cmLifecycles.AcquireCalls[1].LimitingConstraints[0].Kind, constraintapi.ConstraintKindConcurrency)

		// partition was requeued
		require.Equal(t, start.Add(osqueue.PartitionConcurrencyLimitRequeueExtension).Unix(), int64(score(t, r, kg.GlobalPartitionIndex(), p.ID)))
	})
}
