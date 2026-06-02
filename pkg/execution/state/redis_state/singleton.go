package redis_state

import (
	"context"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
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

var getAndDeleteScript = rueidis.NewLuaScript(getAndDeleteLua)

// SingletonGetRunID implements queue.ShardOperations.
func (q *queue) SingletonGetRunID(ctx context.Context, scope osqueue.Scope, key string) (*ulid.ULID, error) {
	client := q.RedisClient.Client()
	redisKey := q.RedisClient.KeyGenerator().SingletonKey(&osqueue.Singleton{Key: key})

	val, err := client.Do(ctx, client.B().Get().Key(redisKey).Build()).ToString()
	return parseRunIDFromRedisValue(val, err)
}

// SingletonReleaseRunID implements queue.ShardOperations.
func (q *queue) SingletonReleaseRunID(ctx context.Context, scope osqueue.Scope, key string) (*ulid.ULID, error) {
	client := q.RedisClient.Client()
	redisKey := q.RedisClient.KeyGenerator().SingletonKey(&osqueue.Singleton{Key: key})

	val, err := getAndDeleteScript.Exec(ctx, client, []string{redisKey}, nil).ToString()
	return parseRunIDFromRedisValue(val, err)
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
