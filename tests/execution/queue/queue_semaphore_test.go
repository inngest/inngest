package queue

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestQueueSemaphoreWithConstraintAPI(t *testing.T) {
	ctx := context.Background()

	l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelDebug))
	ctx = logger.WithStdlib(ctx, l)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)

	cmLifecycles := constraintapi.NewConstraintAPIDebugLifecycles()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClock(clock),
		constraintapi.WithEnableDebugLogs(true),
		constraintapi.WithQueueShards(map[string]rueidis.Client{
			"test": queueClient.Client(),
		}),
		constraintapi.WithQueueStateKeyPrefix(redis_state.QueueDefaultKey),
		constraintapi.WithRateLimitClient(rc),
		constraintapi.WithRateLimitKeyPrefix("rl"),
		constraintapi.WithLifecycles(cmLifecycles),
	)
	require.NoError(t, err)

	options := []queue.QueueOpt{
		queue.WithClock(clock),

		// Use Constraint API
		queue.WithCapacityManager(cm),
		queue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
			return true, true
		}),

		// Simulate a limit of 1
		queue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p queue.PartitionIdentifier) queue.PartitionConstraintConfig {
			return queue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: queue.PartitionConcurrency{
					AccountConcurrency:  1,
					FunctionConcurrency: 1,
				},
			}
		}),
	}

	shard := redis_state.NewQueueShard("test", queueClient, options...)

	q, err := queue.New(ctx, "test", shard, nil, nil, options...)
	require.NoError(t, err)

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	qi1, err := shard.EnqueueItem(ctx, queue.QueueItem{
		Data: queue.Item{
			Kind: queue.KindStart,
			Identifier: state.Identifier{
				AccountID:   accountID,
				WorkspaceID: envID,
				WorkflowID:  fnID,
			},
			WorkspaceID: envID,
		},
		FunctionID: fnID,
	}, clock.Now(), queue.EnqueueOpts{})
	require.NoError(t, err)

	qi2, err := shard.EnqueueItem(ctx, queue.QueueItem{
		Data: queue.Item{
			Kind: queue.KindStart,
			Identifier: state.Identifier{
				AccountID:   accountID,
				WorkspaceID: envID,
				WorkflowID:  fnID,
			},
			WorkspaceID: envID,
		},
		FunctionID: fnID,
	}, clock.Now(), queue.EnqueueOpts{})
	require.NoError(t, err)

	partition := queue.ItemPartition(ctx, qi1)

	iter := queue.ProcessorIterator{
		Partition:  &partition,
		Items:      []*queue.QueueItem{&qi1, &qi2},
		Queue:      q,
		Denies:     queue.NewLeaseDenyList(),
		StaticTime: clock.Now(),
	}

	// Initially, the semaphore must be at 0
	require.Equal(t, int64(0), q.Semaphore().Count())
	require.False(t, iter.IsRequeuable())

	// Attempt to process items sequentially
	err = iter.Iterate(ctx)
	require.NoError(t, err)

	// Expect 2 Acquire requests
	require.Len(t, cmLifecycles.AcquireCalls, 2)

	// First Acquire request should have been successful
	require.Equal(t, len(cmLifecycles.AcquireCalls[0].GrantedLeases), 1)
	require.Equal(t, len(cmLifecycles.AcquireCalls[0].LimitingConstraints), 0)

	// Second Acquire request should have been limited
	require.Equal(t, len(cmLifecycles.AcquireCalls[1].GrantedLeases), 0)
	require.Equal(t, len(cmLifecycles.AcquireCalls[1].LimitingConstraints), 1)
	require.Equal(t, cmLifecycles.AcquireCalls[1].LimitingConstraints[0].Kind, constraintapi.ConstraintKindConcurrency)

	require.True(t, iter.IsRequeuable())

	// Verify semaphore only accounted for the first item
	// This must not include the second item that got limited
	require.Equal(t, int64(1), q.Semaphore().Count())
}

func TestQueueSemaphore(t *testing.T) {
	ctx := context.Background()

	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	type deps struct {
		r            *miniredis.Miniredis
		rc           rueidis.Client
		clock        clockwork.FakeClock
		cmLifecycles *constraintapi.ConstraintApiDebugLifecycles
		cm           constraintapi.CapacityManager
		shard        redis_state.RedisQueueShard
		qp           queue.QueueProcessor
	}

	type testCase struct {
		name                string
		run                 func(t *testing.T, deps deps)
		enableConstraintAPI constraintapi.UseConstraintAPIFn
	}

	testCases := []testCase{
		{
			name: "already leased item should not increase semaphore",
			run: func(t *testing.T, deps deps) {
				shard := deps.shard
				clock := deps.clock

				qi1, err := shard.EnqueueItem(ctx, queue.QueueItem{
					Data: queue.Item{
						Kind: queue.KindStart,
						Identifier: state.Identifier{
							AccountID:   accountID,
							WorkspaceID: envID,
							WorkflowID:  fnID,
						},
						WorkspaceID: envID,
					},
					FunctionID: fnID,
				}, clock.Now(), queue.EnqueueOpts{})
				require.NoError(t, err)

				qi2, err := shard.EnqueueItem(ctx, queue.QueueItem{
					Data: queue.Item{
						Kind: queue.KindStart,
						Identifier: state.Identifier{
							AccountID:   accountID,
							WorkspaceID: envID,
							WorkflowID:  fnID,
						},
						WorkspaceID: envID,
					},
					FunctionID: fnID,
				}, clock.Now(), queue.EnqueueOpts{})
				require.NoError(t, err)

				partition := queue.ItemPartition(ctx, qi1)

				iter := queue.ProcessorIterator{
					Partition:  &partition,
					Items:      []*queue.QueueItem{&qi1, &qi2},
					Queue:      deps.qp,
					Denies:     queue.NewLeaseDenyList(),
					StaticTime: clock.Now(),
				}

				// Initially, the semaphore must be at 0
				require.Equal(t, int64(0), deps.qp.Semaphore().Count())
				require.False(t, iter.IsRequeuable())

				// Attempt to process items sequentially
				err = iter.Iterate(ctx)
				require.NoError(t, err)

				// Expect 2 Acquire requests
				require.Len(t, deps.cmLifecycles.AcquireCalls, 2)

				// First Acquire request should have been successful
				require.Equal(t, len(deps.cmLifecycles.AcquireCalls[0].GrantedLeases), 1)
				require.Equal(t, len(deps.cmLifecycles.AcquireCalls[0].LimitingConstraints), 0)

				// Second Acquire request should have been limited
				require.Equal(t, len(deps.cmLifecycles.AcquireCalls[1].GrantedLeases), 0)
				require.Equal(t, len(deps.cmLifecycles.AcquireCalls[1].LimitingConstraints), 1)
				require.Equal(t, deps.cmLifecycles.AcquireCalls[1].LimitingConstraints[0].Kind, constraintapi.ConstraintKindConcurrency)

				require.True(t, iter.IsRequeuable())

				// Verify semaphore only accounted for the first item
				// This must not include the second item that got limited
				require.Equal(t, int64(1), deps.qp.Semaphore().Count())
			},
		},

		{
			name: "when no capacity available, should exit with expected error",
		},

		{
			name: "when Constraint API call fails, should free semaphore",
		},
		{
			name: "when limited by Constraint API, should release semaphore",
		},
		{
			name: "when lease fails, should release semaphore",
		},
		{
			name: "when lease encounters concurrency limits, should free semaphore",
		},
		{
			name: "when queue item not found, should free semaphore",
		},
		{
			name: "when item leased and handed off to worker, should not free up semaphore",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelDebug))
			ctx = logger.WithStdlib(ctx, l)

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			r := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{
				InitAddress:  []string{r.Addr()},
				DisableCache: true,
			})
			require.NoError(t, err)
			defer rc.Close()

			clock := clockwork.NewFakeClock()

			queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)

			cmLifecycles := constraintapi.NewConstraintAPIDebugLifecycles()
			cm, err := constraintapi.NewRedisCapacityManager(
				constraintapi.WithClock(clock),
				constraintapi.WithEnableDebugLogs(true),
				constraintapi.WithQueueShards(map[string]rueidis.Client{
					"test": queueClient.Client(),
				}),
				constraintapi.WithQueueStateKeyPrefix(redis_state.QueueDefaultKey),
				constraintapi.WithRateLimitClient(rc),
				constraintapi.WithRateLimitKeyPrefix("rl"),
				constraintapi.WithLifecycles(cmLifecycles),
			)
			require.NoError(t, err)

			options := []queue.QueueOpt{
				queue.WithClock(clock),

				// Use Constraint API
				queue.WithCapacityManager(cm),
				queue.WithUseConstraintAPI(tc.enableConstraintAPI),

				// Simulate a limit of 1
				queue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p queue.PartitionIdentifier) queue.PartitionConstraintConfig {
					return queue.PartitionConstraintConfig{
						FunctionVersion: 1,
						Concurrency: queue.PartitionConcurrency{
							AccountConcurrency:  1,
							FunctionConcurrency: 1,
						},
					}
				}),
			}

			shard := redis_state.NewQueueShard("test", queueClient, options...)

			q, err := queue.New(ctx, "test", shard, nil, nil, options...)
			require.NoError(t, err)

			tc.run(t, deps{
				shard:        shard,
				r:            r,
				rc:           rc,
				clock:        clock,
				cmLifecycles: cmLifecycles,
				cm:           cm,
				qp:           q,
			})
		})
	}
}
