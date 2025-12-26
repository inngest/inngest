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
	ctx := context.Background()
	options := osqueue.NewQueueOptions(ctx, opts...)
	queueClient := NewQueueClient(rc, QueueDefaultKey)
	shard := NewRedisQueue(*options, consts.DefaultQueueShardName, queueClient).(RedisQueueShard)

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

func newQueue(t testing.TB, rc rueidis.Client, opts ...osqueue.QueueOpt) (osqueue.Queue, RedisQueueShard) {
	ctx := context.Background()

	shard := shardFromClient(consts.DefaultQueueShardName, rc, opts...)

	queue, err := osqueue.NewQueueProcessor(
		ctx,
		"test-queue",
		shard,
		mapFromShards(shard),
		alwaysSelectShard(shard),
		opts...)
	require.NoError(t, err)

	return queue, shard
}
