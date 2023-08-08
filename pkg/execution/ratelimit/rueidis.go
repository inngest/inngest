package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/inngest"
	"github.com/rueian/rueidis"
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

func New(ctx context.Context, r rueidis.Client, prefix string) RateLimiter {
	return &rueidisStore{
		r:         r,
		casScript: rueidis.NewLuaScript(redisCASScript),
		prefix:    prefix,
	}
}

type rueidisStore struct {
	r         rueidis.Client
	casScript *rueidis.Lua

	prefix string
}

func (r *rueidisStore) RateLimit(ctx context.Context, key string, c inngest.RateLimit) (bool, time.Duration, error) {
	return rateLimit(ctx, r, key, c)
}

// GetWithTime returns the value of the key if it is in the store
// or -1 if it does not exist. It also returns the current time at
// the redis server to microsecond precision.
func (r *rueidisStore) GetWithTime(ctx context.Context, key string) (int64, time.Time, error) {
	key = r.prefix + key

	timeCmd := r.r.B().Time().Build()
	getKeyCmd := r.r.B().Get().Key(key).Build()

	res, timeErr := r.r.Do(ctx, timeCmd).AsStrSlice()
	v, valErr := r.r.Do(ctx, getKeyCmd).AsInt64()

	if timeErr != nil {
		return 0, time.Time{}, timeErr
	}
	now, err := redisTime(res)
	if err != nil {
		return 0, time.Time{}, err
	}

	if valErr != nil && rueidis.IsRedisNil(valErr) {
		return -1, now, nil
	}
	if valErr != nil {
		return 0, now, valErr
	}

	return v, now, nil
}

func redisTime(strs []string) (time.Time, error) {
	if len(strs) != 2 {
		return time.Time{}, fmt.Errorf("unknown slice transforming redis time")
	}

	secs, err := strconv.Atoi(strs[0])
	if err != nil {
		return time.Time{}, err
	}
	msecs, err := strconv.Atoi(strs[1])
	if err != nil {
		return time.Time{}, err
	}

	now := time.Unix(int64(secs), int64(msecs)*int64(time.Microsecond))
	return now, nil
}

// SetIfNotExistsWithTTL sets the value of key only if it is not
// already set in the store it returns whether a new value was set.
// If a new value was set, the ttl in the key is also set, though this
// operation is not performed atomically.
func (r *rueidisStore) SetIfNotExistsWithTTL(ctx context.Context, key string, value int64, ttl time.Duration) (bool, error) {
	key = r.prefix + key

	updated, err := r.r.Do(
		ctx,
		r.r.B().Setnx().Key(key).Value(fmt.Sprintf("%d", value)).Build(),
	).AsInt64()
	if err != nil {
		return false, err
	}

	// An `EXPIRE 0` will delete the key immediately, so make sure that we set
	// expiry for a minimum of one second out so that our results stay in the
	// store.
	if ttl < 1*time.Second {
		ttl = 1 * time.Second
	}

	err = r.r.Do(ctx, r.r.B().Expire().Key(key).Seconds(int64(ttl.Seconds())).Build()).Error()
	return updated == 1, err
}

// CompareAndSwapWithTTL atomically compares the value at key to the
// old value. If it matches, it sets it to the new value and returns
// true. Otherwise, it returns false. If the key does not exist in the
// store, it returns false with no error. If the swap succeeds, the
// ttl for the key is updated atomically.
func (r *rueidisStore) CompareAndSwapWithTTL(ctx context.Context, key string, old, new int64, ttl time.Duration) (bool, error) {
	key = r.prefix + key
	ttlSeconds := int(ttl.Seconds())

	// An `EXPIRE 0` will delete the key immediately, so make sure that we set
	// expiry for a minimum of one second out so that our results stay in the
	// store.
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	// result will be 0 or 1
	result, err := r.casScript.Exec(
		ctx,
		r.r,
		[]string{key},
		[]string{
			fmt.Sprintf("%d", old),
			fmt.Sprintf("%d", new),
			strconv.Itoa(ttlSeconds),
		},
	).AsInt64()

	swapped := result == 1
	if err != nil {
		if strings.Contains(err.Error(), redisCASMissingKey) {
			return false, nil
		}
		return false, err
	}
	return swapped, nil
}
