package redis_state

import (
	"context"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestShadowPartitionActiveCheck(t *testing.T) {
	ctx := context.Background()

	cluster, rc := initRedis(t)
	defer rc.Close()

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

	client := defaultShard.RedisClient.Client()
	kg := defaultShard.RedisClient.KeyGenerator()

	t.Run("should not clean up active items from account active set", func(t *testing.T) {
		cluster.FlushAll()

		// goes to ready queue
		enqueueToBacklog = false
		qi, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = cluster.ZAdd(sp.accountActiveKey(kg), float64(clock.Now().UnixMilli()), qi.ID)
		require.NoError(t, err)

		err = q.shadowPartitionActiveCheck(ctx, &sp, client, kg)
		require.NoError(t, err)

		require.True(t, cluster.Exists(sp.accountActiveKey(kg)), cluster.Dump())
		require.NoError(t, err)
	})

	t.Run("should clean up non-active items from account active set", func(t *testing.T) {
		cluster.FlushAll()

		enqueueToBacklog = true
		qi, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = cluster.ZAdd(sp.accountActiveKey(kg), float64(clock.Now().UnixMilli()), qi.ID)
		require.NoError(t, err)

		err = q.shadowPartitionActiveCheck(ctx, &sp, client, kg)
		require.NoError(t, err)

		require.False(t, cluster.Exists(sp.accountActiveKey(kg)), cluster.Dump())
		require.NoError(t, err)
	})

	t.Run("should clean up missing items from account active set", func(t *testing.T) {
		cluster.FlushAll()

		_, err = cluster.ZAdd(sp.accountActiveKey(kg), float64(clock.Now().UnixMilli()), "missing-lol")
		require.NoError(t, err)

		err = q.shadowPartitionActiveCheck(ctx, &sp, client, kg)
		require.NoError(t, err)

		require.False(t, cluster.Exists(sp.accountActiveKey(kg)))
		require.NoError(t, err)
	})

	t.Run("should not clean up active items from partition active set", func(t *testing.T) {
		cluster.FlushAll()

		enqueueToBacklog = false
		qi, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = cluster.ZAdd(sp.activeKey(kg), float64(clock.Now().UnixMilli()), qi.ID)
		require.NoError(t, err)

		err = q.shadowPartitionActiveCheck(ctx, &sp, client, kg)
		require.NoError(t, err)

		require.True(t, cluster.Exists(sp.activeKey(kg)))
		require.NoError(t, err)
	})

	t.Run("should clean up non-active items from partition active set", func(t *testing.T) {
		cluster.FlushAll()

		enqueueToBacklog = true
		qi, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = cluster.ZAdd(sp.activeKey(kg), float64(clock.Now().UnixMilli()), qi.ID)
		require.NoError(t, err)

		err = q.shadowPartitionActiveCheck(ctx, &sp, client, kg)
		require.NoError(t, err)

		require.False(t, cluster.Exists(sp.activeKey(kg)))
		require.NoError(t, err)
	})

	t.Run("should clean up missing items from partition active set", func(t *testing.T) {
		cluster.FlushAll()

		_, err = cluster.ZAdd(sp.activeKey(kg), float64(clock.Now().UnixMilli()), "missing-lol")
		require.NoError(t, err)

		err = q.shadowPartitionActiveCheck(ctx, &sp, client, kg)
		require.NoError(t, err)

		require.False(t, cluster.Exists(sp.activeKey(kg)))
		require.NoError(t, err)
	})

	t.Run("should not clean up active items from custom concurrency key active set", func(t *testing.T) {
		cluster.FlushAll()

		enqueueToBacklog = false
		qi, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = cluster.ZAdd(backlog.customKeyActive(kg, 1), float64(clock.Now().UnixMilli()), qi.ID)
		require.NoError(t, err)

		err = q.shadowPartitionActiveCheck(ctx, &sp, client, kg)
		require.NoError(t, err)

		require.True(t, cluster.Exists(backlog.customKeyActive(kg, 1)))
		require.NoError(t, err)
	})

	t.Run("should clean up non-active items from custom concurrency key active set", func(t *testing.T) {
		cluster.FlushAll()

		enqueueToBacklog = true
		qi, err := q.EnqueueItem(ctx, defaultShard, item, clock.Now(), osqueue.EnqueueOpts{})
		require.NoError(t, err)

		_, err = cluster.ZAdd(backlog.customKeyActive(kg, 1), float64(clock.Now().UnixMilli()), qi.ID)
		require.NoError(t, err)

		err = q.shadowPartitionActiveCheck(ctx, &sp, client, kg)
		require.NoError(t, err)

		require.False(t, cluster.Exists(backlog.customKeyActive(kg, 1)))
		require.NoError(t, err)
	})

	t.Run("should clean up missing items from custom concurrency key active set", func(t *testing.T) {

	})
}
