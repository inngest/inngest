package redis_state

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func initRedis(t *testing.T) (*miniredis.Miniredis, rueidis.Client) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	return r, rc
}

func shardFromClient(name string, rc rueidis.Client, opts ...osqueue.QueueOpt) RedisQueueShard {
	queueClient := NewQueueClient(rc, QueueDefaultKey)
	shard := NewQueueShard(name, queueClient, osqueue.ExecutorAssignmentConfig{}, opts...)

	return shard
}

func mapFromShards(shards ...osqueue.QueueShard) map[string]osqueue.QueueShard {
	shardMap := make(map[string]osqueue.QueueShard)
	for _, qs := range shards {
		shardMap[qs.Name()] = qs
	}

	return shardMap
}

func alwaysSelectShard(shard osqueue.QueueShard) osqueue.ShardSelector {
	return func(ctx context.Context, accountId uuid.UUID, queueName *string) (osqueue.QueueShard, error) {
		return shard, nil
	}
}

type queueImpl interface {
	osqueue.QueueManager
	osqueue.QueueProcessor
}

func newQueue(t testing.TB, rc rueidis.Client, opts ...osqueue.QueueOpt) (queueImpl, RedisQueueShard) {
	ctx := context.Background()

	shard := shardFromClient(consts.DefaultQueueShardName, rc, opts...)

	queue, err := osqueue.New(
		ctx,
		"test-queue",
		shard,
		mapFromShards(shard),
		alwaysSelectShard(shard),
		opts...)
	require.NoError(t, err)

	return queue, shard
}
