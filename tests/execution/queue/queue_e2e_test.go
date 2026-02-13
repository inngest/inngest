package queue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/jonboulle/clockwork"
	"github.com/redis/rueidis"
	"github.com/sourcegraph/conc/pool"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

func TestQueueE2E(t *testing.T) {
	accountID, workspaceID := uuid.New(), uuid.New()

	cases := []struct {
		name             string
		numItems         int
		numFunctions     int
		interval         time.Duration
		concurrency      int
		queueOptions     []queue.QueueOpt
		useConstraintAPI constraintapi.UseConstraintAPIFn
	}{
		{
			name:         "basic test",
			numItems:     10,
			numFunctions: 1,
			concurrency:  1,
			queueOptions: []queue.QueueOpt{
				queue.WithRunMode(
					queue.QueueRunMode{
						Sequential:                        true,
						Scavenger:                         true,
						Partition:                         true,
						Account:                           true,
						AccountWeight:                     85,
						ShadowPartition:                   true,
						AccountShadowPartition:            true,
						AccountShadowPartitionWeight:      85,
						NormalizePartition:                true,
						ShadowContinuationSkipProbability: consts.QueueContinuationSkipProbability,
						Continuations:                     true,
						ShadowContinuations:               true,
					},
				),
				queue.WithAllowKeyQueues(func(ctx context.Context, acctID, envID, fnID uuid.UUID) bool {
					return false
				}),
				queue.WithPollTick(150 * time.Millisecond),
			},
		},
		{
			name:         "with key queues",
			numItems:     10,
			numFunctions: 1,
			concurrency:  1,
			queueOptions: []queue.QueueOpt{
				queue.WithRunMode(
					queue.QueueRunMode{
						Sequential:                        true,
						Scavenger:                         true,
						Partition:                         true,
						Account:                           true,
						AccountWeight:                     85,
						ShadowPartition:                   true,
						AccountShadowPartition:            true,
						AccountShadowPartitionWeight:      85,
						NormalizePartition:                true,
						ShadowContinuationSkipProbability: consts.QueueContinuationSkipProbability,
						Continuations:                     true,
						ShadowContinuations:               true,
					},
				),
				queue.WithAllowKeyQueues(func(ctx context.Context, acctID, envID, fnID uuid.UUID) bool {
					return true
				}),
				queue.WithPollTick(150 * time.Millisecond),
			},
		},
		{
			name:         "with capacity manager and key queues",
			numItems:     10,
			numFunctions: 1,
			concurrency:  1,
			queueOptions: []queue.QueueOpt{
				queue.WithRunMode(
					queue.QueueRunMode{
						Sequential:                        true,
						Scavenger:                         true,
						Partition:                         true,
						Account:                           true,
						AccountWeight:                     85,
						ShadowPartition:                   true,
						AccountShadowPartition:            true,
						AccountShadowPartitionWeight:      85,
						NormalizePartition:                true,
						ShadowContinuationSkipProbability: consts.QueueContinuationSkipProbability,
						Continuations:                     true,
						ShadowContinuations:               true,
					},
				),
				queue.WithAllowKeyQueues(func(ctx context.Context, acctID, envID, fnID uuid.UUID) bool {
					return true
				}),
				queue.WithPollTick(150 * time.Millisecond),
			},
			useConstraintAPI: func(ctx context.Context, accountID uuid.UUID) (enable bool) {
				return true
			},
		},
		{
			name:         "with capacity manager",
			numItems:     10,
			numFunctions: 1,
			concurrency:  1,
			queueOptions: []queue.QueueOpt{
				queue.WithRunMode(
					queue.QueueRunMode{
						Sequential:                        true,
						Scavenger:                         true,
						Partition:                         true,
						Account:                           true,
						AccountWeight:                     85,
						ShadowPartition:                   true,
						AccountShadowPartition:            true,
						AccountShadowPartitionWeight:      85,
						NormalizePartition:                true,
						ShadowContinuationSkipProbability: consts.QueueContinuationSkipProbability,
						Continuations:                     true,
						ShadowContinuations:               true,
					},
				),
				queue.WithAllowKeyQueues(func(ctx context.Context, acctID, envID, fnID uuid.UUID) bool {
					return false
				}),
				queue.WithPollTick(150 * time.Millisecond),
			},
			useConstraintAPI: func(ctx context.Context, accountID uuid.UUID) (enable bool) {
				return true
			},
		},
	}

	timeTick := 100 * time.Millisecond
	timeMultiplier := 1

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			err := trace.NewSystemTracer(ctx, trace.TracerOpts{
				ServiceName:   "tracing-system",
				TraceEndpoint: "localhost:4318",
				Type:          trace.TracerTypeOTLPHTTP,
			})
			require.NoError(t, err)
			defer func() {
				_ = trace.CloseSystemTracer(ctx)
			}()

			tracer := trace.NewConditionalTracer(trace.QueueTracer(), trace.AlwaysTrace)

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
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-time.Tick(timeTick):
						clock.Advance(time.Duration(timeMultiplier) * timeTick)
						r.FastForward(time.Duration(timeMultiplier) * timeTick)
						r.SetTime(clock.Now())
					}
				}
			}()

			fnIDs := make([]uuid.UUID, tc.numFunctions)
			for i := range fnIDs {
				fnIDs[i] = uuid.New()
			}

			options := append([]queue.QueueOpt{
				queue.WithClock(clock),
				queue.WithConditionalTracer(tracer),
				queue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p queue.PartitionIdentifier) queue.PartitionConstraintConfig {
					return queue.PartitionConstraintConfig{
						FunctionVersion: 1,
						Concurrency: queue.PartitionConcurrency{
							SystemConcurrency:   consts.DefaultConcurrencyLimit,
							AccountConcurrency:  consts.DefaultConcurrencyLimit,
							FunctionConcurrency: consts.DefaultConcurrencyLimit,
						},
					}
				}),
			}, tc.queueOptions...)

			cm, err := constraintapi.NewRedisCapacityManager(
				constraintapi.WithClient(rc),
				constraintapi.WithShardName("test"),
				constraintapi.WithClock(clock),
				constraintapi.WithEnableDebugLogs(true),
			)
			require.NoError(t, err)

			if tc.useConstraintAPI != nil {
				options = append(options, queue.WithCapacityManager(cm))
				options = append(options,
					queue.WithUseConstraintAPI(tc.useConstraintAPI),
				)
			}

			queueClient := redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey)
			shard := redis_state.NewQueueShard("test", queueClient, options...)

			q, err := queue.New(ctx, "test", shard, nil, nil, options...)
			require.NoError(t, err)

			sem := pool.New().
				WithErrors().WithFirstError().WithMaxGoroutines(tc.concurrency)

			// Start enqueueing
			go func() {
				for i := range tc.numItems {
					fnID := fnIDs[i%len(fnIDs)]

					sem.Go(func() error {
						at := clock.Now().Add(time.Duration(i) * tc.interval)
						jobID := fmt.Sprintf("item%d", i)
						err := q.Enqueue(ctx, queue.Item{
							JobID:       &jobID,
							WorkspaceID: workspaceID,
							Identifier: state.Identifier{
								AccountID:   accountID,
								WorkspaceID: workspaceID,
								WorkflowID:  fnID,
							},
							Kind: queue.KindStart,
						}, at, queue.EnqueueOpts{
							PassthroughJobId: true,
						})
						return err
					})
				}
			}()

			// Immediately acquire all capacity
			// When this hits 0, we can quit (all items are processed)
			waitUntilCompleted := semaphore.NewWeighted(int64(tc.numItems))
			require.NoError(t, waitUntilCompleted.Acquire(ctx, int64(tc.numItems)))

			// Start running
			go func() {
				err := q.Run(ctx, func(ctx context.Context, ri queue.RunInfo, i queue.Item) (queue.RunResult, error) {
					l.Debug("completed", "id", *i.JobID)

					// Decrease in-progress semaphore
					waitUntilCompleted.Release(1)

					return queue.RunResult{}, nil
				})
				require.NoError(t, err)

				// Wait until all enqueues finished
				err = sem.Wait()
				require.NoError(t, err)
			}()

			// Wait until completed
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Second):
				}

				if waitUntilCompleted.TryAcquire(int64(tc.numItems)) {
					// Stop the worker
					break
				}
			}
		})
	}
}
