package queue

import (
	"context"
	"crypto/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
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
		constraintapi.WithClient(rc),
		constraintapi.WithShardName("test"),
		constraintapi.WithClock(clock),
		constraintapi.WithEnableDebugLogs(true),
		constraintapi.WithLifecycles(cmLifecycles),
	)
	require.NoError(t, err)

	options := []queue.QueueOpt{
		queue.WithClock(clock),

		// Use Constraint API
		queue.WithCapacityManager(cm),
		queue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
			return true
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
		r                         *miniredis.Miniredis
		rc                        rueidis.Client
		clock                     clockwork.FakeClock
		cmLifecycles              *constraintapi.ConstraintApiDebugLifecycles
		cm                        constraintapi.CapacityManager
		shard                     redis_state.RedisQueueShard
		qp                        queue.QueueProcessor
		failingAcquireCallCounter *int64
	}

	type testCase struct {
		name                      string
		run                       func(t *testing.T, deps deps)
		enableConstraintAPI       constraintapi.UseConstraintAPIFn
		useFailingCapacityManager bool
		config                    queue.PartitionConstraintConfig
		numWorkers                int32
	}

	testCases := []testCase{
		{
			name: "happy path: semaphore should increase for queue item",
			config: queue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: queue.PartitionConcurrency{
					AccountConcurrency:  1,
					FunctionConcurrency: 1,
				},
			},
			numWorkers: 10,
			run: func(t *testing.T, deps deps) {
				shard := deps.shard
				clock := deps.clock

				qi, err := shard.EnqueueItem(ctx, queue.QueueItem{
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

				partition := queue.ItemPartition(ctx, qi)

				iter := queue.ProcessorIterator{
					Partition:  &partition,
					Items:      []*queue.QueueItem{&qi},
					Queue:      deps.qp,
					Denies:     queue.NewLeaseDenyList(),
					StaticTime: clock.Now(),
				}

				// Initially, the semaphore must be at 0
				require.Equal(t, int64(0), deps.qp.Semaphore().Count())

				err = iter.Process(ctx, &qi)
				require.NoError(t, err)

				// Ensure that semaphore was increased
				require.Equal(t, int64(1), deps.qp.Semaphore().Count())

				// Check if item was added to worker
				item := <-deps.qp.Workers()
				require.Equal(t, qi, item.I)
			},
		},

		{
			name: "when item already leased, should release semaphore",
			config: queue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: queue.PartitionConcurrency{
					AccountConcurrency:  1,
					FunctionConcurrency: 1,
				},
			},
			numWorkers: 10,
			run: func(t *testing.T, deps deps) {
				shard := deps.shard
				clock := deps.clock

				leaseID := ulid.MustNew(ulid.Timestamp(clock.Now().Add(10*time.Second)), rand.Reader)
				qi, err := shard.EnqueueItem(ctx, queue.QueueItem{
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
					LeaseID:    &leaseID,
				}, clock.Now(), queue.EnqueueOpts{})
				require.NoError(t, err)

				partition := queue.ItemPartition(ctx, qi)

				iter := queue.ProcessorIterator{
					Partition:  &partition,
					Items:      []*queue.QueueItem{&qi},
					Queue:      deps.qp,
					Denies:     queue.NewLeaseDenyList(),
					StaticTime: clock.Now(),
				}

				// Initially, the semaphore must be at 0
				require.Equal(t, int64(0), deps.qp.Semaphore().Count())

				err = iter.Process(ctx, &qi)
				require.NoError(t, err)

				// Semaphore should still be 0
				require.Equal(t, int64(0), deps.qp.Semaphore().Count())
			},
		},

		{
			name:       "when no capacity available, should exit with expected error",
			numWorkers: 1,
			run: func(t *testing.T, deps deps) {
				shard := deps.shard
				clock := deps.clock

				qi, err := shard.EnqueueItem(ctx, queue.QueueItem{
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

				partition := queue.ItemPartition(ctx, qi)

				iter := queue.ProcessorIterator{
					Partition:  &partition,
					Items:      []*queue.QueueItem{&qi},
					Queue:      deps.qp,
					Denies:     queue.NewLeaseDenyList(),
					StaticTime: clock.Now(),
				}

				// Initially, the semaphore must be at 0
				require.Equal(t, int64(0), deps.qp.Semaphore().Count())

				// Simulate acquiring worker capacity
				require.NoError(t, deps.qp.Semaphore().Acquire(ctx, 1))
				require.Equal(t, int64(1), deps.qp.Semaphore().Count())

				err = iter.Process(ctx, &qi)
				require.Error(t, err)
				require.ErrorIs(t, err, queue.ErrProcessNoCapacity)

				// Semaphore should still be 1
				require.Equal(t, int64(1), deps.qp.Semaphore().Count())
			},
		},

		{
			name: "when Constraint API call fails, should release semaphore",
			enableConstraintAPI: func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				// Constraint API enabled with fail-hard behavior
				return true
			},
			useFailingCapacityManager: true,
			config: queue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: queue.PartitionConcurrency{
					AccountConcurrency:  1,
					FunctionConcurrency: 1,
				},
			},
			run: func(t *testing.T, deps deps) {
				shard := deps.shard
				clock := deps.clock

				qi, err := shard.EnqueueItem(ctx, queue.QueueItem{
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

				partition := queue.ItemPartition(ctx, qi)

				iter := queue.ProcessorIterator{
					Partition:  &partition,
					Items:      []*queue.QueueItem{&qi},
					Queue:      deps.qp,
					Denies:     queue.NewLeaseDenyList(),
					StaticTime: clock.Now(),
				}

				require.Equal(t, int64(0), deps.qp.Semaphore().Count())

				err = iter.Process(ctx, &qi)
				require.Error(t, err)

				require.Equal(t, int64(1), atomic.LoadInt64(deps.failingAcquireCallCounter))

				require.Equal(t, int64(0), deps.qp.Semaphore().Count())
			},
		},
		{
			name: "when limited by Constraint API, should release semaphore",
			enableConstraintAPI: func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return true
			},
			config: queue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: queue.PartitionConcurrency{
					AccountConcurrency:  1,
					FunctionConcurrency: 1,
				},
			},
			run: func(t *testing.T, deps deps) {
				shard := deps.shard
				clock := deps.clock

				// First item should not be limited
				qi, err := shard.EnqueueItem(ctx, queue.QueueItem{
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

				// Second item should be limited
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

				partition := queue.ItemPartition(ctx, qi)

				iter := queue.ProcessorIterator{
					Partition:  &partition,
					Items:      []*queue.QueueItem{&qi, &qi2},
					Queue:      deps.qp,
					Denies:     queue.NewLeaseDenyList(),
					StaticTime: clock.Now(),
				}

				require.Equal(t, int64(0), deps.qp.Semaphore().Count())

				err = iter.Process(ctx, &qi)
				require.NoError(t, err)

				require.Equal(t, int64(1), deps.qp.Semaphore().Count())
				require.Equal(t, int32(0), iter.CtrConcurrency.Load())

				err = iter.Process(ctx, &qi2)
				require.Error(t, err)
				require.ErrorIs(t, err, queue.ErrProcessStopIterator)
				require.ErrorContains(t, err, "concurrency hit")
				require.Equal(t, int32(1), iter.CtrConcurrency.Load())

				require.Equal(t, int64(1), deps.qp.Semaphore().Count())
			},
		},
		{
			name: "when lease fails, should release semaphore",
			config: queue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: queue.PartitionConcurrency{
					AccountConcurrency:  1,
					FunctionConcurrency: 1,
				},
			},
			run: func(t *testing.T, deps deps) {
				clock := deps.clock

				// Simply fake queue item -- this will not exist so Lease will fail
				qi := queue.QueueItem{
					Data: queue.Item{
						Kind: queue.KindStart,
						Identifier: state.Identifier{
							AccountID:   accountID,
							WorkspaceID: envID,
							WorkflowID:  fnID,
						},
						WorkspaceID: envID,
					},
					FunctionID:  fnID,
					ID:          util.XXHash("random"),
					AtMS:        clock.Now().UnixMilli(),
					WallTimeMS:  clock.Now().UnixMilli(),
					WorkspaceID: envID,
					EnqueuedAt:  clock.Now().UnixMilli(),
				}

				partition := queue.ItemPartition(ctx, qi)

				iter := queue.ProcessorIterator{
					Partition:  &partition,
					Items:      []*queue.QueueItem{&qi},
					Queue:      deps.qp,
					Denies:     queue.NewLeaseDenyList(),
					StaticTime: clock.Now(),
				}

				require.Equal(t, int64(0), deps.qp.Semaphore().Count())

				err := iter.Process(ctx, &qi)
				require.NoError(t, err)

				// Still expect semaphore to be at 0
				require.Equal(t, int64(0), deps.qp.Semaphore().Count())
			},
		},
		{
			name: "when lease encounters concurrency limits, should free semaphore",
			enableConstraintAPI: func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
				return false
			},
			config: queue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: queue.PartitionConcurrency{
					AccountConcurrency:  1,
					FunctionConcurrency: 1,
				},
			},
			run: func(t *testing.T, deps deps) {
				shard := deps.shard
				clock := deps.clock

				// First item should not be limited
				qi, err := shard.EnqueueItem(ctx, queue.QueueItem{
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

				// Second item should be limited
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

				partition := queue.ItemPartition(ctx, qi)

				iter := queue.ProcessorIterator{
					Partition:  &partition,
					Items:      []*queue.QueueItem{&qi, &qi2},
					Queue:      deps.qp,
					Denies:     queue.NewLeaseDenyList(),
					StaticTime: clock.Now(),
				}

				require.Equal(t, int64(0), deps.qp.Semaphore().Count())

				err = iter.Process(ctx, &qi)
				require.NoError(t, err)

				require.Equal(t, int64(1), deps.qp.Semaphore().Count())
				require.Equal(t, int32(0), iter.CtrConcurrency.Load())

				err = iter.Process(ctx, &qi2)
				require.Error(t, err)
				require.ErrorIs(t, err, queue.ErrProcessStopIterator)
				require.ErrorContains(t, err, "concurrency hit")
				require.Equal(t, int32(1), iter.CtrConcurrency.Load())

				require.Equal(t, int64(1), deps.qp.Semaphore().Count())
			},
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

			var cm constraintapi.CapacityManager
			var failingAcquireCallCounter int64

			if tc.useFailingCapacityManager {
				cm = newFailingCapacityManager(&failingAcquireCallCounter)
			} else {
				cm, err = constraintapi.NewRedisCapacityManager(
					constraintapi.WithClient(rc),
					constraintapi.WithShardName("test"),
					constraintapi.WithClock(clock),
					constraintapi.WithEnableDebugLogs(true),
					constraintapi.WithLifecycles(cmLifecycles),
				)
				require.NoError(t, err)
			}

			if tc.numWorkers == 0 {
				tc.numWorkers = 5_000
			}

			options := []queue.QueueOpt{
				queue.WithClock(clock),

				// Use Constraint API
				queue.WithCapacityManager(cm),
				queue.WithUseConstraintAPI(tc.enableConstraintAPI),

				// Simulate a limit of 1
				queue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p queue.PartitionIdentifier) queue.PartitionConstraintConfig {
					return tc.config
				}),

				queue.WithNumWorkers(tc.numWorkers),
			}

			shard := redis_state.NewQueueShard("test", queueClient, options...)

			q, err := queue.New(ctx, "test", shard, nil, nil, options...)
			require.NoError(t, err)

			require.NotNil(t, tc.run)

			tc.run(t, deps{
				shard:                     shard,
				r:                         r,
				rc:                        rc,
				clock:                     clock,
				cmLifecycles:              cmLifecycles,
				cm:                        cm,
				qp:                        q,
				failingAcquireCallCounter: &failingAcquireCallCounter,
			})
		})
	}
}

type failingCapacityManagerImpl struct {
	acquireCalls *int64
}

func (f *failingCapacityManagerImpl) Acquire(ctx context.Context, req *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	atomic.AddInt64(f.acquireCalls, 1)
	return nil, errs.Wrap(0, false, "fake err")
}

func (f *failingCapacityManagerImpl) Check(ctx context.Context, req *constraintapi.CapacityCheckRequest) (*constraintapi.CapacityCheckResponse, errs.UserError, errs.InternalError) {
	return nil, nil, errs.Wrap(0, false, "fake err")
}

func (f *failingCapacityManagerImpl) ExtendLease(ctx context.Context, req *constraintapi.CapacityExtendLeaseRequest) (*constraintapi.CapacityExtendLeaseResponse, errs.InternalError) {
	return nil, errs.Wrap(0, false, "fake err")
}

func (f *failingCapacityManagerImpl) Release(ctx context.Context, req *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	return nil, errs.Wrap(0, false, "fake err")
}

func newFailingCapacityManager(acquireCallCounter *int64) constraintapi.CapacityManager {
	return &failingCapacityManagerImpl{
		acquireCalls: acquireCallCounter,
	}
}
