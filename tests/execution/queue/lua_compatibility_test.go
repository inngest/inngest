package queue

import (
	"context"
	"crypto/rand"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/tests/execution/queue/helper"
	"github.com/inngest/inngest/tests/testutil"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// getItemIDsFromBacklog is a helper function to peek items from a backlog and extract their IDs
func getItemIDsFromBacklog(ctx context.Context, mgr queue.ShardOperations, backlog *queue.QueueBacklog, refillUntil time.Time, limit int64) ([]string, error) {
	items, _, err := mgr.BacklogPeek(ctx, backlog, time.Time{}, refillUntil, limit)
	if err != nil {
		return nil, err
	}

	itemIDs := make([]string, len(items))
	for i, item := range items {
		itemIDs[i] = item.ID
	}
	return itemIDs, nil
}

// LuaCompatibilityTestCase defines a test case for Lua compatibility across different Redis-compatible servers
type LuaCompatibilityTestCase struct {
	Name       string                // Test case name
	ServerType string                // "valkey" or "garnet"
	ValkeyOpts []helper.ValkeyOption // Optional Valkey configuration
	GarnetOpts []helper.GarnetOption // Optional Garnet configuration
}

func TestLuaCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional tests")
	}

	testCases := []LuaCompatibilityTestCase{
		{
			Name:       "Basic Valkey",
			ServerType: "valkey",
			ValkeyOpts: []helper.ValkeyOption{
				helper.WithValkeyImage(testutil.ValkeyDefaultImage),
			},
		},
		{
			Name:       "Basic Garnet",
			ServerType: "garnet",
			GarnetOpts: []helper.GarnetOption{
				helper.WithImage(testutil.GarnetDefaultImage),
				helper.WithConfiguration(&helper.GarnetConfiguration{
					EnableLua: true,
				}),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()

			setup := func(t *testing.T, opts ...queue.QueueOpt) redis_state.RedisQueueShard {
				// Start the appropriate server based on test case
				var client rueidis.Client

				switch tc.ServerType {
				case "valkey":
					container, err := helper.StartValkey(t, tc.ValkeyOpts...)
					require.NoError(t, err)
					t.Cleanup(func() {
						_ = container.Terminate(ctx)
					})

					valkeyClient, err := helper.NewValkeyClient(container.Addr, container.Username, container.Password, false)
					require.NoError(t, err)
					t.Cleanup(func() {
						valkeyClient.Close()
					})

					client = valkeyClient

				case "garnet":
					container, err := helper.StartGarnet(t, tc.GarnetOpts...)
					require.NoError(t, err)

					t.Cleanup(func() {
						_ = container.Terminate(ctx)
					})

					garnetClient, err := helper.NewRedisClient(container.Addr, container.Username, container.Password)
					require.NoError(t, err)
					t.Cleanup(func() {
						garnetClient.Close()
					})

					client = garnetClient

				default:
					t.Fatalf("unknown server type: %s", tc.ServerType)
				}

				shard := redis_state.NewQueueShard(
					consts.DefaultQueueShardName,
					redis_state.NewQueueClient(client, redis_state.QueueDefaultKey),
					opts...,
				)

				return shard
			}

			serverType := tc.ServerType

			t.Run("basic operations", func(t *testing.T) {
				shard := setup(t)

				// Initialize queue
				q, err := queue.New(
					context.Background(),
					"test-queue",
					shard,
					map[string]queue.QueueShard{
						shard.Name(): shard,
					},
					func(ctx context.Context, accountId uuid.UUID, queueName *string) (queue.QueueShard, error) {
						return shard, nil
					})
				require.NoError(t, err)

				// Test data setup
				accountID := uuid.New()
				functionID := uuid.New()
				runID := ulid.Make()
				now := time.Now().Truncate(time.Second)

				// Create a queue item for testing
				queueItem := queue.QueueItem{
					FunctionID: functionID,
					Data: queue.Item{
						Kind: queue.KindStart,
						Identifier: state.Identifier{
							AccountID: accountID,
							RunID:     runID,
						},
					},
				}

				// - Enqueue item
				enqueuedItem, err := shard.EnqueueItem(ctx, queueItem, now, queue.EnqueueOpts{})
				require.NoError(t, err, "Failed to enqueue item on %s", serverType)
				require.NotEqual(t, enqueuedItem.ID, ulid.Zero, "Enqueued item should have valid ID on %s", serverType)
				require.Equal(t, enqueuedItem.FunctionID, functionID, "Function ID should match on %s", serverType)

				// - Peek partition
				partitions, err := shard.PartitionPeek(ctx, true, now.Add(time.Minute), 10)
				require.NoError(t, err)
				require.Len(t, partitions, 1)
				qp := partitions[0]

				// - Peek item
				peekedItems, err := shard.Peek(ctx, qp, now, 10)
				require.NoError(t, err, "Failed to peek partition on %s", serverType)
				require.NotEmpty(t, peekedItems, "Should find items in partition on %s", serverType)

				// Verify the item is in the peeked results
				var foundItem *queue.QueueItem
				for i := range peekedItems {
					if peekedItems[i].ID == enqueuedItem.ID {
						foundItem = peekedItems[i]
						break
					}
				}
				require.NotNil(t, foundItem, "Should find our enqueued item in partition on %s", serverType)
				require.Equal(t, foundItem.FunctionID, functionID, "Found item should have correct function ID on %s", serverType)

				// - Lease item
				leaseDuration := 30 * time.Second
				leaseID, err := shard.Lease(ctx, enqueuedItem, leaseDuration, now, nil)
				require.NoError(t, err, "Failed to lease item on %s", serverType)
				require.NotNil(t, leaseID, "Lease ID should not be nil on %s", serverType)

				// - Requeue item
				requeueTime := now.Add(5 * time.Second)
				err = q.Requeue(ctx, shard, enqueuedItem, requeueTime)
				require.NoError(t, err)

				// - Requeue partition
				err = shard.PartitionRequeue(ctx, qp, requeueTime, false)
				require.NoError(t, err, "Failed to requeue partition on %s", serverType)

				// Verify the item is available for leasing again after requeue
				peekedAfterRequeue, err := shard.Peek(ctx, qp, requeueTime.Add(5*time.Second), 10)
				require.NoError(t, err, "Failed to peek partition after requeue on %s", serverType)
				require.NotEmpty(t, peekedAfterRequeue, "Should find items in partition after requeue on %s", serverType)

				err = q.Dequeue(ctx, shard, enqueuedItem)
				require.NoError(t, err)

				peekedAfterDequeue, err := shard.Peek(ctx, qp, requeueTime.Add(5*time.Second), 10)
				require.NoError(t, err, "Failed to peek partition after requeue on %s", serverType)
				require.Empty(t, peekedAfterDequeue, "Should not find items in partition after dequeue on %s", serverType)
			})

			t.Run("lease with throttle", func(t *testing.T) {
				expr := "event.data.customerID"
				exprHash := util.XXHash(expr)

				opts := []queue.QueueOpt{
					queue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p queue.PartitionIdentifier) queue.PartitionConstraintConfig {
						return queue.PartitionConstraintConfig{
							Throttle: &queue.PartitionThrottle{
								ThrottleKeyExpressionHash: exprHash,
								Limit:                     5,
								Burst:                     0,
								Period:                    60,
							},
						}
					}),
				}

				shard := setup(t, opts...)

				// Test data setup
				accountID := uuid.New()
				functionID := uuid.New()
				runID := ulid.Make()
				now := time.Now().Truncate(time.Second)

				throttleKey := "customer-test"
				keyHash := util.XXHash(throttleKey)

				// Initialize queue
				_, err := queue.New(
					context.Background(),
					"test-queue",
					shard,
					map[string]queue.QueueShard{
						shard.Name(): shard,
					},
					func(ctx context.Context, accountId uuid.UUID, queueName *string) (queue.QueueShard, error) {
						return shard, nil
					},
					opts...,
				)
				require.NoError(t, err)

				// Create a queue item for testing
				queueItem := queue.QueueItem{
					FunctionID: functionID,
					Data: queue.Item{
						Kind: queue.KindStart,
						Identifier: state.Identifier{
							AccountID: accountID,
							RunID:     runID,
						},
						Throttle: &queue.Throttle{
							Limit:               5,
							Burst:               0,
							Period:              60,
							Key:                 keyHash,
							UnhashedThrottleKey: throttleKey,
							KeyExpressionHash:   exprHash,
						},
					},
				}

				qi, err := shard.EnqueueItem(ctx, queueItem, now, queue.EnqueueOpts{})
				require.NoError(t, err)

				leaseID, err := shard.Lease(ctx, qi, 5*time.Second, now, nil)
				require.NoError(t, err)
				require.NotNil(t, leaseID)
			})

			t.Run("backlog refill with throttle", func(t *testing.T) {
				// Test data setup
				accountID := uuid.New()
				functionID := uuid.New()
				runID := ulid.Make()
				now := time.Now().Truncate(time.Second)

				expr := "event.data.customerID"
				exprHash := util.XXHash(expr)
				throttleKey := "customer-test"
				keyHash := util.XXHash(throttleKey)

				constraints := queue.PartitionConstraintConfig{
					Throttle: &queue.PartitionThrottle{
						ThrottleKeyExpressionHash: exprHash,
						Limit:                     5,
						Burst:                     0,
						Period:                    60,
					},
				}

				opts := []queue.QueueOpt{
					queue.WithPartitionConstraintConfigGetter(func(ctx context.Context, p queue.PartitionIdentifier) queue.PartitionConstraintConfig {
						return constraints
					}),
					queue.WithAllowKeyQueues(func(ctx context.Context, acctID, fnID uuid.UUID) bool {
						return true
					}),
				}

				shard := setup(t, opts...)

				q, err := queue.New(
					context.Background(),
					"test-queue",
					shard,
					map[string]queue.QueueShard{
						shard.Name(): shard,
					},
					func(ctx context.Context, accountId uuid.UUID, queueName *string) (queue.QueueShard, error) {
						return shard, nil
					},
					opts...,
				)
				require.NoError(t, err)

				// Create a queue item for testing
				queueItem := queue.QueueItem{
					FunctionID: functionID,
					Data: queue.Item{
						Kind: queue.KindStart,
						Identifier: state.Identifier{
							AccountID: accountID,
							RunID:     runID,
						},
						Throttle: &queue.Throttle{
							Limit:               5,
							Burst:               0,
							Period:              60,
							Key:                 keyHash,
							UnhashedThrottleKey: throttleKey,
							KeyExpressionHash:   exprHash,
						},
					},
				}

				// Enqueue to backlog
				qi, err := shard.EnqueueItem(ctx, queueItem, now, queue.EnqueueOpts{})
				require.NoError(t, err)

				backlog := queue.ItemBacklog(ctx, qi)
				sp := queue.ItemShadowPartition(ctx, qi)

				// Use BacklogManager interface to peek items and get their IDs
				refillUntil := now.Add(time.Minute)
				refillItems, err := getItemIDsFromBacklog(ctx, shard, &backlog, refillUntil, 10)
				require.NoError(t, err)

				res, err := shard.BacklogRefill(ctx, &backlog, &sp, refillUntil, refillItems, constraints)
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, 1, res.Refill, *res)
				require.Equal(t, 1, res.Refilled)

				// Add second item with capacity lease
				capacityLeaseID := ulid.MustNew(ulid.Timestamp(refillUntil.Add(5*time.Second)), rand.Reader)
				item2 := queue.QueueItem{
					FunctionID: functionID,
					Data: queue.Item{
						Kind: queue.KindStart,
						Identifier: state.Identifier{
							AccountID: accountID,
							RunID:     runID,
						},
						Throttle: &queue.Throttle{
							Limit:               5,
							Burst:               0,
							Period:              60,
							Key:                 keyHash,
							UnhashedThrottleKey: throttleKey,
							KeyExpressionHash:   exprHash,
						},
					},
				}

				// Enqueue to backlog
				qi2, err := shard.EnqueueItem(ctx, item2, now, queue.EnqueueOpts{})
				require.NoError(t, err)

				refillItems, err = getItemIDsFromBacklog(ctx, shard, &backlog, refillUntil, 10)
				require.NoError(t, err)

				// Refill with capacity lease awareness
				res, err = shard.BacklogRefill(
					ctx,
					&backlog,
					&sp,
					refillUntil,
					refillItems,
					constraints,
					queue.WithBacklogRefillConstraintCheckIdempotencyKey("acquire-refill"),
					queue.WithBacklogRefillDisableConstraintChecks(true),
					queue.WithBacklogRefillItemCapacityLeases([]queue.CapacityLease{{
						LeaseID: capacityLeaseID,
					}}),
				)
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, 1, res.Refill)
				require.Equal(t, 1, res.Refilled)

				refilled, err := q.ItemByID(ctx, shard, qi2.ID)
				require.NoError(t, err)
				require.Equal(t, capacityLeaseID.String(), refilled.CapacityLease.LeaseID.String())
			})

			t.Run("current time is returned for rate limiting", func(t *testing.T) {
				shard := setup(t)

				rc := shard.Client().Client()

				cmd := rc.B().Time().Build()
				res, err := rc.Do(ctx, cmd).AsStrSlice()
				require.NoError(t, err)
				require.Len(t, res, 2)

				t.Log(res)

				parsed, err := strconv.Atoi(res[0])
				require.NoError(t, err)

				t.Log(res[0], parsed)

				parsed, err = strconv.Atoi(res[1])
				require.NoError(t, err)

				t.Log(res[1], parsed)
			})

			t.Run("acquiring capacity works", func(t *testing.T) {
				shard := setup(t)

				cm, err := constraintapi.NewRedisCapacityManager(
					constraintapi.WithClock(clockwork.NewRealClock()),
					constraintapi.WithEnableDebugLogs(false),
					constraintapi.WithNumScavengerShards(4),
					constraintapi.WithQueueShards(map[string]rueidis.Client{
						shard.Name(): shard.Client().Client(),
					}),
					constraintapi.WithQueueStateKeyPrefix("q:v1"),
					constraintapi.WithRateLimitClient(shard.Client().Client()),
					constraintapi.WithRateLimitKeyPrefix("rl"),
				)
				require.NoError(t, err)

				config := constraintapi.ConstraintConfig{
					FunctionVersion: 1,
					Concurrency: constraintapi.ConcurrencyConfig{
						AccountConcurrency:  10,
						FunctionConcurrency: 5,
					},
				}

				accountID := uuid.New()
				envID := uuid.New()
				functionID := uuid.New()

				constraints := []constraintapi.ConstraintItem{
					{
						Kind: constraintapi.ConstraintKindConcurrency,
						Concurrency: &constraintapi.ConcurrencyConstraint{
							Scope:             enums.ConcurrencyScopeAccount,
							Mode:              enums.ConcurrencyModeStep,
							InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
						},
					},
					{
						Kind: constraintapi.ConstraintKindConcurrency,
						Concurrency: &constraintapi.ConcurrencyConstraint{
							Scope:             enums.ConcurrencyScopeFn,
							Mode:              enums.ConcurrencyModeStep,
							InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:p:%s", functionID),
						},
					},
				}

				checkResp, userErr, internalErr := cm.Check(ctx, &constraintapi.CapacityCheckRequest{
					Migration: constraintapi.MigrationIdentifier{
						IsRateLimit: false,
						QueueShard:  shard.Name(),
					},
					AccountID:     accountID,
					EnvID:         envID,
					FunctionID:    functionID,
					Configuration: config,
					Constraints:   constraints,
				})
				require.NoError(t, internalErr)
				require.NoError(t, userErr)
				require.NotNil(t, checkResp)
				require.Equal(t, 5, checkResp.AvailableCapacity)
				require.Len(t, checkResp.Usage, 2)
				require.Equal(t, 10, checkResp.Usage[0].Limit)
				require.Equal(t, 0, checkResp.Usage[0].Used)
				require.Equal(t, 5, checkResp.Usage[1].Limit)
				require.Equal(t, 0, checkResp.Usage[1].Used)

				acquireResp, err := cm.Acquire(ctx, &constraintapi.CapacityAcquireRequest{
					Migration: constraintapi.MigrationIdentifier{
						IsRateLimit: false,
						QueueShard:  shard.Name(),
					},
					IdempotencyKey:       "acquire-test",
					AccountID:            accountID,
					EnvID:                envID,
					FunctionID:           functionID,
					Configuration:        config,
					Constraints:          constraints,
					Amount:               1,
					LeaseIdempotencyKeys: []string{"item1"},
					LeaseRunIDs:          nil,
					CurrentTime:          time.Now(),
					Duration:             5 * time.Second,
					MaximumLifetime:      time.Hour,
					Source: constraintapi.LeaseSource{
						Service:           constraintapi.ServiceAPI,
						Location:          constraintapi.CallerLocationItemLease,
						RunProcessingMode: constraintapi.RunProcessingModeDurableEndpoint,
					},
				})

				require.NoError(t, err)
				require.NotNil(t, acquireResp)
				require.Equal(t, 1, len(acquireResp.Leases))

				extendResp, err := cm.ExtendLease(ctx, &constraintapi.CapacityExtendLeaseRequest{
					Migration: constraintapi.MigrationIdentifier{
						IsRateLimit: false,
						QueueShard:  shard.Name(),
					},
					IdempotencyKey: "extend-test",
					AccountID:      accountID,
					Duration:       5 * time.Second,
					LeaseID:        acquireResp.Leases[0].LeaseID,
				})

				require.NoError(t, err)
				require.NotNil(t, extendResp)
				require.NotNil(t, extendResp.LeaseID)

				releaseResp, err := cm.Release(ctx, &constraintapi.CapacityReleaseRequest{
					Migration: constraintapi.MigrationIdentifier{
						IsRateLimit: false,
						QueueShard:  shard.Name(),
					},
					IdempotencyKey: "release-test",
					AccountID:      accountID,
					LeaseID:        *extendResp.LeaseID,
				})

				require.NoError(t, err)
				require.NotNil(t, releaseResp)
			})
		})
	}
}
