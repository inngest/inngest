package singleton

import (
	"context"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

const getAndDeleteLua = `
local v = redis.call('get', KEYS[1])
if v == false then
  return nil
end
redis.call('del', KEYS[1])
return v
`

func New(ctx context.Context, queueShards map[string]redis_state.RedisQueueShard, shardSelector redis_state.ShardSelector) Singleton {
	return &redisStore{
		queueShards:     queueShards,
		shardSelector:   shardSelector,
		getAndDelScript: rueidis.NewLuaScript(getAndDeleteLua),
	}
}

type redisStore struct {
	queueShards     map[string]redis_state.RedisQueueShard
	shardSelector   redis_state.ShardSelector
	getAndDelScript *rueidis.Lua
}

func (r *redisStore) HandleSingleton(ctx context.Context, key string, s inngest.Singleton, accountID uuid.UUID) (*ulid.ULID, error) {
	return singleton(ctx, r, key, s, accountID)
}

func (r *redisStore) ReleaseSingleton(ctx context.Context, key string, accountID uuid.UUID) (*ulid.ULID, error) {
	shard, err := r.shardByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	client := shard.RedisClient
	redisKey := r.generateSingletonKey(client, key)

	val, err := r.getAndDelScript.Exec(ctx, client.Client(), []string{redisKey}, nil).ToString()
	return parseRunIDFromRedisValue(val, err)
}

func (r *redisStore) GetCurrentRunID(ctx context.Context, key string, accountID uuid.UUID) (*ulid.ULID, error) {
	shard, err := r.shardByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	client := shard.RedisClient
	redisKey := r.generateSingletonKey(client, key)

	val, err := client.Client().Do(ctx, client.Client().B().Get().Key(redisKey).Build()).ToString()
	return parseRunIDFromRedisValue(val, err)
}

func (r *redisStore) shardByAccount(ctx context.Context, accountID uuid.UUID) (redis_state.RedisQueueShard, error) {
	return r.shardSelector(ctx, accountID, nil)
}

func (r *redisStore) generateSingletonKey(client *redis_state.QueueClient, key string) string {
	return client.KeyGenerator().SingletonKey(&queue.Singleton{Key: key})
}

func parseRunIDFromRedisValue(val string, err error) (*ulid.ULID, error) {
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, nil
		}
		return nil, err
	}

	runID, err := ulid.Parse(val)
	if err != nil {
		return nil, err
	}

	return &runID, nil
}
