package constraintapi_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/ratelimit"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestConstraintEnforcement(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	// Instantiate the user tracer singleton, for some reason we will run into race conditions otherwise
	trace.UserTracer()

	type deps struct {
		cm    constraintapi.CapacityManager
		clock clockwork.FakeClock
		r     *miniredis.Miniredis
		rc    rueidis.Client

		shard queue.ShardOperations

		exec execution.Executor

		config      constraintapi.ConstraintConfig
		constraints []constraintapi.ConstraintItem
	}

	type testCase struct {
		name string

		amount           int
		config           constraintapi.ConstraintConfig
		constraints      []constraintapi.ConstraintItem
		queueConstraints queue.PartitionConstraintConfig

		beforeAcquire func(t *testing.T, deps *deps)

		afterPreAcquireCheck func(t *testing.T, deps *deps, resp *constraintapi.CapacityCheckResponse)

		afterAcquire func(t *testing.T, deps *deps, resp *constraintapi.CapacityAcquireResponse)

		afterPostAcquireCheck func(t *testing.T, deps *deps, resp *constraintapi.CapacityCheckResponse)

		expectedLeaseAmount int

		afterExtend  func(t *testing.T, deps *deps, resp *constraintapi.CapacityExtendLeaseResponse)
		afterRelease func(t *testing.T, deps *deps, resp *constraintapi.CapacityReleaseResponse)

		executorUseConstraintAPI constraintapi.UseConstraintAPIFn
	}

	testCases := []testCase{}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			r := miniredis.RunT(t)
			ctx := context.Background()

			rc, err := rueidis.NewClient(rueidis.ClientOption{
				InitAddress:  []string{r.Addr()},
				DisableCache: true,
			})
			require.NoError(t, err)
			defer rc.Close()

			clock := clockwork.NewFakeClock()

			cm, err := constraintapi.NewRedisCapacityManager(
				constraintapi.WithClient(rc),
				constraintapi.WithShardName("default"),
				constraintapi.WithClock(clock),
				constraintapi.WithEnableDebugLogs(true),
			)
			require.NoError(t, err)
			require.NotNil(t, cm)

			queueOpts := []queue.QueueOpt{
				queue.WithClock(clock),
				queue.WithCapacityManager(cm),
				queue.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (bool, bool) {
					return true, true
				}),
				queue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p queue.PartitionIdentifier) queue.PartitionConstraintConfig {
					return test.queueConstraints
				}),
			}
			shard := redis_state.NewQueueShard("test", redis_state.NewQueueClient(rc, "q:v1"), queueOpts...)

			q, err := queue.New(
				ctx,
				"test-queue",
				shard,
				map[string]queue.QueueShard{
					shard.Name(): shard,
				},
				func(ctx context.Context, accountId uuid.UUID, queueName *string) (queue.QueueShard, error) {
					return shard, nil
				},
				queueOpts...,
			)
			require.NoError(t, err)

			rl := ratelimit.New(ctx, rc, "{rl}:")

			unsharded := redis_state.NewUnshardedClient(rc, "estate", "q:v1")
			sharded := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
				UnshardedClient:        unsharded,
				FunctionRunStateClient: rc,
				BatchClient:            rc,
				StateDefaultKey:        "estate",
				QueueDefaultKey:        "q:v1",
				FnRunIsSharded:         redis_state.AlwaysShardOnRun,
			})

			pauseMgr := pauses.NewPauseStoreManager(unsharded)

			sm, err := redis_state.New(ctx,
				redis_state.WithShardedClient(sharded),
				redis_state.WithPauseDeleter(pauseMgr),
			)
			require.NoError(t, err)
			exec, err := executor.NewExecutor(
				executor.WithRateLimiter(rl),
				executor.WithAssignedQueueShard(shard),
				executor.WithQueue(q),
				executor.WithStateManager(redis_state.MustRunServiceV2(sm)),
				executor.WithPauseManager(pauseMgr),
				executor.WithCapacityManager(cm),
				executor.WithLogger(logger.StdlibLogger(ctx)),
				executor.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
					if test.executorUseConstraintAPI != nil {
						return test.executorUseConstraintAPI(ctx, accountID, envID, functionID)
					}

					return true, true
				}),
				executor.WithClock(clock),
			)
			require.NoError(t, err)

			deps := &deps{
				config:      test.config,
				constraints: test.constraints,
				cm:          cm,
				clock:       clock,
				r:           r,
				rc:          rc,
				shard:       shard,
				exec:        exec,
			}

			if test.beforeAcquire != nil {
				test.beforeAcquire(t, deps)
			}

			leaseIdempotencyKeys := make([]string, test.amount)
			for i := range test.amount {
				leaseIdempotencyKeys[i] = fmt.Sprintf("item%d", i)
			}

			checkResp, _, err := cm.Check(ctx, &constraintapi.CapacityCheckRequest{
				AccountID:     accountID,
				Configuration: test.config,
				Constraints:   test.constraints,
				EnvID:         envID,
				FunctionID:    fnID,
			})
			require.NoError(t, err)

			if test.afterPreAcquireCheck != nil {
				test.afterPreAcquireCheck(t, deps, checkResp)
			}

			acquireResp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
				AccountID:            accountID,
				IdempotencyKey:       "acquire",
				Constraints:          test.constraints,
				Amount:               test.amount,
				EnvID:                envID,
				FunctionID:           fnID,
				Configuration:        test.config,
				LeaseIdempotencyKeys: leaseIdempotencyKeys,
				LeaseRunIDs:          make(map[string]ulid.ULID),
				CurrentTime:          clock.Now(),
				Duration:             5 * time.Second,
				MaximumLifetime:      time.Hour,
				Source: constraintapi.LeaseSource{
					Service:           constraintapi.ServiceExecutor,
					Location:          constraintapi.CallerLocationItemLease,
					RunProcessingMode: constraintapi.RunProcessingModeBackground,
				},
			})
			require.NoError(t, err)

			t.Log(acquireResp.Debug())

			if test.afterAcquire != nil {
				test.afterAcquire(t, deps, acquireResp)
			}

			require.Len(t, acquireResp.Leases, test.expectedLeaseAmount)

			if test.expectedLeaseAmount == 0 {
				return
			}

			checkResp, _, err = cm.Check(ctx, &constraintapi.CapacityCheckRequest{
				AccountID:     accountID,
				Configuration: test.config,
				Constraints:   test.constraints,
				EnvID:         envID,
				FunctionID:    fnID,
			})
			require.NoError(t, err)

			if test.afterPostAcquireCheck != nil {
				test.afterPostAcquireCheck(t, deps, checkResp)
			}

			clock.Advance(2 * time.Second)
			r.FastForward(2 * time.Second)
			r.SetTime(clock.Now())

			for _, lease := range acquireResp.Leases {
				extendResp, err := cm.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{
					IdempotencyKey: "extend",
					LeaseID:        lease.LeaseID,
					AccountID:      accountID,
					Duration:       5 * time.Second,
				})
				require.NoError(t, err)

				if test.afterExtend != nil {
					test.afterExtend(t, deps, extendResp)
				}

				releaseResp, err := cm.Release(ctx, &constraintapi.CapacityReleaseRequest{
					AccountID:      accountID,
					IdempotencyKey: "release",
					LeaseID:        *extendResp.LeaseID,
				})
				require.NoError(t, err)

				if test.afterRelease != nil {
					test.afterRelease(t, deps, releaseResp)
				}
			}
		})
	}
}

