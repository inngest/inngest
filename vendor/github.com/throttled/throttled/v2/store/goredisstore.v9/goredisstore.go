// Package goredisstore offers Redis-based store implementation for throttled using v9 of go-redis.
package goredisstore // import "github.com/throttled/throttled/v2/store/goredisstore"

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	redisCASMissingKey = "key does not exist"
	redisCASScript     = `
local v = redis.call('get', KEYS[1])
if v == false then
  return redis.error_reply("key does not exist")
end
if v ~= ARGV[1] then
  return 0
end
redis.call('setex', KEYS[1], ARGV[3], ARGV[2])
return 1
`
)

// GoRedisStore implements a Redis-based store using go-redis v9.
type GoRedisStore struct {
	client redis.UniversalClient
	prefix string
}

// NewCtx creates a new Redis-based store, using the provided pool to get
// its connections. The keys will have the specified keyPrefix, which
// may be an empty string, and the database index specified by db will
// be selected to store the keys. Any updating operations will reset
// the key TTL to the provided value rounded down to the nearest
// second. Depends on Redis 2.6+ for EVAL support.
func NewCtx(client redis.UniversalClient, keyPrefix string) (*GoRedisStore, error) {
	return &GoRedisStore{
		client: client,
		prefix: keyPrefix,
	}, nil
}

// GetWithTime returns the value of the key if it is in the store
// or -1 if it does not exist. It also returns the current time at
// the redis server to microsecond precision.
func (r *GoRedisStore) GetWithTime(ctx context.Context, key string) (int64, time.Time, error) {
	key = r.prefix + key

	pipe := r.client.Pipeline()
	timeCmd := pipe.Time(ctx)
	getKeyCmd := pipe.Get(ctx, key)
	_, err := pipe.Exec(ctx)

	now, err := timeCmd.Result()
	if err != nil {
		return 0, now, err
	}

	v, err := getKeyCmd.Int64()
	if err == redis.Nil {
		return -1, now, nil
	} else if err != nil {
		return 0, now, err
	}

	return v, now, nil
}

// SetIfNotExistsWithTTL sets the value of key only if it is not
// already set in the store it returns whether a new value was set.
// If a new value was set, the ttl in the key is also set, though this
// operation is not performed atomically.
func (r *GoRedisStore) SetIfNotExistsWithTTL(ctx context.Context, key string, value int64, ttl time.Duration) (bool, error) {
	key = r.prefix + key

	updated, err := r.client.SetNX(ctx, key, value, 0).Result()
	if err != nil {
		return false, err
	}

	// An `EXPIRE 0` will delete the key immediately, so make sure that we set
	// expiry for a minimum of one second out so that our results stay in the
	// store.
	if ttl < 1*time.Second {
		ttl = 1 * time.Second
	}

	err = r.client.Expire(ctx, key, ttl).Err()
	return updated, err
}

// CompareAndSwapWithTTL atomically compares the value at key to the
// old value. If it matches, it sets it to the new value and returns
// true. Otherwise, it returns false. If the key does not exist in the
// store, it returns false with no error. If the swap succeeds, the
// ttl for the key is updated atomically.
func (r *GoRedisStore) CompareAndSwapWithTTL(ctx context.Context, key string, old, new int64, ttl time.Duration) (bool, error) {
	key = r.prefix + key

	ttlSeconds := int(ttl.Seconds())

	// An `EXPIRE 0` will delete the key immediately, so make sure that we set
	// expiry for a minimum of one second out so that our results stay in the
	// store.
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	// result will be 0 or 1
	result, err := r.client.Eval(ctx, redisCASScript, []string{key}, old, new, ttlSeconds).Result()

	var swapped bool
	if s, ok := result.(int64); ok {
		swapped = s == 1
	} // if not ok, zero value of swapped is false.

	if err != nil {
		if strings.Contains(err.Error(), redisCASMissingKey) {
			return false, nil
		}
		return false, err
	}

	return swapped, nil
}
