package redis_state

import (
	"context"
	"strconv"
	"time"

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

// singletonJoinLua registers a waiter on an active singleton run.
//
// It is atomic with singletonJoinCompleteLua (Redis runs scripts serially): a
// waiter either lands in the set BEFORE completion (and is returned by the
// completion script), or sees the completion payload and is never registered.
// There is no interleaving in which a waiter is neither.
//
// KEYS[1] = waiter set, KEYS[2] = completion payload
// ARGV[1] = member, ARGV[2] = ttl in ms
const singletonJoinLua = `
local done = redis.call('GET', KEYS[2])
if done then
  return done
end

redis.call('SADD', KEYS[1], ARGV[1])

-- Only ever extend the set's lifetime, so a long-lived waiter is not cut
-- short by a shorter one.
local ttl = redis.call('PTTL', KEYS[1])
if ttl < tonumber(ARGV[2]) then
  redis.call('PEXPIRE', KEYS[1], ARGV[2])
end

return false
`

// singletonJoinCompleteLua stores the completion payload and returns the
// waiters. It deliberately does not delete the waiter set so that a retried
// finalize returns the same waiters (at-least-once delivery); both keys expire
// via TTL instead.
//
// KEYS[1] = waiter set, KEYS[2] = completion payload
// ARGV[1] = payload, ARGV[2] = ttl in ms
const singletonJoinCompleteLua = `
redis.call('SET', KEYS[2], ARGV[1], 'PX', ARGV[2])

if redis.call('EXISTS', KEYS[1]) == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[2])
end

return redis.call('SMEMBERS', KEYS[1])
`

var (
	singletonJoinScript         = rueidis.NewLuaScript(singletonJoinLua)
	singletonJoinCompleteScript = rueidis.NewLuaScript(singletonJoinCompleteLua)
)

// SingletonJoin implements queue.ShardOperations.
func (q *queue) SingletonJoin(ctx context.Context, scope osqueue.Scope, activeRunID ulid.ULID, member string, ttl time.Duration) ([]byte, error) {
	client := q.RedisClient.Client()
	kg := q.RedisClient.KeyGenerator()

	res := singletonJoinScript.Exec(
		ctx,
		client,
		[]string{kg.SingletonJoinKey(activeRunID.String()), kg.SingletonJoinDoneKey(activeRunID.String())},
		[]string{member, strconv.FormatInt(ttl.Milliseconds(), 10)},
	)

	val, err := res.ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			// Registered; the run has not completed.
			return nil, nil
		}
		return nil, err
	}

	return []byte(val), nil
}

// SingletonJoinComplete implements queue.ShardOperations.
func (q *queue) SingletonJoinComplete(ctx context.Context, scope osqueue.Scope, activeRunID ulid.ULID, payload []byte, ttl time.Duration) ([]string, error) {
	client := q.RedisClient.Client()
	kg := q.RedisClient.KeyGenerator()

	return singletonJoinCompleteScript.Exec(
		ctx,
		client,
		[]string{kg.SingletonJoinKey(activeRunID.String()), kg.SingletonJoinDoneKey(activeRunID.String())},
		[]string{string(payload), strconv.FormatInt(ttl.Milliseconds(), 10)},
	).AsStrSlice()
}
