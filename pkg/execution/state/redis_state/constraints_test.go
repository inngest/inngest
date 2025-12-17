package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
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
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItemLeaseConstraintCheck(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	shard := QueueShard{
		Kind:        string(enums.QueueShardKindRedis),
		RedisClient: NewQueueClient(rc, "q:v1"),
		Name:        consts.DefaultQueueShardName,
	}
	kg := shard.RedisClient.kg

	ctx := context.Background()
	l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelTrace))
	ctx = logger.WithStdlib(ctx, l)

	cmLifecycles := newConstraintAPIDebugLifecycles()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClock(clock),
		constraintapi.WithEnableDebugLogs(true),
		constraintapi.WithLifecycles(cmLifecycles),
		constraintapi.WithNumScavengerShards(1),
		constraintapi.WithQueueShards(map[string]rueidis.Client{
			consts.DefaultQueueShardName: rc,
		}),
		constraintapi.WithQueueStateKeyPrefix("q:v1"),
		constraintapi.WithRateLimitClient(rc),
		constraintapi.WithRateLimitKeyPrefix("rl"),
	)
	require.NoError(t, err)

	reset := func() {
		r.FlushAll()
		r.SetTime(clock.Now())
		cmLifecycles.reset()
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

	constraints := PartitionConstraintConfig{
		FunctionVersion: 1,
		Concurrency: PartitionConcurrency{
			AccountConcurrency:  10,
			FunctionConcurrency: 5,
		},
	}

	start := clock.Now()

	t.Run("waive checks for system queues", func(t *testing.T) {
		reset()

		qn := "example-system-queue"
		item := osqueue.QueueItem{
			Data: osqueue.Item{
				Payload:    json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{},
				QueueName:  &qn,
			},
			QueueName: &qn,
		}

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			WithCapacityManager(cm),
			// make lease extensions more frequent
			WithCapacityLeaseExtendInterval(time.Second),
			WithLogger(l),
			WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := q.ItemShadowPartition(ctx, qi)
		backlog := q.ItemBacklog(ctx, qi)

		res, err := q.itemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now(), kg)
		require.NoError(t, err)

		// No lease acquired
		require.Nil(t, res.capacityLease)
		require.True(t, res.skipConstraintChecks)

		// Do not expect a call for the system queue
		require.Equal(t, 0, len(cmLifecycles.acquireCalls))
		require.Equal(t, 0, len(cmLifecycles.extendCalls))
		require.Equal(t, 0, len(cmLifecycles.releaseCalls))
	})

	t.Run("skip constraintapi but require checks when missing identifier", func(t *testing.T) {
		reset()

		item := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					WorkflowID: fnID,
				},
			},
		}

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			WithCapacityManager(cm),
			// make lease extensions more frequent
			WithCapacityLeaseExtendInterval(time.Second),
			WithLogger(l),
			WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := q.ItemShadowPartition(ctx, qi)
		backlog := q.ItemBacklog(ctx, qi)

		res, err := q.itemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now(), kg)
		require.NoError(t, err)

		// No lease acquired
		require.Nil(t, res.capacityLease)
		require.False(t, res.skipConstraintChecks)

		// Do not expect a ConstraintAPI call for missing identifiers
		require.Equal(t, 0, len(cmLifecycles.acquireCalls))
		require.Equal(t, 0, len(cmLifecycles.extendCalls))
		require.Equal(t, 0, len(cmLifecycles.releaseCalls))
	})

	t.Run("skip constraintapi but require checks when capacity manager not configured", func(t *testing.T) {
		reset()

		item := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					WorkflowID: fnID,
				},
			},
		}

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			// make lease extensions more frequent
			WithCapacityLeaseExtendInterval(time.Second),
			WithLogger(l),
			WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := q.ItemShadowPartition(ctx, qi)
		backlog := q.ItemBacklog(ctx, qi)

		res, err := q.itemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now(), kg)
		require.NoError(t, err)

		// No lease acquired
		require.Nil(t, res.capacityLease)
		require.False(t, res.skipConstraintChecks)

		// Do not expect a ConstraintAPI call for missing capacity manager
		require.Equal(t, 0, len(cmLifecycles.acquireCalls))
		require.Equal(t, 0, len(cmLifecycles.extendCalls))
		require.Equal(t, 0, len(cmLifecycles.releaseCalls))
	})

	t.Run("skip constraintapi but require checks when feature flag disabled", func(t *testing.T) {
		reset()

		item := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					WorkflowID: fnID,
				},
			},
		}

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool) {
				return false, false // disable flag
			}),
			WithCapacityManager(cm),
			// make lease extensions more frequent
			WithCapacityLeaseExtendInterval(time.Second),
			WithLogger(l),
			WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := q.ItemShadowPartition(ctx, qi)
		backlog := q.ItemBacklog(ctx, qi)

		res, err := q.itemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now(), kg)
		require.NoError(t, err)

		// No lease acquired
		require.Nil(t, res.capacityLease)
		require.False(t, res.skipConstraintChecks) // Require checks

		// Do not expect a ConstraintAPI call for disabled feature flag
		require.Equal(t, 0, len(cmLifecycles.acquireCalls))
		require.Equal(t, 0, len(cmLifecycles.extendCalls))
		require.Equal(t, 0, len(cmLifecycles.releaseCalls))
	})

	t.Run("should not acquire lease with valid existing item lease", func(t *testing.T) {
		reset()

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			WithCapacityManager(cm),
			// make lease extensions more frequent
			WithCapacityLeaseExtendInterval(time.Second),
			WithLogger(l),
			WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		// Simulate valid lease
		capacityLeaseID := ulid.MustNew(ulid.Timestamp(clock.Now().Add(10*time.Second)), rand.Reader)

		qi.CapacityLease = &osqueue.CapacityLease{
			LeaseID: capacityLeaseID,
		}

		sp := q.ItemShadowPartition(ctx, qi)
		backlog := q.ItemBacklog(ctx, qi)

		res, err := q.itemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now(), kg)
		require.NoError(t, err)

		require.NotNil(t, res.capacityLease)
		require.True(t, res.skipConstraintChecks)

		// This time, we do not expect a call to the Constraint API, simply use the valid lease
		require.Equal(t, 0, len(cmLifecycles.acquireCalls))
		require.Equal(t, 0, len(cmLifecycles.extendCalls))
		require.Equal(t, 0, len(cmLifecycles.releaseCalls))
	})

	t.Run("should acquire lease with expired existing item lease", func(t *testing.T) {
		reset()

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			WithCapacityManager(cm),
			// make lease extensions more frequent
			WithCapacityLeaseExtendInterval(time.Second),
			WithLogger(l),
			WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		// Simulate expired lease
		capacityLeaseID := ulid.MustNew(ulid.Timestamp(clock.Now().Add(-10*time.Second)), rand.Reader)

		qi.CapacityLease = &osqueue.CapacityLease{
			LeaseID: capacityLeaseID,
		}

		sp := q.ItemShadowPartition(ctx, qi)
		backlog := q.ItemBacklog(ctx, qi)

		res, err := q.itemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now(), kg)
		require.NoError(t, err)

		require.NotNil(t, res.capacityLease)
		require.True(t, res.skipConstraintChecks)

		// Expect call because lease expired
		require.Equal(t, 1, len(cmLifecycles.acquireCalls))
		require.Equal(t, 0, len(cmLifecycles.extendCalls))
		require.Equal(t, 0, len(cmLifecycles.releaseCalls))
	})

	t.Run("acquire lease from constraint api", func(t *testing.T) {
		reset()

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			WithCapacityManager(cm),
			// make lease extensions more frequent
			WithCapacityLeaseExtendInterval(time.Second),
			WithLogger(l),
			WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := q.ItemShadowPartition(ctx, qi)
		backlog := q.ItemBacklog(ctx, qi)

		res, err := q.itemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now(), kg)
		require.NoError(t, err)

		require.NotNil(t, res.capacityLease)
		require.True(t, res.skipConstraintChecks)

		require.Equal(t, 1, len(cmLifecycles.acquireCalls))
		require.Equal(t, 0, len(cmLifecycles.extendCalls))
		require.Equal(t, 0, len(cmLifecycles.releaseCalls))
	})

	t.Run("lacking constraint capacity", func(t *testing.T) {
		reset()

		for i := range 10 {
			_, err := r.ZAdd(
				kg.Concurrency("account", accountID.String()),
				float64(clock.Now().Add(5*time.Second).UnixMilli()),
				fmt.Sprintf("i%d", i),
			)
			require.NoError(t, err)
		}

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			WithCapacityManager(cm),
			// make lease extensions more frequent
			WithCapacityLeaseExtendInterval(time.Second),
			WithLogger(l),
			WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := q.ItemShadowPartition(ctx, qi)
		backlog := q.ItemBacklog(ctx, qi)

		res, err := q.itemLeaseConstraintCheck(ctx, &sp, &backlog, constraints, &qi, clock.Now(), kg)
		require.NoError(t, err)

		require.Equal(t, enums.QueueConstraintAccountConcurrency, res.limitingConstraint)
		require.Nil(t, res.capacityLease)
		require.False(t, res.skipConstraintChecks)

		require.Equal(t, 1, len(cmLifecycles.acquireCalls))
		require.Equal(t, 0, len(cmLifecycles.extendCalls))
		require.Equal(t, 0, len(cmLifecycles.releaseCalls))

		require.Len(t, cmLifecycles.acquireCalls[0].GrantedLeases, 0)
		require.Len(t, cmLifecycles.acquireCalls[0].LimitingConstraints, 1)
		require.Equal(t, constraintapi.ConstraintKindConcurrency, cmLifecycles.acquireCalls[0].LimitingConstraints[0].Kind)
	})
}

func TestBacklogRefillConstraintCheck(t *testing.T) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()

	clock := clockwork.NewFakeClock()

	shard := QueueShard{
		Kind:        string(enums.QueueShardKindRedis),
		RedisClient: NewQueueClient(rc, "q:v1"),
		Name:        consts.DefaultQueueShardName,
	}
	kg := shard.RedisClient.kg

	ctx := context.Background()
	l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelTrace))
	ctx = logger.WithStdlib(ctx, l)

	cmLifecycles := newConstraintAPIDebugLifecycles()
	cm, err := constraintapi.NewRedisCapacityManager(
		constraintapi.WithClock(clock),
		constraintapi.WithEnableDebugLogs(true),
		constraintapi.WithLifecycles(cmLifecycles),
		constraintapi.WithNumScavengerShards(1),
		constraintapi.WithQueueShards(map[string]rueidis.Client{
			consts.DefaultQueueShardName: rc,
		}),
		constraintapi.WithQueueStateKeyPrefix("q:v1"),
		constraintapi.WithRateLimitClient(rc),
		constraintapi.WithRateLimitKeyPrefix("rl"),
	)
	require.NoError(t, err)

	reset := func() {
		r.FlushAll()
		r.SetTime(clock.Now())
		cmLifecycles.reset()
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

	constraints := PartitionConstraintConfig{
		FunctionVersion: 1,
		Concurrency: PartitionConcurrency{
			AccountConcurrency:  10,
			FunctionConcurrency: 5,
		},
	}

	start := clock.Now()

	t.Run("skip constraintapi but require checks when missing identifier", func(t *testing.T) {
		reset()

		item := osqueue.QueueItem{
			FunctionID: fnID,
			Data: osqueue.Item{
				Payload: json.RawMessage("{\"test\":\"payload\"}"),
				Identifier: state.Identifier{
					WorkflowID: fnID,
				},
			},
		}

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			WithCapacityManager(cm),
			// make lease extensions more frequent
			WithCapacityLeaseExtendInterval(time.Second),
			WithLogger(l),
			WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := q.ItemShadowPartition(ctx, qi)
		backlog := q.ItemBacklog(ctx, qi)

		opIdempotencyKey := "refill1"
		res, err := q.backlogRefillConstraintCheck(ctx, &sp, &backlog, constraints, []*osqueue.QueueItem{&qi}, kg, opIdempotencyKey, clock.Now())
		require.NoError(t, err)

		// No lease acquired
		require.Nil(t, res.itemCapacityLeases)
		require.False(t, res.skipConstraintChecks)

		// Do not expect a ConstraintAPI call for missing identifiers
		require.Equal(t, 0, len(cmLifecycles.acquireCalls))
		require.Equal(t, 0, len(cmLifecycles.extendCalls))
		require.Equal(t, 0, len(cmLifecycles.releaseCalls))
	})

	t.Run("skip constraintapi but require checks without capacity manager", func(t *testing.T) {
		reset()

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			// make lease extensions more frequent
			WithCapacityLeaseExtendInterval(time.Second),
			WithLogger(l),
			WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := q.ItemShadowPartition(ctx, qi)
		backlog := q.ItemBacklog(ctx, qi)

		opIdempotencyKey := "refill1"
		res, err := q.backlogRefillConstraintCheck(ctx, &sp, &backlog, constraints, []*osqueue.QueueItem{&qi}, kg, opIdempotencyKey, clock.Now())
		require.NoError(t, err)

		// No lease acquired
		require.Nil(t, res.itemCapacityLeases)
		require.False(t, res.skipConstraintChecks)

		// Do not expect a ConstraintAPI call for missing capacity manager
		require.Equal(t, 0, len(cmLifecycles.acquireCalls))
		require.Equal(t, 0, len(cmLifecycles.extendCalls))
		require.Equal(t, 0, len(cmLifecycles.releaseCalls))
	})

	t.Run("skip constraintapi but require checks with disabled feature flag", func(t *testing.T) {
		reset()

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool) {
				return false, false
			}),
			WithCapacityManager(cm),
			// make lease extensions more frequent
			WithCapacityLeaseExtendInterval(time.Second),
			WithLogger(l),
			WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := q.ItemShadowPartition(ctx, qi)
		backlog := q.ItemBacklog(ctx, qi)

		opIdempotencyKey := "refill1"
		res, err := q.backlogRefillConstraintCheck(ctx, &sp, &backlog, constraints, []*osqueue.QueueItem{&qi}, kg, opIdempotencyKey, clock.Now())
		require.NoError(t, err)

		// No lease acquired
		require.Nil(t, res.itemCapacityLeases)
		require.False(t, res.skipConstraintChecks)

		// Do not expect a ConstraintAPI call for missing capacity manager
		require.Equal(t, 0, len(cmLifecycles.acquireCalls))
		require.Equal(t, 0, len(cmLifecycles.extendCalls))
		require.Equal(t, 0, len(cmLifecycles.releaseCalls))
	})

	t.Run("acquire leases from constraintapi", func(t *testing.T) {
		reset()

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			WithCapacityManager(cm),
			// make lease extensions more frequent
			WithCapacityLeaseExtendInterval(time.Second),
			WithLogger(l),
			WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := q.ItemShadowPartition(ctx, qi)
		backlog := q.ItemBacklog(ctx, qi)

		opIdempotencyKey := "refill1"
		res, err := q.backlogRefillConstraintCheck(ctx, &sp, &backlog, constraints, []*osqueue.QueueItem{&qi}, kg, opIdempotencyKey, clock.Now())
		require.NoError(t, err)

		// Acquired lease and request to skip checks
		require.Len(t, res.itemCapacityLeases, 1)
		require.Len(t, res.itemsToRefill, 1)
		require.Equal(t, qi.ID, res.itemsToRefill[0])
		require.True(t, res.skipConstraintChecks)

		// Expect exactly one acquire request
		require.Equal(t, 1, len(cmLifecycles.acquireCalls))
		require.Equal(t, 0, len(cmLifecycles.extendCalls))
		require.Equal(t, 0, len(cmLifecycles.releaseCalls))
	})

	t.Run("lacking capacity returns 0 leases from constraintapi", func(t *testing.T) {
		reset()

		for i := range 10 {
			_, err := r.ZAdd(
				kg.Concurrency("account", accountID.String()),
				float64(clock.Now().Add(5*time.Second).UnixMilli()),
				fmt.Sprintf("i%d", i),
			)
			require.NoError(t, err)
		}

		q := NewQueue(
			shard,
			WithClock(clock),
			WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool {
				return true
			}),
			WithUseConstraintAPI(func(ctx context.Context, accountID uuid.UUID) (enable bool, fallback bool) {
				return true, true
			}),
			WithCapacityManager(cm),
			// make lease extensions more frequent
			WithCapacityLeaseExtendInterval(time.Second),
			WithLogger(l),
			WithPartitionConstraintConfigGetter(func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig {
				return constraints
			}),
		)

		qi, err := q.EnqueueItem(ctx, q.primaryQueueShard, item, start, osqueue.EnqueueOpts{})
		require.NoError(t, err)

		sp := q.ItemShadowPartition(ctx, qi)
		backlog := q.ItemBacklog(ctx, qi)

		opIdempotencyKey := "refill1"
		res, err := q.backlogRefillConstraintCheck(ctx, &sp, &backlog, constraints, []*osqueue.QueueItem{&qi}, kg, opIdempotencyKey, clock.Now())
		require.NoError(t, err)

		// Acquired lease and request to skip checks
		require.Len(t, res.itemCapacityLeases, 0)
		require.False(t, res.skipConstraintChecks)
		require.Equal(t, enums.QueueConstraintAccountConcurrency, res.limitingConstraint)

		// Expect exactly one acquire request
		require.Equal(t, 1, len(cmLifecycles.acquireCalls))
		require.Equal(t, 0, len(cmLifecycles.extendCalls))
		require.Equal(t, 0, len(cmLifecycles.releaseCalls))
	})
}

func TestConstraintConfigFromConstraints(t *testing.T) {
	tests := []struct {
		name        string
		constraints PartitionConstraintConfig
		expected    constraintapi.ConstraintConfig
	}{
		{
			name:        "empty constraints",
			constraints: PartitionConstraintConfig{},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 0,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:     0,
					FunctionConcurrency:    0,
					AccountRunConcurrency:  0,
					FunctionRunConcurrency: 0,
				},
			},
		},
		{
			name: "basic concurrency limits",
			constraints: PartitionConstraintConfig{
				FunctionVersion: 1,
				Concurrency: PartitionConcurrency{
					AccountConcurrency:     100,
					FunctionConcurrency:    10,
					AccountRunConcurrency:  50,
					FunctionRunConcurrency: 5,
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:     100,
					FunctionConcurrency:    10,
					AccountRunConcurrency:  50,
					FunctionRunConcurrency: 5,
				},
			},
		},
		{
			name: "with custom concurrency keys",
			constraints: PartitionConstraintConfig{
				FunctionVersion: 2,
				Concurrency: PartitionConcurrency{
					AccountConcurrency:  100,
					FunctionConcurrency: 10,
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							Limit:               5,
							HashedKeyExpression: "key1-hash",
						},
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeFn,
							Limit:               3,
							HashedKeyExpression: "key2-hash",
						},
					},
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 2,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:  100,
					FunctionConcurrency: 10,
					CustomConcurrencyKeys: []constraintapi.CustomConcurrencyLimit{
						{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeAccount,
							Limit:             5,
							KeyExpressionHash: "key1-hash",
						},
						{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeFn,
							Limit:             3,
							KeyExpressionHash: "key2-hash",
						},
					},
				},
			},
		},
		{
			name: "with throttle",
			constraints: PartitionConstraintConfig{
				FunctionVersion: 1,
				Throttle: &PartitionThrottle{
					Limit:                     10,
					Burst:                     5,
					Period:                    60,
					ThrottleKeyExpressionHash: "throttle-hash",
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:     0,
					FunctionConcurrency:    0,
					AccountRunConcurrency:  0,
					FunctionRunConcurrency: 0,
				},
				Throttle: []constraintapi.ThrottleConfig{
					{
						Limit:             10,
						Burst:             5,
						Period:            60,
						KeyExpressionHash: "throttle-hash",
					},
				},
			},
		},
		{
			name: "complete configuration",
			constraints: PartitionConstraintConfig{
				FunctionVersion: 3,
				Concurrency: PartitionConcurrency{
					AccountConcurrency:     200,
					FunctionConcurrency:    20,
					AccountRunConcurrency:  100,
					FunctionRunConcurrency: 10,
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							Limit:               15,
							HashedKeyExpression: "custom-key-hash",
						},
					},
				},
				Throttle: &PartitionThrottle{
					Limit:                     20,
					Burst:                     10,
					Period:                    30,
					ThrottleKeyExpressionHash: "complete-throttle-hash",
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 3,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:     200,
					FunctionConcurrency:    20,
					AccountRunConcurrency:  100,
					FunctionRunConcurrency: 10,
					CustomConcurrencyKeys: []constraintapi.CustomConcurrencyLimit{
						{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeAccount,
							Limit:             15,
							KeyExpressionHash: "custom-key-hash",
						},
					},
				},
				Throttle: []constraintapi.ThrottleConfig{
					{
						Limit:             20,
						Burst:             10,
						Period:            30,
						KeyExpressionHash: "complete-throttle-hash",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constraintConfigFromConstraints(tt.constraints)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConstraintItemsFromBacklog(t *testing.T) {
	accountID, fnID := uuid.New(), uuid.New()
	tests := []struct {
		name     string
		backlog  *QueueBacklog
		sp       *QueueShadowPartition
		expected []constraintapi.ConstraintItem
	}{
		{
			name: "minimal backlog",
			backlog: &QueueBacklog{
				ShadowPartitionID: fnID.String(),
			},
			sp: &QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			expected: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeFn,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:p:%s", fnID),
					},
				},
			},
		},
		{
			name: "with throttle",
			backlog: &QueueBacklog{
				Throttle: &BacklogThrottle{
					ThrottleKeyExpressionHash: "throttle-expr-hash",
					ThrottleKey:               "throttle-key-value",
				},
			},
			sp: &QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			expected: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeFn,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:p:%s", fnID),
					},
				},
				{
					Kind: constraintapi.ConstraintKindThrottle,
					Throttle: &constraintapi.ThrottleConstraint{
						KeyExpressionHash: "throttle-expr-hash",
						EvaluatedKeyHash:  "throttle-key-value",
					},
				},
			},
		},
		{
			name: "with custom concurrency keys",
			backlog: &QueueBacklog{
				ConcurrencyKeys: []BacklogConcurrencyKey{
					{
						CanonicalKeyID:      fmt.Sprintf("a:%s:%s", accountID, "custom-key-1-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeAccount,
						EntityID:            accountID,
						HashedKeyExpression: "custom-key-1-hash",
						HashedValue:         "custom-key-1-value",
					},
					{
						CanonicalKeyID:      fmt.Sprintf("f:%s:%s", fnID, "custom-key-2-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeFn,
						EntityID:            fnID,
						HashedKeyExpression: "custom-key-2-hash",
						HashedValue:         "custom-key-2-value",
					},
				},
			},
			sp: &QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			expected: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeFn,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:p:%s", fnID),
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "custom-key-1-hash",
						EvaluatedKeyHash:  "custom-key-1-value",
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:custom:a:%s:%s", accountID, "custom-key-1-value"),
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeFn,
						KeyExpressionHash: "custom-key-2-hash",
						EvaluatedKeyHash:  "custom-key-2-value",
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:custom:f:%s:%s", fnID, "custom-key-2-value"),
					},
				},
			},
		},
		{
			name: "complete backlog with throttle and concurrency keys",
			backlog: &QueueBacklog{
				Throttle: &BacklogThrottle{
					ThrottleKeyExpressionHash: "complete-throttle-hash",
					ThrottleKey:               "complete-throttle-value",
				},
				ConcurrencyKeys: []BacklogConcurrencyKey{
					{
						CanonicalKeyID:      fmt.Sprintf("e:%s:%s", fnID, "complete-key-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeEnv,
						EntityID:            fnID,
						HashedKeyExpression: "complete-key-hash",
						HashedValue:         "complete-key-value",
					},
				},
			},
			sp: &QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			expected: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeAccount,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:account:%s", accountID),
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeFn,
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:p:%s", fnID),
					},
				},
				{
					Kind: constraintapi.ConstraintKindThrottle,
					Throttle: &constraintapi.ThrottleConstraint{
						KeyExpressionHash: "complete-throttle-hash",
						EvaluatedKeyHash:  "complete-throttle-value",
					},
				},
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeEnv,
						KeyExpressionHash: "complete-key-hash",
						EvaluatedKeyHash:  "complete-key-value",
						InProgressItemKey: fmt.Sprintf("{q:v1}:concurrency:custom:e:%s:%s", fnID, "complete-key-value"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constraintItemsFromBacklog(tt.sp, tt.backlog, queueKeyGenerator{queueDefaultKey: "q:v1"})
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertLimitingConstraint(t *testing.T) {
	tests := []struct {
		name                string
		constraints         PartitionConstraintConfig
		limitingConstraints []constraintapi.ConstraintItem
		expected            enums.QueueConstraint
	}{
		{
			name:                "no limiting constraints",
			constraints:         PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{},
			expected:            enums.QueueConstraintNotLimited,
		},
		{
			name:        "account concurrency constraint",
			constraints: PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "",
					},
				},
			},
			expected: enums.QueueConstraintAccountConcurrency,
		},
		{
			name:        "function concurrency constraint",
			constraints: PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeFn,
						KeyExpressionHash: "",
					},
				},
			},
			expected: enums.QueueConstraintFunctionConcurrency,
		},
		{
			name: "custom concurrency key 1",
			constraints: PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							HashedKeyExpression: "custom-key-1",
						},
					},
				},
			},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "custom-key-1",
					},
				},
			},
			expected: enums.QueueConstraintCustomConcurrencyKey1,
		},
		{
			name: "custom concurrency key 2",
			constraints: PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							HashedKeyExpression: "custom-key-1",
						},
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeFn,
							HashedKeyExpression: "custom-key-2",
						},
					},
				},
			},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeFn,
						KeyExpressionHash: "custom-key-2",
					},
				},
			},
			expected: enums.QueueConstraintCustomConcurrencyKey2,
		},
		{
			name:        "throttle constraint",
			constraints: PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindThrottle,
				},
			},
			expected: enums.QueueConstraintThrottle,
		},
		{
			name:        "multiple constraints - last one wins",
			constraints: PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "",
					},
				},
				{
					Kind: constraintapi.ConstraintKindThrottle,
				},
			},
			expected: enums.QueueConstraintThrottle,
		},
		{
			name:        "unknown constraint kind",
			constraints: PartitionConstraintConfig{},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: "unknown-kind",
				},
			},
			expected: enums.QueueConstraintNotLimited,
		},
		{
			name: "custom concurrency key without matching configuration",
			constraints: PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							HashedKeyExpression: "different-key",
						},
					},
				},
			},
			limitingConstraints: []constraintapi.ConstraintItem{
				{
					Kind: constraintapi.ConstraintKindConcurrency,
					Concurrency: &constraintapi.ConcurrencyConstraint{
						Mode:              enums.ConcurrencyModeStep,
						Scope:             enums.ConcurrencyScopeAccount,
						KeyExpressionHash: "non-matching-key",
					},
				},
			},
			expected: enums.QueueConstraintNotLimited,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLimitingConstraint(tt.constraints, tt.limitingConstraints)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConstraintItemsBacklogToLimitingConstraintRoundTrip(t *testing.T) {
	accountID, fnID := uuid.New(), uuid.New()
	tests := []struct {
		name                    string
		backlog                 *QueueBacklog
		sp                      *QueueShadowPartition
		constraints             PartitionConstraintConfig
		expectedQueueConstraint enums.QueueConstraint
		description             string
	}{
		{
			name:    "account concurrency constraint round trip",
			backlog: &QueueBacklog{},
			sp: &QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			constraints: PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					AccountConcurrency: 10,
				},
			},
			expectedQueueConstraint: enums.QueueConstraintAccountConcurrency,
			description:             "Account concurrency constraint items should map back to account concurrency queue constraint",
		},
		{
			name:    "function concurrency constraint round trip",
			backlog: &QueueBacklog{},
			sp: &QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			constraints: PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					FunctionConcurrency: 5,
				},
			},
			expectedQueueConstraint: enums.QueueConstraintFunctionConcurrency,
			description:             "Function concurrency constraint items should map back to function concurrency queue constraint",
		},
		{
			name: "throttle constraint round trip",
			backlog: &QueueBacklog{
				Throttle: &BacklogThrottle{
					ThrottleKeyExpressionHash: "throttle-hash",
					ThrottleKey:               "throttle-value",
				},
			},
			sp: &QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			constraints: PartitionConstraintConfig{
				Throttle: &PartitionThrottle{
					Limit:                     10,
					Burst:                     5,
					Period:                    60,
					ThrottleKeyExpressionHash: "throttle-hash",
				},
			},
			expectedQueueConstraint: enums.QueueConstraintThrottle,
			description:             "Throttle constraint items should map back to throttle queue constraint",
		},
		{
			name: "custom concurrency key 1 round trip",
			backlog: &QueueBacklog{
				ConcurrencyKeys: []BacklogConcurrencyKey{
					{
						CanonicalKeyID:      fmt.Sprintf("a:%s:%s", accountID, "custom-key-1-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeAccount,
						EntityID:            accountID,
						HashedKeyExpression: "custom-key-1-hash",
						HashedValue:         "custom-key-1-value",
					},
				},
			},
			sp: &QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			constraints: PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							Limit:               3,
							HashedKeyExpression: "custom-key-1-hash",
						},
					},
				},
			},
			expectedQueueConstraint: enums.QueueConstraintCustomConcurrencyKey1,
			description:             "First custom concurrency key constraint items should map back to custom key 1 queue constraint",
		},
		{
			name: "custom concurrency key 2 round trip",
			backlog: &QueueBacklog{
				ConcurrencyKeys: []BacklogConcurrencyKey{
					{
						CanonicalKeyID:      fmt.Sprintf("a:%s:%s", accountID, "key-1-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeAccount,
						EntityID:            accountID,
						HashedKeyExpression: "key-1-hash",
						HashedValue:         "key-1-value",
					},
					{
						CanonicalKeyID:      fmt.Sprintf("f:%s:%s", fnID, "custom-key-2-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeFn,
						EntityID:            fnID,
						HashedKeyExpression: "custom-key-2-hash",
						HashedValue:         "custom-key-2-value",
					},
				},
			},
			sp: &QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			constraints: PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							Limit:               5,
							HashedKeyExpression: "key-1-hash",
						},
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeFn,
							Limit:               2,
							HashedKeyExpression: "custom-key-2-hash",
						},
					},
				},
			},
			expectedQueueConstraint: enums.QueueConstraintCustomConcurrencyKey2,
			description:             "Second custom concurrency key constraint items should map back to custom key 2 queue constraint",
		},
		{
			name: "multiple constraints with throttle taking precedence",
			backlog: &QueueBacklog{
				Throttle: &BacklogThrottle{
					ThrottleKeyExpressionHash: "throttle-hash",
					ThrottleKey:               "throttle-value",
				},
				ConcurrencyKeys: []BacklogConcurrencyKey{
					{
						CanonicalKeyID:      fmt.Sprintf("a:%s:%s", accountID, "custom-key-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeAccount,
						EntityID:            accountID,
						HashedKeyExpression: "custom-key-hash",
						HashedValue:         "custom-key-value",
					},
				},
			},
			sp: &QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			constraints: PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					AccountConcurrency: 100,
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							Limit:               3,
							HashedKeyExpression: "custom-key-hash",
						},
					},
				},
				Throttle: &PartitionThrottle{
					Limit:                     15,
					Burst:                     3,
					Period:                    30,
					ThrottleKeyExpressionHash: "throttle-hash",
				},
			},
			expectedQueueConstraint: enums.QueueConstraintThrottle,
			description:             "When multiple constraints exist, throttle should take precedence (last one wins)",
		},
		{
			name: "non-matching custom concurrency key should not limit",
			backlog: &QueueBacklog{
				ConcurrencyKeys: []BacklogConcurrencyKey{
					{
						CanonicalKeyID:      fmt.Sprintf("a:%s:%s", accountID, "different-value"),
						ConcurrencyMode:     enums.ConcurrencyModeStep,
						Scope:               enums.ConcurrencyScopeAccount,
						EntityID:            accountID,
						HashedKeyExpression: "different-hash",
						HashedValue:         "different-value",
					},
				},
			},
			sp: &QueueShadowPartition{
				PartitionID: fnID.String(),
				AccountID:   &accountID,
				FunctionID:  &fnID,
			},
			constraints: PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					CustomConcurrencyKeys: []CustomConcurrencyLimit{
						{
							Mode:                enums.ConcurrencyModeStep,
							Scope:               enums.ConcurrencyScopeAccount,
							Limit:               3,
							HashedKeyExpression: "non-matching-hash",
						},
					},
				},
			},
			expectedQueueConstraint: enums.QueueConstraintNotLimited,
			description:             "Custom concurrency keys that don't match configuration should not create limiting constraints",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Generate constraint items from the backlog
			constraintItems := constraintItemsFromBacklog(tt.sp, tt.backlog, queueKeyGenerator{queueDefaultKey: "q:v1"})

			// Step 2: Filter the constraint items to find the ones that would be limiting
			// We simulate what the constraint API would return as limiting constraints
			var simulatedLimitingConstraints []constraintapi.ConstraintItem

			// Determine which constraint type we expect to be limiting based on the test case
			switch tt.expectedQueueConstraint {
			case enums.QueueConstraintAccountConcurrency:
				// Only account concurrency would be limiting
				for _, item := range constraintItems {
					if item.Kind == constraintapi.ConstraintKindConcurrency && item.Concurrency != nil &&
						item.Concurrency.Scope == enums.ConcurrencyScopeAccount && item.Concurrency.KeyExpressionHash == "" {
						simulatedLimitingConstraints = append(simulatedLimitingConstraints, item)
						break
					}
				}
			case enums.QueueConstraintFunctionConcurrency:
				// Only function concurrency would be limiting
				for _, item := range constraintItems {
					if item.Kind == constraintapi.ConstraintKindConcurrency && item.Concurrency != nil &&
						item.Concurrency.Scope == enums.ConcurrencyScopeFn && item.Concurrency.KeyExpressionHash == "" {
						simulatedLimitingConstraints = append(simulatedLimitingConstraints, item)
						break
					}
				}
			case enums.QueueConstraintThrottle:
				// Only throttle would be limiting
				for _, item := range constraintItems {
					if item.Kind == constraintapi.ConstraintKindThrottle && item.Throttle != nil {
						simulatedLimitingConstraints = append(simulatedLimitingConstraints, item)
						break
					}
				}
			case enums.QueueConstraintCustomConcurrencyKey1:
				// Only the first custom concurrency key would be limiting
				for _, item := range constraintItems {
					if item.Kind == constraintapi.ConstraintKindConcurrency && item.Concurrency != nil &&
						item.Concurrency.KeyExpressionHash != "" && item.Concurrency.EvaluatedKeyHash != "" {
						// Check if this matches the first custom concurrency key in the configuration
						if len(tt.constraints.Concurrency.CustomConcurrencyKeys) > 0 {
							expectedKey := tt.constraints.Concurrency.CustomConcurrencyKeys[0]
							if item.Concurrency.Mode == expectedKey.Mode &&
								item.Concurrency.Scope == expectedKey.Scope &&
								item.Concurrency.KeyExpressionHash == expectedKey.HashedKeyExpression {
								simulatedLimitingConstraints = append(simulatedLimitingConstraints, item)
								break
							}
						}
					}
				}
			case enums.QueueConstraintCustomConcurrencyKey2:
				// Only the second custom concurrency key would be limiting
				for _, item := range constraintItems {
					if item.Kind == constraintapi.ConstraintKindConcurrency && item.Concurrency != nil &&
						item.Concurrency.KeyExpressionHash != "" && item.Concurrency.EvaluatedKeyHash != "" {
						// Check if this matches the second custom concurrency key in the configuration
						if len(tt.constraints.Concurrency.CustomConcurrencyKeys) > 1 {
							expectedKey := tt.constraints.Concurrency.CustomConcurrencyKeys[1]
							if item.Concurrency.Mode == expectedKey.Mode &&
								item.Concurrency.Scope == expectedKey.Scope &&
								item.Concurrency.KeyExpressionHash == expectedKey.HashedKeyExpression {
								simulatedLimitingConstraints = append(simulatedLimitingConstraints, item)
								break
							}
						}
					}
				}
			case enums.QueueConstraintNotLimited:
				// No constraints would be limiting - leave the slice empty
			}

			// Step 3: Convert the limiting constraints back to a queue constraint
			queueConstraint := convertLimitingConstraint(tt.constraints, simulatedLimitingConstraints)

			// Step 4: Verify the round trip matches expectations
			assert.Equal(t, tt.expectedQueueConstraint, queueConstraint, tt.description)

			// Additional verification: ensure the constraint items contain the expected types
			if tt.expectedQueueConstraint != enums.QueueConstraintNotLimited {
				assert.NotEmpty(t, simulatedLimitingConstraints, "Should have found limiting constraints for non-NotLimited queue constraint")
			}

			// Verify that basic account and function concurrency constraints are always present
			hasAccountConcurrency := false
			hasFunctionConcurrency := false
			for _, item := range constraintItems {
				if item.Kind == constraintapi.ConstraintKindConcurrency && item.Concurrency != nil {
					if item.Concurrency.Scope == enums.ConcurrencyScopeAccount && item.Concurrency.KeyExpressionHash == "" {
						hasAccountConcurrency = true
					}
					if item.Concurrency.Scope == enums.ConcurrencyScopeFn && item.Concurrency.KeyExpressionHash == "" {
						hasFunctionConcurrency = true
					}
				}
			}
			assert.True(t, hasAccountConcurrency, "Should always include account concurrency constraint")
			assert.True(t, hasFunctionConcurrency, "Should always include function concurrency constraint")
		})
	}
}
