package redis_state

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
	"time"
)

func TestShadowPartitionActiveCheck(t *testing.T) {
	ctx := context.Background()

	cluster, rc := initRedis(t)
	defer rc.Close()

	l := logger.StdlibLogger(ctx, logger.WithLoggerLevel(slog.LevelDebug))

	ctx = logger.WithStdlib(ctx, l)

	defaultShard := QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: NewQueueClient(rc, QueueDefaultKey), Name: consts.DefaultQueueShardName}
	clock := clockwork.NewFakeClock()

	enqueueToBacklog := false
	q := NewQueue(
		defaultShard,
		WithClock(clock),
		WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID) bool {
			return enqueueToBacklog
		}),
		WithDisableLeaseChecks(func(ctx context.Context, acctID uuid.UUID) bool {
			return true
		}),
		WithReadOnlySpotChecks(func(ctx context.Context, acctID uuid.UUID) bool {
			return false
		}),
		WithActiveSpotCheckProbability(func(ctx context.Context, acctID uuid.UUID) (int, int) {
			return 100, 100
		}),
		WithRunMode(QueueRunMode{
			ActiveChecker: true,
		}),
	)

	fnID, accountID, envID := uuid.New(), uuid.New(), uuid.New()

	ck1 := createConcurrencyKey(enums.ConcurrencyScopeFn, fnID, "bruno", 5)

	item := osqueue.QueueItem{
		ID:          "test",
		FunctionID:  fnID,
		WorkspaceID: envID,
		Data: osqueue.Item{
			WorkspaceID: envID,
			Kind:        osqueue.KindEdge,
			Identifier: state.Identifier{
				WorkflowID:  fnID,
				AccountID:   accountID,
				WorkspaceID: envID,
			},
			CustomConcurrencyKeys: []state.CustomConcurrency{
				ck1,
			},
		},
		QueueName: nil,
	}

	qi, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
	require.NoError(t, err)

	sp := q.ItemShadowPartition(ctx, qi)
	backlog := q.ItemBacklog(ctx, qi)

	kg := defaultShard.RedisClient.KeyGenerator()

	setup := func(t *testing.T) {
		cluster.FlushAll()

		marshaled, err := json.Marshal(backlog)
		require.NoError(t, err)
		cluster.HSet(kg.BacklogMeta(), backlog.BacklogID, string(marshaled))

		marshaled, err = json.Marshal(sp)
		require.NoError(t, err)
		cluster.HSet(kg.ShadowPartitionMeta(), sp.PartitionID, string(marshaled))

		q.activeCheckAccountProbability = 0
	}

	t.Run("should work on missing account set", func(t *testing.T) {
		setup(t)

		q.activeCheckAccountProbability = 100
		_, err := q.ActiveCheck(ctx)
		require.NoError(t, err)
	})

	t.Run("account entrypoint should work", func(t *testing.T) {
		setup(t)

		testAccountID := uuid.New()
		_, err := cluster.ZAdd(kg.AccountActiveCheckSet(), float64(q.clock.Now().UnixMilli()), testAccountID.String())
		require.NoError(t, err)

		keyActive := kg.ActiveSet("account", testAccountID.String())
		_, err = cluster.SAdd(keyActive, "invalid")
		require.NoError(t, err)

		require.True(t, cluster.Exists(keyActive))

		q.activeCheckAccountProbability = 100
		_, err = q.ActiveCheck(ctx)
		require.NoError(t, err)

		require.False(t, cluster.Exists(keyActive))
	})

	t.Run("should work on missing set", func(t *testing.T) {
		setup(t)

		res, err := q.activeCheckScan(ctx, defaultShard, sp.accountActiveKey(kg), sp.accountInProgressKey(kg), 0, 10)
		require.NoError(t, err)
		require.NotNil(t, res)
	})

	t.Run("adding to active check should work", func(t *testing.T) {
		setup(t)

		err := q.AddBacklogToActiveCheck(ctx, defaultShard, accountID, backlog.BacklogID)
		require.NoError(t, err)

		require.True(t, cluster.Exists(kg.BacklogActiveCheckSet()))
		require.True(t, hasMember(t, cluster, kg.BacklogActiveCheckSet(), backlog.BacklogID))
	})

	t.Run("should not clean up active items from account active set", func(t *testing.T) {
		setup(t)

		// goes to ready queue
		enqueueToBacklog = false
		qi, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = cluster.SAdd(sp.accountActiveKey(kg), qi.ID)
		require.NoError(t, err)

		_, err = q.backlogActiveCheck(ctx, &backlog, defaultShard, kg)
		require.NoError(t, err)

		require.True(t, cluster.Exists(sp.accountActiveKey(kg)), cluster.Dump())
		require.NoError(t, err)
	})

	t.Run("should not clean up leased items", func(t *testing.T) {
		setup(t)

		// goes to ready queue
		enqueueToBacklog = false
		qi, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		leaseID, err := q.Lease(ctx, qi, 20*time.Second, q.clock.Now(), nil)
		require.NoError(t, err)
		require.NotNil(t, leaseID)

		res, err := q.activeCheckScan(ctx, defaultShard, sp.accountActiveKey(kg), sp.accountInProgressKey(kg), 0, 10)
		require.NoError(t, err)
		require.Len(t, res.LeasedItems, 1)
		require.Equal(t, qi.ID, res.LeasedItems[0])

		res, err = q.activeCheckScan(ctx, defaultShard, sp.activeKey(kg), sp.inProgressKey(kg), 0, 10)
		require.NoError(t, err)
		require.Len(t, res.LeasedItems, 1)
		require.Equal(t, qi.ID, res.LeasedItems[0])

		_, err = q.backlogActiveCheck(ctx, &backlog, defaultShard, kg)
		require.NoError(t, err)

		require.True(t, cluster.Exists(sp.accountActiveKey(kg)), cluster.Dump())
		require.NoError(t, err)
	})

	t.Run("should clean up non-active items from account active set", func(t *testing.T) {
		setup(t)

		enqueueToBacklog = true
		qi, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = cluster.SAdd(sp.accountActiveKey(kg), qi.ID)
		require.NoError(t, err)

		res, err := q.activeCheckScan(ctx, defaultShard, sp.accountActiveKey(kg), sp.accountInProgressKey(kg), 0, 10)
		require.NoError(t, err)
		require.Len(t, res.StaleItems, 1)
		require.Equal(t, qi.ID, res.StaleItems[0].ID)

		_, err = q.backlogActiveCheck(ctx, &backlog, defaultShard, kg)
		require.NoError(t, err)

		require.False(t, cluster.Exists(sp.accountActiveKey(kg)), cluster.Dump())
		require.NoError(t, err)
	})

	t.Run("should clean up missing items from account active set", func(t *testing.T) {
		setup(t)

		enqueueToBacklog = true
		_, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = cluster.SAdd(sp.accountActiveKey(kg), "missing-lol")
		require.NoError(t, err)

		_, err = q.backlogActiveCheck(ctx, &backlog, defaultShard, kg)
		require.NoError(t, err)

		require.False(t, cluster.Exists(sp.accountActiveKey(kg)))
		require.NoError(t, err)
	})

	t.Run("should not clean up active items from partition active set", func(t *testing.T) {
		setup(t)

		enqueueToBacklog = false
		qi, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = cluster.SAdd(sp.activeKey(kg), qi.ID)
		require.NoError(t, err)

		_, err = q.backlogActiveCheck(ctx, &backlog, defaultShard, kg)
		require.NoError(t, err)

		require.True(t, cluster.Exists(sp.activeKey(kg)))
		require.NoError(t, err)
	})

	t.Run("should clean up non-active items from partition active set", func(t *testing.T) {
		setup(t)

		enqueueToBacklog = true
		qi, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = cluster.SAdd(sp.activeKey(kg), qi.ID)
		require.NoError(t, err)

		_, err = q.backlogActiveCheck(ctx, &backlog, defaultShard, kg)
		require.NoError(t, err)

		require.False(t, cluster.Exists(sp.activeKey(kg)))
		require.NoError(t, err)
	})

	t.Run("should clean up missing items from partition active set", func(t *testing.T) {
		setup(t)

		enqueueToBacklog = true
		_, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = cluster.SAdd(sp.activeKey(kg), "missing-lol")
		require.NoError(t, err)

		_, err = q.backlogActiveCheck(ctx, &backlog, defaultShard, kg)
		require.NoError(t, err)

		require.False(t, cluster.Exists(sp.activeKey(kg)))
		require.NoError(t, err)
	})

	t.Run("should not clean up active items from custom concurrency key active set", func(t *testing.T) {
		setup(t)

		enqueueToBacklog = false
		qi, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = cluster.SAdd(backlog.customKeyActive(kg, 1), qi.ID)
		require.NoError(t, err)

		_, err = q.backlogActiveCheck(ctx, &backlog, defaultShard, kg)
		require.NoError(t, err)

		require.True(t, cluster.Exists(backlog.customKeyActive(kg, 1)))
		require.NoError(t, err)
	})

	t.Run("should clean up non-active items from custom concurrency key active set", func(t *testing.T) {
		setup(t)

		enqueueToBacklog = true
		qi, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = cluster.SAdd(backlog.customKeyActive(kg, 1), qi.ID)
		require.NoError(t, err)

		_, err = q.backlogActiveCheck(ctx, &backlog, defaultShard, kg)
		require.NoError(t, err)

		require.False(t, cluster.Exists(backlog.customKeyActive(kg, 1)))
		require.NoError(t, err)
	})

	t.Run("should clean up missing items from custom concurrency key active set", func(t *testing.T) {
		setup(t)

		enqueueToBacklog = true
		qi, err = q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = cluster.SAdd(backlog.customKeyActive(kg, 1), "missing-lol")
		require.NoError(t, err)

		_, err = q.backlogActiveCheck(ctx, &backlog, defaultShard, kg)
		require.NoError(t, err)

		require.False(t, cluster.Exists(backlog.customKeyActive(kg, 1)))
		require.NoError(t, err)
	})
}
