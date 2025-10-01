package queue

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/tests/execution/queue/helper"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

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
				helper.WithValkeyImage("valkey/valkey:8.0.1"),
			},
		},
		{
			Name:       "Basic Garnet",
			ServerType: "garnet",
			GarnetOpts: []helper.GarnetOption{
				helper.WithImage("ghcr.io/microsoft/garnet:1.0.84"),
				helper.WithConfiguration(&helper.GarnetConfiguration{
					EnableLua: true,
				}),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()

			setup := func(t *testing.T) redis_state.QueueShard {
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

				shard := redis_state.QueueShard{
					Kind:        string(enums.QueueShardKindRedis),
					RedisClient: redis_state.NewQueueClient(client, redis_state.QueueDefaultKey),
					Name:        consts.DefaultQueueShardName,
				}
				return shard
			}

			serverType := tc.ServerType

			t.Run("basic operations", func(t *testing.T) {
				shard := setup(t)

				// Initialize queue
				q := redis_state.NewQueue(shard)

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
				enqueuedItem, err := q.EnqueueItem(ctx, shard, queueItem, now, queue.EnqueueOpts{})
				require.NoError(t, err, "Failed to enqueue item on %s", serverType)
				require.NotEqual(t, enqueuedItem.ID, ulid.Zero, "Enqueued item should have valid ID on %s", serverType)
				require.Equal(t, enqueuedItem.FunctionID, functionID, "Function ID should match on %s", serverType)

				// - Peek partition
				partitions, err := q.PartitionPeek(ctx, true, now.Add(time.Minute), 10)
				require.NoError(t, err)
				require.Len(t, partitions, 1)
				qp := partitions[0]

				// - Peek item
				peekedItems, err := q.Peek(ctx, qp, now, 10)
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
				leaseID, err := q.Lease(ctx, enqueuedItem, leaseDuration, now, nil)
				require.NoError(t, err, "Failed to lease item on %s", serverType)
				require.NotNil(t, leaseID, "Lease ID should not be nil on %s", serverType)

				// - Requeue item
				requeueTime := now.Add(5 * time.Second)
				err = q.Requeue(ctx, shard, enqueuedItem, requeueTime)
				require.NoError(t, err)

				// - Requeue partition
				err = q.PartitionRequeue(ctx, shard, qp, requeueTime, false)
				require.NoError(t, err, "Failed to requeue partition on %s", serverType)

				// Verify the item is available for leasing again after requeue
				peekedAfterRequeue, err := q.Peek(ctx, qp, requeueTime.Add(5*time.Second), 10)
				require.NoError(t, err, "Failed to peek partition after requeue on %s", serverType)
				require.NotEmpty(t, peekedAfterRequeue, "Should find items in partition after requeue on %s", serverType)

				err = q.Dequeue(ctx, shard, enqueuedItem)
				require.NoError(t, err)

				peekedAfterDequeue, err := q.Peek(ctx, qp, requeueTime.Add(5*time.Second), 10)
				require.NoError(t, err, "Failed to peek partition after requeue on %s", serverType)
				require.Empty(t, peekedAfterDequeue, "Should not find items in partition after dequeue on %s", serverType)
			})

			t.Run("lease with throttle", func(t *testing.T) {
				shard := setup(t)

				// Test data setup
				accountID := uuid.New()
				functionID := uuid.New()
				runID := ulid.Make()
				now := time.Now().Truncate(time.Second)

				expr := "event.data.customerID"
				exprHash := util.XXHash(expr)
				throttleKey := "customer-test"
				keyHash := util.XXHash(throttleKey)

				// Initialize queue
				q := redis_state.NewQueue(shard,
					redis_state.WithPartitionConstraintConfigGetter(func(ctx context.Context, p redis_state.PartitionIdentifier) redis_state.PartitionConstraintConfig {
						return redis_state.PartitionConstraintConfig{
							Throttle: &redis_state.PartitionThrottle{
								ThrottleKeyExpressionHash: exprHash,
								Limit:                     5,
								Burst:                     0,
								Period:                    60,
							},
						}
					}))

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

				qi, err := q.EnqueueItem(ctx, shard, queueItem, now, queue.EnqueueOpts{})
				require.NoError(t, err)

				leaseID, err := q.Lease(ctx, qi, 5*time.Second, now, nil)
				require.NoError(t, err)
				require.NotNil(t, leaseID)
			})

			t.Run("backlog refill with throttle", func(t *testing.T) {
				shard := setup(t)

				// Test data setup
				accountID := uuid.New()
				functionID := uuid.New()
				runID := ulid.Make()
				now := time.Now().Truncate(time.Second)

				expr := "event.data.customerID"
				exprHash := util.XXHash(expr)
				throttleKey := "customer-test"
				keyHash := util.XXHash(throttleKey)

				constraints := redis_state.PartitionConstraintConfig{
					Throttle: &redis_state.PartitionThrottle{
						ThrottleKeyExpressionHash: exprHash,
						Limit:                     5,
						Burst:                     0,
						Period:                    60,
					},
				}

				// Initialize queue
				q := redis_state.NewQueue(shard,
					redis_state.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
						return true
					}),
					redis_state.WithPartitionConstraintConfigGetter(func(ctx context.Context, p redis_state.PartitionIdentifier) redis_state.PartitionConstraintConfig {
						return constraints
					}))

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
				qi, err := q.EnqueueItem(ctx, shard, queueItem, now, queue.EnqueueOpts{})
				require.NoError(t, err)

				backlog := q.ItemBacklog(ctx, qi)
				sp := q.ItemShadowPartition(ctx, qi)

				leaseID, err := q.BacklogRefill(ctx, &backlog, &sp, now.Add(time.Minute), constraints)
				require.NoError(t, err)
				require.NotNil(t, leaseID)
			})
		})
	}
}
