package constraintapi_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/ratelimit"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestConstraintEnforcement(t *testing.T) {
	accountID, envID, fnID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// Instantiate the user tracer singleton, for some reason we will run into race conditions otherwise
	trace.UserTracer()

	type deps struct {
		cm    constraintapi.RolloutManager
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
		mi               constraintapi.MigrationIdentifier

		beforeAcquire func(t *testing.T, deps *deps)

		afterPreAcquireCheck func(t *testing.T, deps *deps, resp *constraintapi.CapacityCheckResponse)

		afterAcquire func(t *testing.T, deps *deps, resp *constraintapi.CapacityAcquireResponse)

		afterPostAcquireCheck func(t *testing.T, deps *deps, resp *constraintapi.CapacityCheckResponse)

		expectedLeaseAmount int

		afterExtend  func(t *testing.T, deps *deps, resp *constraintapi.CapacityExtendLeaseResponse)
		afterRelease func(t *testing.T, deps *deps, resp *constraintapi.CapacityReleaseResponse)

		executorUseConstraintAPI constraintapi.UseConstraintAPIFn
	}

	kg := redis_state.NewQueueClient(nil, "q:v1").KeyGenerator()

	testCases := []testCase{
		{
			name: "account concurrency limited due to legacy concurrency with queue",
			config: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency: 10,
				},
			},
			constraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						Mode:              enums.ConcurrencyModeStep,
						InProgressItemKey: kg.Concurrency("account", accountID.String()),
					},
				},
			},
			mi: constraintapi.MigrationIdentifier{
				QueueShard: "test",
			},
			beforeAcquire: func(t *testing.T, deps *deps) {
				clock := deps.clock
				q := deps.shard
				// Simulate existing concurrency usage (in progress item Leased by queue)
				for i := range 5 { // 5/10
					qi, err := q.EnqueueItem(
						context.Background(),
						queue.QueueItem{
							ID:          fmt.Sprintf("item%d", i),
							FunctionID:  fnID,
							WorkspaceID: envID,
							Data: queue.Item{
								WorkspaceID: envID,
								Kind:        queue.KindStart,
								Identifier: state.Identifier{
									AccountID:   accountID,
									WorkspaceID: envID,
									WorkflowID:  fnID,
								},
							},
						},
						clock.Now(),
						queue.EnqueueOpts{},
					)
					require.NoError(t, err)
					require.NotNil(t, qi)

					leaseID, err := q.Lease(context.Background(), qi, 5*time.Second, clock.Now(), nil)
					require.NoError(t, err)
					require.NotNil(t, leaseID)
				}
			},
			amount:              10,
			expectedLeaseAmount: 5,
		},

		{
			// This test checks ensures that Acquire ignores expired concurrency claimed by the queue.
			name: "expired legacy account concurrency should be gracefully ignored",
			config: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency: 10,
				},
			},
			constraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						Mode:              enums.ConcurrencyModeStep,
						InProgressItemKey: kg.Concurrency("account", accountID.String()),
					},
				},
			},
			mi: constraintapi.MigrationIdentifier{
				QueueShard: "test",
			},
			beforeAcquire: func(t *testing.T, deps *deps) {
				clock := deps.clock
				q := deps.shard

				// Simulate existing concurrency usage (in progress item Leased by queue)
				for i := range 5 { // 5/10
					qi, err := q.EnqueueItem(
						context.Background(),
						queue.QueueItem{
							ID:          fmt.Sprintf("item%d", i),
							FunctionID:  fnID,
							WorkspaceID: envID,
							Data: queue.Item{
								WorkspaceID: envID,
								Kind:        queue.KindStart,
								Identifier: state.Identifier{
									AccountID:   accountID,
									WorkspaceID: envID,
									WorkflowID:  fnID,
								},
							},
						},
						clock.Now(),
						queue.EnqueueOpts{},
					)
					require.NoError(t, err)
					require.NotNil(t, qi)

					leaseID, err := q.Lease(context.Background(), qi, 5*time.Second, clock.Now(), nil)
					require.NoError(t, err)
					require.NotNil(t, leaseID)
				}

				// Advance time to expire leases
				clock.Advance(10 * time.Second)
				deps.r.FastForward(10 * time.Second)
				deps.r.SetTime(clock.Now())
			},
			amount:              10,
			expectedLeaseAmount: 10,
		},

		{
			// This test checks ensures that Throttle constraint state set in the queue is respected by the Constraint API
			name: "existing throttle should be respected",
			config: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Throttle: []constraintapi.ThrottleConfig{
					{
						Scope:             enums.ThrottleScopeFn,
						Period:            60,
						Limit:             1,
						Burst:             0,
						KeyExpressionHash: "expr-hash",
					},
				},
			},
			constraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindThrottle,
					Throttle: &constraintapi.ThrottleConstraint{
						Scope:             enums.ThrottleScopeFn,
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "throttle-key",
					},
				},
			},
			queueConstraints: queue.PartitionConstraintConfig{
				FunctionVersion: 1,
				Throttle: &queue.PartitionThrottle{
					ThrottleKeyExpressionHash: "expr-hash",
					Period:                    60,
					Limit:                     1,
					Burst:                     0,
				},
			},
			mi: constraintapi.MigrationIdentifier{
				QueueShard: "test",
			},
			beforeAcquire: func(t *testing.T, deps *deps) {
				clock := deps.clock
				q := deps.shard
				r := deps.r

				for i := range 1 {
					clock.Advance(time.Millisecond)
					deps.r.FastForward(time.Millisecond)
					deps.r.SetTime(clock.Now())

					// Simulate existing throttle usage
					qi, err := q.EnqueueItem(
						context.Background(),
						queue.QueueItem{
							ID:          fmt.Sprintf("item%d", i),
							FunctionID:  fnID,
							WorkspaceID: envID,
							Data: queue.Item{
								WorkspaceID: envID,
								Kind:        queue.KindStart,
								Identifier: state.Identifier{
									AccountID:   accountID,
									WorkspaceID: envID,
									WorkflowID:  fnID,
								},
								Throttle: &queue.Throttle{
									KeyExpressionHash: "expr-hash",
									Key:               "throttle-key",
									Period:            60,
									Limit:             1,
									Burst:             1,
								},
							},
						},
						clock.Now(),
						queue.EnqueueOpts{},
					)
					require.NoError(t, err)
					require.NotNil(t, qi)

					leaseID, err := q.Lease(context.Background(), qi, 5*time.Second, clock.Now(), nil)
					require.NoError(t, err)
					require.NotNil(t, leaseID)

					r.Exists(kg.ThrottleKey(qi.Data.Throttle))
				}
			},
			amount:              1,
			expectedLeaseAmount: 0,
			afterAcquire: func(t *testing.T, deps *deps, resp *constraintapi.CapacityAcquireResponse) {
				require.Len(t, resp.Leases, 0)
				require.Len(t, resp.LimitingConstraints, 1)
				require.Equal(t, "throttle-key", resp.LimitingConstraints[0].Throttle.EvaluatedKeyHash)
			},
		},

		// Rate limit set by Schedule() is respected
		{
			name: "rate limited by gcra state set in schedule",
			config: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				RateLimit: []constraintapi.RateLimitConfig{
					{
						Limit:             1,
						Period:            60,
						KeyExpressionHash: util.XXHash("event.data.customerID"),
					},
				},
			},
			queueConstraints: queue.PartitionConstraintConfig{
				FunctionVersion: 1,
			},
			constraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindRateLimit,
					RateLimit: &constraintapi.RateLimitConstraint{
						KeyExpressionHash: util.XXHash("event.data.customerID"),
						EvaluatedKeyHash:  fmt.Sprintf("%s-%s", fnID, util.XXHash("user1")),
					},
				},
			},
			mi: constraintapi.MigrationIdentifier{
				IsRateLimit: true,
			},
			amount:              1,
			expectedLeaseAmount: 0,
			executorUseConstraintAPI: func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				// Disable Constraint API for this test
				return false, false
			},
			beforeAcquire: func(t *testing.T, deps *deps) {
				ctx := context.Background()

				idempotencyKey := "outside-idempotency-key"
				eventID := ulid.MustNew(ulid.Timestamp(deps.clock.Now()), rand.Reader)

				rateLimitExpr := "event.data.customerID"
				fnConfig := inngest.Function{
					ID:              fnID,
					FunctionVersion: 1,
					RateLimit: &inngest.RateLimit{
						Limit:  5,
						Period: "60s",
						Key:    &rateLimitExpr,
					},
					Name: "test function",
					Slug: "test-function",
					Triggers: inngest.MultipleTriggers{
						inngest.Trigger{
							EventTrigger: &inngest.EventTrigger{
								Event: "test/event",
							},
							CronTrigger: nil,
						},
					},
				}

				md, err := deps.exec.Schedule(ctx, execution.ScheduleRequest{
					Function:       fnConfig,
					AccountID:      accountID,
					WorkspaceID:    envID,
					AppID:          appID,
					IdempotencyKey: &idempotencyKey,
					Events: []event.TrackedEvent{
						event.NewBaseTrackedEventWithID(event.Event{
							Name: "test/event",
							Data: map[string]any{
								"customerID": "user1",
							},
						}, eventID),
					},
				})
				require.NoError(t, err)
				require.NotNil(t, md)
				require.NotNil(t, md.ID.RunID)

				rateLimitKeyHash := util.XXHash("user1")
				keyRateLimitState := fmt.Sprintf("{rl}:%s-%s", fnID, rateLimitKeyHash)
				require.True(t, deps.r.Exists(keyRateLimitState))
			},
			afterAcquire: func(t *testing.T, deps *deps, resp *constraintapi.CapacityAcquireResponse) {
				t.Log(resp.Debug())

				require.Len(t, resp.LimitingConstraints, 1)
				require.Equal(t, constraintapi.ConstraintKindRateLimit, resp.LimitingConstraints[0].Kind)
			},
		},

		// Rate limit set by Schedule() is respected
		{
			name: "rate limit gcra state set in schedule checked but allowed by Acquire call",
			config: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				RateLimit: []constraintapi.RateLimitConfig{
					{
						Limit:             10,
						Period:            60,
						KeyExpressionHash: util.XXHash("event.data.customerID"),
					},
				},
			},
			queueConstraints: queue.PartitionConstraintConfig{
				FunctionVersion: 1,
			},
			constraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindRateLimit,
					RateLimit: &constraintapi.RateLimitConstraint{
						KeyExpressionHash: util.XXHash("event.data.customerID"),
						EvaluatedKeyHash:  fmt.Sprintf("%s-%s", fnID, util.XXHash("user1")),
					},
				},
			},
			mi: constraintapi.MigrationIdentifier{
				IsRateLimit: true,
			},
			amount:              1,
			expectedLeaseAmount: 1,
			executorUseConstraintAPI: func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				// Disable Constraint API for this test
				return false, false
			},
			beforeAcquire: func(t *testing.T, deps *deps) {
				ctx := context.Background()

				idempotencyKey := "outside-idempotency-key"
				eventID := ulid.MustNew(ulid.Timestamp(deps.clock.Now()), rand.Reader)

				rateLimitExpr := "event.data.customerID"
				fnConfig := inngest.Function{
					ID:              fnID,
					FunctionVersion: 1,
					RateLimit: &inngest.RateLimit{
						Limit:  10,
						Period: "60s",
						Key:    &rateLimitExpr,
					},
					Name: "test function",
					Slug: "test-function",
					Triggers: inngest.MultipleTriggers{
						inngest.Trigger{
							EventTrigger: &inngest.EventTrigger{
								Event: "test/event",
							},
							CronTrigger: nil,
						},
					},
				}

				md, err := deps.exec.Schedule(ctx, execution.ScheduleRequest{
					Function:       fnConfig,
					AccountID:      accountID,
					WorkspaceID:    envID,
					AppID:          appID,
					IdempotencyKey: &idempotencyKey,
					Events: []event.TrackedEvent{
						event.NewBaseTrackedEventWithID(event.Event{
							Name: "test/event",
							Data: map[string]any{
								"customerID": "user1",
							},
						}, eventID),
					},
				})
				require.NoError(t, err)
				require.NotNil(t, md)
				require.NotNil(t, md.ID.RunID)

				rateLimitKeyHash := util.XXHash("user1")
				keyRateLimitState := fmt.Sprintf("{rl}:%s-%s", fnID, rateLimitKeyHash)
				require.True(t, deps.r.Exists(keyRateLimitState))

				raw, err := deps.r.Get(keyRateLimitState)
				require.NoError(t, err)
				parsed, err := strconv.Atoi(raw)
				require.NoError(t, err)
				tat := time.Unix(0, int64(parsed))
				require.WithinDuration(t, deps.clock.Now().Add(6*time.Second), tat, time.Second)
			},
			afterPreAcquireCheck: func(t *testing.T, deps *deps, resp *constraintapi.CapacityCheckResponse) { // Usage should already be visible in check
				t.Log(resp.Debug())
				require.Len(t, resp.Usage, 1)
				require.Equal(t, constraintapi.ConstraintKindRateLimit, resp.Usage[0].Constraint.Kind)
				require.Equal(t, 1, resp.Usage[0].Used)
				require.Equal(t, 10, resp.Usage[0].Limit)
			},
			afterAcquire: func(t *testing.T, deps *deps, resp *constraintapi.CapacityAcquireResponse) {
				t.Log(resp.Debug())

				require.Len(t, resp.LimitingConstraints, 0)
			},
			afterPostAcquireCheck: func(t *testing.T, deps *deps, resp *constraintapi.CapacityCheckResponse) {
				require.Len(t, resp.Usage, 1)
				require.Equal(t, constraintapi.ConstraintKindRateLimit, resp.Usage[0].Constraint.Kind)
				require.Equal(t, 2, resp.Usage[0].Used)
				require.Equal(t, 10, resp.Usage[0].Limit)
			},
		},
	}

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
				constraintapi.WithRateLimitClient(rc),
				constraintapi.WithQueueShards(map[string]rueidis.Client{
					"test": rc,
				}),
				constraintapi.WithClock(clock),
				constraintapi.WithNumScavengerShards(1),
				constraintapi.WithQueueStateKeyPrefix("q:v1"),
				constraintapi.WithRateLimitKeyPrefix("rl"),
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
			queueOptions := queue.NewQueueOptions(ctx, queueOpts...)
			shard := redis_state.NewRedisQueue(*queueOptions, "test", redis_state.NewQueueClient(rc, "q:v1"))

			q, err := queue.NewQueueProcessor(
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

			sm, err := redis_state.New(ctx,
				redis_state.WithShardedClient(sharded),
				redis_state.WithUnshardedClient(unsharded),
			)
			require.NoError(t, err)
			exec, err := executor.NewExecutor(
				executor.WithRateLimiter(rl),
				executor.WithAssignedQueueShard(shard),
				executor.WithQueue(q),
				executor.WithStateManager(redis_state.MustRunServiceV2(sm)),
				executor.WithPauseManager(pauses.NewRedisOnlyManager(sm)),
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
				Migration:     test.mi,
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
				Migration:            test.mi,
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
				Migration:     test.mi,
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
					Migration:      test.mi,
				})
				require.NoError(t, err)

				if test.afterExtend != nil {
					test.afterExtend(t, deps, extendResp)
				}

				releaseResp, err := cm.Release(ctx, &constraintapi.CapacityReleaseRequest{
					AccountID:      accountID,
					IdempotencyKey: "release",
					Migration:      test.mi,
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

// TestQueueConstraintAPICompatibility ensures the current queue implementation is compatible with the Constraint API
func TestQueueConstraintAPICompatibility(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	t.Run("queue should check in progress leases during Lease", func(t *testing.T) {
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

		kg := redis_state.NewQueueClient(nil, "q:v1").KeyGenerator()

		config := constraintapi.ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: constraintapi.ConcurrencyConfig{
				AccountConcurrency: 10,
			},
		}
		constraints := []constraintapi.ConstraintItem{
			{
				Kind: constraintapi.ConstraintKindConcurrency,
				Concurrency: &constraintapi.ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeAccount,
					Mode:              enums.ConcurrencyModeStep,
					InProgressItemKey: kg.Concurrency("account", accountID.String()),
				},
			},
		}
		partitionConstraints := queue.PartitionConstraintConfig{
			Concurrency: queue.PartitionConcurrency{
				AccountConcurrency: 10,
			},
		}

		cm, err := constraintapi.NewRedisCapacityManager(
			constraintapi.WithRateLimitClient(rc),
			constraintapi.WithQueueShards(map[string]rueidis.Client{
				"test": rc,
			}),
			constraintapi.WithClock(clock),
			constraintapi.WithNumScavengerShards(1),
			constraintapi.WithQueueStateKeyPrefix("q:v1"),
			constraintapi.WithRateLimitKeyPrefix("rl"),
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
				return partitionConstraints
			}),
		}
		queueOptions := queue.NewQueueOptions(ctx, queueOpts...)
		shard := redis_state.NewRedisQueue(*queueOptions, "test", redis_state.NewQueueClient(rc, "q:v1"))

		amount := 10

		leaseIdempotencyKeys := make([]string, amount)
		for i := range amount {
			leaseIdempotencyKeys[i] = fmt.Sprintf("item%d", i)
		}

		// Claim concurrency capacity
		acquireResp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			Migration: constraintapi.MigrationIdentifier{
				QueueShard: "test",
			},
			AccountID:            accountID,
			IdempotencyKey:       "acquire",
			Constraints:          constraints,
			Amount:               amount,
			EnvID:                envID,
			FunctionID:           fnID,
			Configuration:        config,
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
		require.Len(t, acquireResp.Leases, 10)

		// Leasing should fail
		for i := range 1 {
			// Simulate existing throttle usage
			qi, err := shard.EnqueueItem(
				context.Background(),
				queue.QueueItem{
					ID:          fmt.Sprintf("item%d", i),
					FunctionID:  fnID,
					WorkspaceID: envID,
					Data: queue.Item{
						WorkspaceID: envID,
						Kind:        queue.KindStart,
						Identifier: state.Identifier{
							AccountID:   accountID,
							WorkspaceID: envID,
							WorkflowID:  fnID,
						},
					},
				},
				clock.Now(),
				queue.EnqueueOpts{},
			)
			require.NoError(t, err)
			require.NotNil(t, qi)

			leaseID, err := shard.Lease(context.Background(), qi, 5*time.Second, clock.Now(), nil)
			require.Error(t, err)
			require.Nil(t, leaseID)
			require.ErrorContains(t, err, "at account concurrency limit")
		}
	})
	t.Run("queue should check in progress leases during PartitionLease", func(t *testing.T) {
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

		kg := redis_state.NewQueueClient(nil, "q:v1").KeyGenerator()

		config := constraintapi.ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: constraintapi.ConcurrencyConfig{
				AccountConcurrency: 10,
			},
		}
		constraints := []constraintapi.ConstraintItem{
			{
				Kind: constraintapi.ConstraintKindConcurrency,
				Concurrency: &constraintapi.ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeAccount,
					Mode:              enums.ConcurrencyModeStep,
					InProgressItemKey: kg.Concurrency("account", accountID.String()),
				},
			},
		}
		partitionConstraints := queue.PartitionConstraintConfig{
			Concurrency: queue.PartitionConcurrency{
				AccountConcurrency: 10,
			},
		}

		cm, err := constraintapi.NewRedisCapacityManager(
			constraintapi.WithRateLimitClient(rc),
			constraintapi.WithQueueShards(map[string]rueidis.Client{
				"test": rc,
			}),
			constraintapi.WithClock(clock),
			constraintapi.WithNumScavengerShards(1),
			constraintapi.WithQueueStateKeyPrefix("q:v1"),
			constraintapi.WithRateLimitKeyPrefix("rl"),
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
				return partitionConstraints
			}),
		}
		queueOptions := queue.NewQueueOptions(ctx, queueOpts...)
		shard := redis_state.NewRedisQueue(*queueOptions, "test", redis_state.NewQueueClient(rc, "q:v1"))

		amount := 10

		leaseIdempotencyKeys := make([]string, amount)
		for i := range amount {
			leaseIdempotencyKeys[i] = fmt.Sprintf("item%d", i)
		}

		// Claim concurrency capacity
		acquireResp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			Migration: constraintapi.MigrationIdentifier{
				QueueShard: "test",
			},
			AccountID:            accountID,
			IdempotencyKey:       "acquire",
			Constraints:          constraints,
			Amount:               amount,
			EnvID:                envID,
			FunctionID:           fnID,
			Configuration:        config,
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
		require.Len(t, acquireResp.Leases, 10)

		qi, err := shard.EnqueueItem(
			context.Background(),
			queue.QueueItem{
				ID:          "item0",
				FunctionID:  fnID,
				WorkspaceID: envID,
				Data: queue.Item{
					WorkspaceID: envID,
					Kind:        queue.KindStart,
					Identifier: state.Identifier{
						AccountID:   accountID,
						WorkspaceID: envID,
						WorkflowID:  fnID,
					},
				},
			},
			clock.Now(),
			queue.EnqueueOpts{},
		)
		require.NoError(t, err)
		require.NotNil(t, qi)

		p := queue.ItemPartition(ctx, qi)

		// Reject when enforcing checks -> This verifies PartitionLease also considers in progress leases.
		// This is relevant during a rollout or when enabling the feature flag for an account, as propagation
		// may take a while. During this time, executors must respect the leases generated by other executors using the Constraint API already.
		leaseID, capacity, err := shard.PartitionLease(ctx, &p, 5*time.Second)
		require.Error(t, err)
		require.ErrorContains(t, err, "at account concurrency limit")
		require.Nil(t, leaseID)
		require.Zero(t, capacity)

		// Allow when skipping checks
		leaseID, _, err = shard.PartitionLease(ctx, &p, 5*time.Second, queue.PartitionLeaseOptionDisableLeaseChecks(true))
		require.NoError(t, err)
		require.NotNil(t, leaseID)
	})

	t.Run("queue should ignore GCRA during Lease if idempotency key set", func(t *testing.T) {
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

		kg := redis_state.NewQueueClient(nil, "q:v1").KeyGenerator()

		config := constraintapi.ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: constraintapi.ConcurrencyConfig{
				AccountConcurrency: 10,
			},
			Throttle: []constraintapi.ThrottleConfig{
				{
					KeyExpressionHash: "expr-hash",
					Limit:             5,
					Period:            60,
				},
			},
		}
		constraints := []constraintapi.ConstraintItem{
			{
				Kind: constraintapi.ConstraintKindConcurrency,
				Concurrency: &constraintapi.ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeAccount,
					Mode:              enums.ConcurrencyModeStep,
					InProgressItemKey: kg.Concurrency("account", accountID.String()),
				},
			},
			{
				Kind: constraintapi.ConstraintKindThrottle,
				Throttle: &constraintapi.ThrottleConstraint{
					KeyExpressionHash: "expr-hash",
					EvaluatedKeyHash:  "key-hash",
				},
			},
		}

		throttle := &queue.Throttle{
			KeyExpressionHash: "expr-hash",
			Key:               "key-hash",
			Limit:             5,
			Period:            60,
		}

		partitionConstraints := queue.PartitionConstraintConfig{
			Concurrency: queue.PartitionConcurrency{
				AccountConcurrency: 10,
			},
			Throttle: &queue.PartitionThrottle{
				ThrottleKeyExpressionHash: "expr-hash",
				Limit:                     5,
				Period:                    60,
			},
		}

		cm, err := constraintapi.NewRedisCapacityManager(
			constraintapi.WithRateLimitClient(rc),
			constraintapi.WithQueueShards(map[string]rueidis.Client{
				"test": rc,
			}),
			constraintapi.WithClock(clock),
			constraintapi.WithNumScavengerShards(1),
			constraintapi.WithQueueStateKeyPrefix("q:v1"),
			constraintapi.WithRateLimitKeyPrefix("rl"),
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
				return partitionConstraints
			}),
		}
		queueOptions := queue.NewQueueOptions(ctx, queueOpts...)
		shard := redis_state.NewRedisQueue(*queueOptions, "test", redis_state.NewQueueClient(rc, "q:v1"))

		qi, err := shard.EnqueueItem(
			context.Background(),
			queue.QueueItem{
				ID:          "item0",
				FunctionID:  fnID,
				WorkspaceID: envID,
				Data: queue.Item{
					WorkspaceID: envID,
					Kind:        queue.KindStart,
					Identifier: state.Identifier{
						AccountID:   accountID,
						WorkspaceID: envID,
						WorkflowID:  fnID,
					},
					Throttle: throttle,
				},
			},
			clock.Now(),
			queue.EnqueueOpts{},
		)
		require.NoError(t, err)
		require.NotNil(t, qi)

		qi2, err := shard.EnqueueItem(
			context.Background(),
			queue.QueueItem{
				ID:          "other-item",
				FunctionID:  fnID,
				WorkspaceID: envID,
				Data: queue.Item{
					WorkspaceID: envID,
					Kind:        queue.KindStart,
					Identifier: state.Identifier{
						AccountID:   accountID,
						WorkspaceID: envID,
						WorkflowID:  fnID,
					},
					Throttle: throttle,
				},
			},
			clock.Now(),
			queue.EnqueueOpts{},
		)
		require.NoError(t, err)
		require.NotNil(t, qi)

		amount := 4
		leaseIdempotencyKeys := make([]string, amount)
		for i := range amount {
			leaseIdempotencyKeys[i] = util.XXHash(fmt.Sprintf("item%d", i))
		}

		// Claim concurrency capacity
		acquireResp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			Migration: constraintapi.MigrationIdentifier{
				QueueShard: "test",
			},
			AccountID:            accountID,
			IdempotencyKey:       "acquire",
			Constraints:          constraints,
			Amount:               amount,
			EnvID:                envID,
			FunctionID:           fnID,
			Configuration:        config,
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
		require.Len(t, acquireResp.Leases, amount)

		keyThrottleState := kg.ThrottleKey(qi.Data.Throttle)
		require.True(t, r.Exists(keyThrottleState))

		// Expect throttle state to match
		rawState, err := r.Get(keyThrottleState)
		require.NoError(t, err)
		parsed, err := strconv.Atoi(rawState)
		require.NoError(t, err)
		tat := time.UnixMilli(int64(parsed))
		require.WithinDuration(t, clock.Now().Add(time.Duration(amount)*12*time.Second), tat, time.Second)

		// Expect constraint check idempotency to be set
		keyConstraintCheckIdempotency := cm.KeyConstraintCheckIdempotency(constraintapi.MigrationIdentifier{
			QueueShard: shard.Name(),
		}, accountID, util.XXHash("item0"))
		require.True(t, r.Exists(keyConstraintCheckIdempotency))

		// Should work because we handle constraint check idempotency
		leaseID, err := shard.Lease(ctx, qi, 5*time.Second, clock.Now(), nil)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		// Expect throttle state to match
		rawState, err = r.Get(keyThrottleState)
		require.NoError(t, err)
		parsed, err = strconv.Atoi(rawState)
		require.NoError(t, err)
		tat = time.UnixMilli(int64(parsed))
		require.WithinDuration(t, clock.Now().Add(time.Duration(amount)*12*time.Second), tat, time.Second)

		// Should not work because item does not have constraint check idempotency set
		// NOTE: This is allowed for the time being as the legacy gcra() implementation uses period_ms for `variance`
		// As soon as we migrate gcra() to use `emission` instead of `period_ms`, we will be limited as expected.
		// For more details, see https://github.com/inngest/inngest/pull/3356
		leaseID, err = shard.Lease(ctx, qi2, 5*time.Second, clock.Now(), nil)
		// TODO: Once gcra() is fixed and this test fails, remove the following assertions and use the commented out assertions instead
		require.NoError(t, err)
		require.NotNil(t, leaseID)
		// require.NoError(t, err, r.Dump())
		// require.ErrorIs(t, err, redis_state.ErrQueueItemThrottled)
		// require.Nil(t, leaseID)
	})
	t.Run("queue should ignore GCRA during BacklogRefill if idempotency key set", func(t *testing.T) {
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

		kg := redis_state.NewQueueClient(nil, "q:v1").KeyGenerator()

		config := constraintapi.ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: constraintapi.ConcurrencyConfig{
				AccountConcurrency: 10,
			},
			Throttle: []constraintapi.ThrottleConfig{
				{
					KeyExpressionHash: "expr-hash",
					Limit:             5,
					Period:            60,
				},
			},
		}
		constraints := []constraintapi.ConstraintItem{
			{
				Kind: constraintapi.ConstraintKindConcurrency,
				Concurrency: &constraintapi.ConcurrencyConstraint{
					Scope:             enums.ConcurrencyScopeAccount,
					Mode:              enums.ConcurrencyModeStep,
					InProgressItemKey: kg.Concurrency("account", accountID.String()),
				},
			},
			{
				Kind: constraintapi.ConstraintKindThrottle,
				Throttle: &constraintapi.ThrottleConstraint{
					KeyExpressionHash: "expr-hash",
					EvaluatedKeyHash:  "key-hash",
				},
			},
		}
		partitionConstraints := queue.PartitionConstraintConfig{
			FunctionVersion: 1,
			Concurrency: queue.PartitionConcurrency{
				AccountConcurrency: 10,
			},
			Throttle: &queue.PartitionThrottle{
				ThrottleKeyExpressionHash: "expr-hash",
				Limit:                     5,
				Period:                    60,
			},
		}

		cm, err := constraintapi.NewRedisCapacityManager(
			constraintapi.WithRateLimitClient(rc),
			constraintapi.WithQueueShards(map[string]rueidis.Client{
				"test": rc,
			}),
			constraintapi.WithClock(clock),
			constraintapi.WithNumScavengerShards(1),
			constraintapi.WithQueueStateKeyPrefix("q:v1"),
			constraintapi.WithRateLimitKeyPrefix("rl"),
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
				return partitionConstraints
			}),
			queue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
		}
		queueOptions := queue.NewQueueOptions(ctx, queueOpts...)
		shard := redis_state.NewRedisQueue(*queueOptions, "test", redis_state.NewQueueClient(rc, "q:v1"))

		amount := 4

		leaseIdempotencyKeys := make([]string, amount)
		for i := range amount {
			leaseIdempotencyKeys[i] = fmt.Sprintf("item%d", i)
		}

		// Claim concurrency capacity
		acquireResp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			Migration: constraintapi.MigrationIdentifier{
				QueueShard: shard.Name(),
			},
			AccountID:            accountID,
			IdempotencyKey:       "acquire",
			Constraints:          constraints,
			Amount:               amount,
			EnvID:                envID,
			FunctionID:           fnID,
			Configuration:        config,
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
		require.Len(t, acquireResp.Leases, amount)

		qi, err := shard.EnqueueItem(
			context.Background(),
			queue.QueueItem{
				ID:          "item0",
				FunctionID:  fnID,
				WorkspaceID: envID,
				Data: queue.Item{
					WorkspaceID: envID,
					Kind:        queue.KindStart,
					Identifier: state.Identifier{
						AccountID:   accountID,
						WorkspaceID: envID,
						WorkflowID:  fnID,
					},
					Throttle: &queue.Throttle{
						KeyExpressionHash: "expr-hash",
						Key:               "key-hash",
						Limit:             5,
						Period:            60,
					},
				},
			},
			clock.Now(),
			queue.EnqueueOpts{},
		)
		require.NoError(t, err)
		require.NotNil(t, qi)

		qi2, err := shard.EnqueueItem(
			context.Background(),
			queue.QueueItem{
				ID:          "other-item",
				FunctionID:  fnID,
				WorkspaceID: envID,
				Data: queue.Item{
					WorkspaceID: envID,
					Kind:        queue.KindStart,
					Identifier: state.Identifier{
						AccountID:   accountID,
						WorkspaceID: envID,
						WorkflowID:  fnID,
					},
					Throttle: &queue.Throttle{
						KeyExpressionHash: "expr-hash",
						Key:               "key-hash",
						Limit:             5,
						Period:            60,
					},
				},
			},
			clock.Now(),
			queue.EnqueueOpts{},
		)
		require.NoError(t, err)
		require.NotNil(t, qi2)

		keyThrottleState := kg.ThrottleKey(qi.Data.Throttle)
		require.True(t, r.Exists(keyThrottleState))

		// Expect throttle state to match
		rawState, err := r.Get(keyThrottleState)
		require.NoError(t, err)
		parsed, err := strconv.Atoi(rawState)
		require.NoError(t, err)
		tat := time.UnixMilli(int64(parsed))
		require.WithinDuration(t, clock.Now().Add(time.Duration(amount)*12*time.Second), tat, time.Second)

		// Expect constraint check idempotency to be set
		keyConstraintCheckIdempotency := cm.KeyConstraintCheckIdempotency(constraintapi.MigrationIdentifier{
			QueueShard: shard.Name(),
		}, accountID, "item0")
		require.True(t, r.Exists(keyConstraintCheckIdempotency))

		b := queue.ItemBacklog(ctx, qi)
		require.NotNil(t, b.Throttle)

		sp := queue.ItemShadowPartition(ctx, qi)

		// Item should be refilled
		res, err := shard.BacklogRefill(ctx, &b, &sp, clock.Now().Add(5*time.Second), []string{qi.ID}, partitionConstraints)
		require.NoError(t, err)
		require.Equal(t, 1, res.Refill)
		require.Equal(t, 1, res.Refilled)
		require.Equal(t, qi.ID, res.RefilledItems[0])

		// Other item should fail
		res, err = shard.BacklogRefill(ctx, &b, &sp, clock.Now().Add(5*time.Second), []string{qi2.ID}, partitionConstraints)
		require.NoError(t, err)
		require.Equal(t, enums.QueueConstraintThrottle, res.Constraint)
		require.Equal(t, 0, res.Refill)
		require.Equal(t, 0, res.Refilled)

		// Should succeed when skipping constraint checks
		res, err = shard.BacklogRefill(
			ctx,
			&b,
			&sp,
			clock.Now().Add(5*time.Second),
			[]string{qi2.ID},
			partitionConstraints,
			queue.WithBacklogRefillDisableConstraintChecks(true),
		)
		require.NoError(t, err)
		require.Equal(t, enums.QueueConstraintNotLimited, res.Constraint)
		require.Equal(t, 1, res.Refill)
		require.Equal(t, 1, res.Refilled)
	})
}

// TestScheduleConstraintCompatibility ensures Schedule() is compatible with the Constraint API
func TestScheduleConstraintAPICompatibility(t *testing.T) {
	t.Run("rate limit should ignore GCRA if idempotency key set", func(t *testing.T) {
		// enforce gcra during acquire which sets constraint check idempotency key
		// run rate limit with same constraint check idempotency key and verify it's ignored

		queueConstraints := queue.PartitionConstraintConfig{
			FunctionVersion: 1,
		}

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
			constraintapi.WithRateLimitClient(rc),
			constraintapi.WithQueueShards(map[string]rueidis.Client{
				"test": rc,
			}),
			constraintapi.WithClock(clock),
			constraintapi.WithNumScavengerShards(1),
			constraintapi.WithQueueStateKeyPrefix("q:v1"),
			constraintapi.WithRateLimitKeyPrefix("rl"),
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
				return queueConstraints
			}),
		}
		queueOptions := queue.NewQueueOptions(ctx, queueOpts...)
		shard := redis_state.NewRedisQueue(*queueOptions, "test", redis_state.NewQueueClient(rc, "q:v1"))

		q, err := queue.NewQueueProcessor(
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

		sm, err := redis_state.New(ctx,
			redis_state.WithShardedClient(sharded),
			redis_state.WithUnshardedClient(unsharded),
		)
		require.NoError(t, err)
		exec, err := executor.NewExecutor(
			executor.WithRateLimiter(rl),
			executor.WithAssignedQueueShard(shard),
			executor.WithQueue(q),
			executor.WithStateManager(redis_state.MustRunServiceV2(sm)),
			executor.WithPauseManager(pauses.NewRedisOnlyManager(sm)),
			executor.WithCapacityManager(cm),
			executor.WithLogger(logger.StdlibLogger(ctx)),
			executor.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				return false, false
			}),
			executor.WithClock(clock),
		)
		require.NoError(t, err)

		eventID := ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader)
		accountID, envID, fnID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		scheduleIdempotencyKey := eventID.String()

		expr := "event.data.customerID"
		val := "user1"

		rlKey := fmt.Sprintf("%s-%s", fnID, util.XXHash(val))

		// Simulate idempotency key generated during Schedule
		leaseIdempotencyKey := fmt.Sprintf("%s-%s", util.XXHash(fnID.String()), util.XXHash(eventID.String()))

		res, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			IdempotencyKey: "acquire",
			AccountID:      accountID,
			EnvID:          envID,
			FunctionID:     fnID,
			Configuration: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				RateLimit: []constraintapi.RateLimitConfig{
					{
						KeyExpressionHash: util.XXHash(expr),
						Limit:             5,
						Period:            60,
					},
				},
			},
			Constraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindRateLimit,
					RateLimit: &constraintapi.RateLimitConstraint{
						KeyExpressionHash: util.XXHash(expr),
						EvaluatedKeyHash:  rlKey,
					},
				},
			},
			Amount:               3,
			LeaseIdempotencyKeys: []string{leaseIdempotencyKey, "item2", "item3"},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			Source: constraintapi.LeaseSource{
				Service:           constraintapi.ServiceAPI,
				Location:          constraintapi.CallerLocationSchedule,
				RunProcessingMode: constraintapi.RunProcessingModeBackground,
			},
			MaximumLifetime: time.Minute,
			Migration: constraintapi.MigrationIdentifier{
				IsRateLimit: true,
			},
		})
		require.NoError(t, err)
		require.Len(t, res.Leases, 1)

		// Idempotency key should be set
		keyConstraintCheckIdempotency := cm.KeyConstraintCheckIdempotency(constraintapi.MigrationIdentifier{IsRateLimit: true}, accountID, leaseIdempotencyKey)
		require.True(t, r.Exists(keyConstraintCheckIdempotency))

		// Rate limit should be allowed with idempotency
		rlRes, err := rl.RateLimit(ctx, rlKey, inngest.RateLimit{
			Limit:  5,
			Period: "60s",
			Key:    &expr,
		},
			ratelimit.WithIdempotency(keyConstraintCheckIdempotency, 5*time.Second),
		)
		require.NoError(t, err)
		require.True(t, rlRes.IdempotencyHit)
		require.False(t, rlRes.Limited)

		// Try as part of schedule
		md, err := exec.Schedule(ctx, execution.ScheduleRequest{
			AccountID:      accountID,
			WorkspaceID:    envID,
			AppID:          appID,
			IdempotencyKey: &scheduleIdempotencyKey,
			Function: inngest.Function{
				FunctionVersion: 1,
				ID:              fnID,
				RateLimit: &inngest.RateLimit{
					Key:    &expr,
					Limit:  5,
					Period: "60s",
				},
				Name: "test function",
				Slug: "test-fn",
				Triggers: inngest.MultipleTriggers{
					inngest.Trigger{
						EventTrigger: &inngest.EventTrigger{
							Event: "test/event",
						},
						CronTrigger: nil,
					},
				},
			},
			Events: []event.TrackedEvent{
				event.NewBaseTrackedEventWithID(event.Event{
					Name: "test/event",
					Data: map[string]any{
						"customerID": val,
					},
				}, eventID),
			},
		})
		require.NoError(t, err)
		require.NotNil(t, md)
	})

	t.Run("rate limit should gracefully use state set by Constraint API", func(t *testing.T) {
		// enforce gcra during acquire which sets constraint check idempotency key
		// run rate limit on different idempotency key and check it still uses the same state
		// this verifies we correctly and consistently enforce rate limits while we are rolling out or rolling back the Constraint API and it's partially used

		queueConstraints := queue.PartitionConstraintConfig{
			FunctionVersion: 1,
		}

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
			constraintapi.WithRateLimitClient(rc),
			constraintapi.WithQueueShards(map[string]rueidis.Client{
				"test": rc,
			}),
			constraintapi.WithClock(clock),
			constraintapi.WithNumScavengerShards(1),
			constraintapi.WithQueueStateKeyPrefix("q:v1"),
			constraintapi.WithRateLimitKeyPrefix("rl"),
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
				return queueConstraints
			}),
		}
		queueOptions := queue.NewQueueOptions(ctx, queueOpts...)
		shard := redis_state.NewRedisQueue(*queueOptions, "test", redis_state.NewQueueClient(rc, "q:v1"))

		q, err := queue.NewQueueProcessor(
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

		sm, err := redis_state.New(ctx,
			redis_state.WithShardedClient(sharded),
			redis_state.WithUnshardedClient(unsharded),
		)
		require.NoError(t, err)
		exec, err := executor.NewExecutor(
			executor.WithRateLimiter(rl),
			executor.WithAssignedQueueShard(shard),
			executor.WithQueue(q),
			executor.WithStateManager(redis_state.MustRunServiceV2(sm)),
			executor.WithPauseManager(pauses.NewRedisOnlyManager(sm)),
			executor.WithCapacityManager(cm),
			executor.WithLogger(logger.StdlibLogger(ctx)),
			executor.WithUseConstraintAPI(func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool, fallback bool) {
				return false, false
			}),
			executor.WithClock(clock),
		)
		require.NoError(t, err)

		eventID := ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader)
		accountID, envID, fnID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

		expr := "event.data.customerID"
		val := "user1"

		rlKey := fmt.Sprintf("%s-%s", fnID, util.XXHash(val))

		// Simulate idempotency key generated during Schedule
		leaseIdempotencyKey := fmt.Sprintf("%s-%s", util.XXHash(fnID.String()), util.XXHash(eventID.String()))

		res, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
			IdempotencyKey: "acquire",
			AccountID:      accountID,
			EnvID:          envID,
			FunctionID:     fnID,
			Configuration: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				RateLimit: []constraintapi.RateLimitConfig{
					{
						KeyExpressionHash: util.XXHash(expr),
						Limit:             10,
						Period:            60,
					},
				},
			},
			Constraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindRateLimit,
					RateLimit: &constraintapi.RateLimitConstraint{
						KeyExpressionHash: util.XXHash(expr),
						EvaluatedKeyHash:  rlKey,
					},
				},
			},
			Amount:               1,
			LeaseIdempotencyKeys: []string{leaseIdempotencyKey},
			CurrentTime:          clock.Now(),
			Duration:             5 * time.Second,
			Source: constraintapi.LeaseSource{
				Service:           constraintapi.ServiceAPI,
				Location:          constraintapi.CallerLocationSchedule,
				RunProcessingMode: constraintapi.RunProcessingModeBackground,
			},
			MaximumLifetime: time.Minute,
			Migration: constraintapi.MigrationIdentifier{
				IsRateLimit: true,
			},
		})
		require.NoError(t, err)
		require.Len(t, res.Leases, 1)

		// Idempotency key should be set
		keyConstraintCheckIdempotency := cm.KeyConstraintCheckIdempotency(constraintapi.MigrationIdentifier{IsRateLimit: true}, accountID, leaseIdempotencyKey)
		require.True(t, r.Exists(keyConstraintCheckIdempotency))

		// Rate limit should be allowed with idempotency
		rlRes, err := rl.RateLimit(ctx, rlKey, inngest.RateLimit{
			Limit:  10,
			Period: "60s",
			Key:    &expr,
		},
			ratelimit.WithIdempotency(keyConstraintCheckIdempotency, 5*time.Second),
		)
		require.NoError(t, err)
		require.True(t, rlRes.IdempotencyHit)
		require.False(t, rlRes.Limited)

		// Check rate limit state was set
		keyRateLimitState := fmt.Sprintf("{rl}:%s", fmt.Sprintf("%s-%s", fnID.String(), util.XXHash(val)))

		raw, err := r.Get(keyRateLimitState)
		require.NoError(t, err)
		parsed, err := strconv.Atoi(raw)
		require.NoError(t, err)
		tat := time.Unix(0, int64(parsed))
		require.WithinDuration(t, clock.Now().Add(6*time.Second), tat, time.Second)

		// Schedule should be allowed for second event with same key
		eventID2 := ulid.MustNew(ulid.Timestamp(clock.Now()), rand.Reader)
		scheduleIdempotencyKey := eventID2.String()
		md, err := exec.Schedule(ctx, execution.ScheduleRequest{
			AccountID:      accountID,
			WorkspaceID:    envID,
			AppID:          appID,
			IdempotencyKey: &scheduleIdempotencyKey,
			Function: inngest.Function{
				FunctionVersion: 1,
				ID:              fnID,
				RateLimit: &inngest.RateLimit{
					Key:    &expr,
					Limit:  10,
					Period: "60s",
				},
				Name: "test function",
				Slug: "test-fn",
				Triggers: inngest.MultipleTriggers{
					inngest.Trigger{
						EventTrigger: &inngest.EventTrigger{
							Event: "test/event",
						},
						CronTrigger: nil,
					},
				},
			},
			Events: []event.TrackedEvent{
				event.NewBaseTrackedEventWithID(event.Event{
					Name: "test/event",
					Data: map[string]any{
						"customerID": val,
					},
				}, eventID2),
			},
		})
		require.NoError(t, err)
		require.NotNil(t, md)

		// Ensure same state was updated
		raw, err = r.Get(keyRateLimitState)
		require.NoError(t, err)
		parsed, err = strconv.Atoi(raw)
		require.NoError(t, err)
		tat = time.Unix(0, int64(parsed))
		require.WithinDuration(t, clock.Now().Add(12*time.Second), tat, time.Second)
	})
}
