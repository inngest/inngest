package redis_state

import (
	"context"
	"fmt"
	"strings"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

// SingletonMigrationKey discovers only a lock still owned by the item's run.
func (q *queue) SingletonMigrationKey(ctx context.Context, item osqueue.QueueItem) (string, error) {
	client := q.RedisClient.Client()
	kg := q.RedisClient.KeyGenerator()
	runID := item.Data.Identifier.RunID.String()
	key, err := client.Do(ctx, client.B().Get().Key(kg.SingletonRunKey(runID)).Build()).ToString()
	if rueidis.IsRedisNil(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	prefix := strings.TrimSuffix(kg.SingletonKey(nil), "-")
	logicalKey, ok := strings.CutPrefix(key, prefix)
	if !ok || logicalKey == "" {
		return "", fmt.Errorf("invalid singleton run mapping")
	}
	holder, err := client.Do(ctx, client.B().Get().Key(key).Build()).ToString()
	if rueidis.IsRedisNil(err) || (err == nil && holder != runID) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return logicalKey, nil
}

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
