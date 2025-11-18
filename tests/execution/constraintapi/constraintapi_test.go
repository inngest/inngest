package constraintapi_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestConstraintEnforcement(t *testing.T) {
	accountID, envID, fnID := uuid.New(), uuid.New(), uuid.New()

	type deps struct {
		cm    constraintapi.RolloutManager
		clock clockwork.FakeClock
		r     *miniredis.Miniredis
		rc    rueidis.Client

		q     redis_state.QueueProcessor
		shard redis_state.QueueShard

		config      constraintapi.ConstraintConfig
		constraints []constraintapi.ConstraintItem
	}

	type testCase struct {
		name string

		amount      int
		config      constraintapi.ConstraintConfig
		constraints []constraintapi.ConstraintItem
		mi          constraintapi.MigrationIdentifier

		beforeAcquire func(t *testing.T, deps *deps)

		afterAcquire func(t *testing.T, deps *deps, resp *constraintapi.CapacityAcquireResponse)

		expectedLeaseAmount int

		afterExtend  func(t *testing.T, deps *deps, resp *constraintapi.CapacityExtendLeaseResponse)
		afterRelease func(t *testing.T, deps *deps, resp *constraintapi.CapacityReleaseResponse)
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
				q := deps.q
				// Simulate existing concurrency usage (in progress item Leased by queue)
				for i := range 5 { // 5/10
					qi, err := q.EnqueueItem(
						context.Background(),
						deps.shard,
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
				q := deps.q

				// Simulate existing concurrency usage (in progress item Leased by queue)
				for i := range 5 { // 5/10
					qi, err := q.EnqueueItem(
						context.Background(),
						deps.shard,
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
			// This test checks ensures that Throttl constraint state set in the queue is respected by the Constraint API
			name: "existing throttle should be respected",
			config: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Throttle: []constraintapi.ThrottleConfig{
					{
						Scope:                     enums.ThrottleScopeFn,
						Period:                    60,
						Limit:                     1,
						Burst:                     0,
						ThrottleKeyExpressionHash: "expr-hash",
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
			mi: constraintapi.MigrationIdentifier{
				QueueShard: "test",
			},
			beforeAcquire: func(t *testing.T, deps *deps) {
				clock := deps.clock
				q := deps.q
				r := deps.r

				// Simulate existing throttle usage
				qi, err := q.EnqueueItem(
					context.Background(),
					deps.shard,
					queue.QueueItem{
						ID:          "item1",
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
			},
			amount:              1,
			expectedLeaseAmount: 0,
			afterAcquire: func(t *testing.T, deps *deps, resp *constraintapi.CapacityAcquireResponse) {
				require.Len(t, resp.Leases, 0)
				require.Len(t, resp.LimitingConstraints, 1)
				require.Equal(t, "throttle-key", resp.LimitingConstraints[0].Throttle.EvaluatedKeyHash)
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

			defaultShard := redis_state.QueueShard{
				Kind:        string(enums.QueueShardKindRedis),
				Name:        "test",
				RedisClient: redis_state.NewQueueClient(rc, "q:v1"),
			}
			q := redis_state.NewQueue(defaultShard,
				redis_state.WithClock(clock),
				redis_state.WithQueueShardClients(map[string]redis_state.QueueShard{
					"test": defaultShard,
				}),
				redis_state.WithShardSelector(func(ctx context.Context, accountId uuid.UUID, queueName *string) (redis_state.QueueShard, error) {
					return defaultShard, nil
				}),
				redis_state.WithCapacityManager(cm),
				redis_state.WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (bool, bool) {
					return true, true
				}),
			)

			deps := &deps{
				config:      test.config,
				constraints: test.constraints,
				cm:          cm,
				clock:       clock,
				r:           r,
				rc:          rc,
				q:           q,
				shard:       defaultShard,
			}

			if test.beforeAcquire != nil {
				test.beforeAcquire(t, deps)
			}

			leaseIdempotencyKeys := make([]string, test.amount)
			for i := range test.amount {
				leaseIdempotencyKeys[i] = fmt.Sprintf("item%d", i)
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
					Location:          constraintapi.LeaseLocationItemLease,
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
