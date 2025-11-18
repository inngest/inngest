package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/redis/rueidis"
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

	redisCASFixupScript = `
local v = redis.call('get', KEYS[1])
if v == false then
  return redis.error_reply("key does not exist")
end
if v ~= ARGV[1] then
  return 0
end
redis.call('set', KEYS[1], ARGV[2], "KEEPTTL")
return 1
`
)

func New(ctx context.Context, r rueidis.Client, prefix string) RateLimiter {
	return &rueidisStore{
		r:              r,
		casScript:      rueidis.NewLuaScript(redisCASScript),
		casFixupScript: rueidis.NewLuaScript(redisCASFixupScript),
		prefix:         prefix,
		luaRateLimiter: newLuaGCRARateLimiter(ctx, r, prefix),
	}
}

type rueidisStore struct {
	r              rueidis.Client
	casScript      *rueidis.Lua
	casFixupScript *rueidis.Lua

	prefix string

	luaRateLimiter RateLimiter

	disableGracefulScientificNotationParsing bool
}

func (r *rueidisStore) RateLimit(ctx context.Context, key string, c inngest.RateLimit, options ...RateLimitOptionFn) (*RateLimitResult, error) {
	o := &rateLimitOptions{}
	for _, opt := range options {
		opt(o)
	}

	// Dynamically switch to Lua implementation
	if o.useLuaImplementation {
		return r.luaRateLimiter.RateLimit(ctx, key, c, options...)
	}

	limited, retryAfter, err := rateLimit(ctx, r, key, c)
	if err != nil {
		return nil, err
	}

	return &RateLimitResult{
		Limited:    limited,
		RetryAfter: retryAfter,
	}, nil
}

// GetWithTime returns the value of the key if it is in the store
// or -1 if it does not exist. It also returns the current time at
// the redis server to microsecond precision.
func (r *rueidisStore) GetWithTime(ctx context.Context, key string) (int64, time.Time, error) {
	l := logger.StdlibLogger(ctx)
	key = r.prefix + key

	// get and parse current redis server time
	timeCmd := r.r.B().Time().Build()
	res, timeErr := r.r.Do(ctx, timeCmd).AsStrSlice()
	if timeErr != nil {
		return 0, time.Time{}, fmt.Errorf("failed to get redis server time: %w", timeErr)
	}
	now, err := redisTime(res)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to parse redis server time: %w", err)
	}

	// get and parse rate limit key value
	getKeyCmd := r.r.B().Get().Key(key).Build()
	v, err := r.r.Do(ctx, getKeyCmd).AsInt64()
	if err != nil && rueidis.IsRedisNil(err) {
		return -1, now, nil
	}

	// Happy path: No error
	if err == nil {
		return v, now, nil
	}

	// We could not load the value for some reason

	// Only continue for integer parsing errors, i/o timeouts should be retried properly
	numErr := &strconv.NumError{}
	if !errors.As(err, &numErr) || numErr.Err != strconv.ErrSyntax {
		return 0, now, fmt.Errorf("failed to get key value: %w", err)
	}

	if r.disableGracefulScientificNotationParsing {
		return 0, now, fmt.Errorf("unexpected scientific notation: %w", err)
	}

	// Try to get the raw string value and parse it as a float first
	rawResult := r.r.Do(ctx, r.r.B().Get().Key(key).Build())
	strVal, err := rawResult.ToString()
	if err != nil {
		return 0, now, fmt.Errorf("failed to get key value as string: %w", err)
	}

	// If the string value does not contain scientific notation, return
	if !strings.Contains(strVal, "e+") && !strings.Contains(strVal, "E+") {
		l.Error("rate limit key contained value that could not be parsed and was not scientific notation",
			"key", key,
			"value", strVal,
		)
		return 0, now, fmt.Errorf("rate limit value %s cannot be parsed: %w", strVal, err)
	}

	// This is scientific notation - try to parse as float and convert to int64
	floatVal, err := strconv.ParseFloat(strVal, 64)
	if err != nil {
		l.Error("rate limit key contained scientific notation value that could not be parsed",
			"parse_err", err,
			"key", key,
			"value", strVal,
		)
		return 0, now, fmt.Errorf("rate limit value %s included invald scientific notation: %w", strVal, err)
	}

	// Convert to int64, but clamp to safe values
	safeVal := int64(floatVal)

	// Optimistically apply CAS to switch from invalid string to parsed integer representation
	swapped, err := r.swapStringWithInt(ctx, key, strVal, safeVal)
	if err != nil {
		return 0, now, fmt.Errorf("could not swap to normalized rate limit value: %w", err)
	}

	if swapped {
		l.Warn("rate limit key contained scientific notation value, handled gracefully",
			"key", key,
			"value", strVal,
			"converted", safeVal,
		)
	} else {
		l.Warn("rate limit value changed while converting to safe value",
			"key", key,
			"value", strVal,
			"converted", safeVal,
		)
	}

	return safeVal, now, nil
}

func redisTime(strs []string) (time.Time, error) {
	if len(strs) != 2 {
		return time.Time{}, fmt.Errorf("unknown slice transforming redis time")
	}

	secs, err := strconv.Atoi(strs[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse secs from redis time: %w", err)
	}
	msecs, err := strconv.Atoi(strs[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse msecs from redis time: %w", err)
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

func (r *rueidisStore) swapStringWithInt(ctx context.Context, key string, old string, new int64) (bool, error) {
	// result will be 0 or 1
	result, err := r.casFixupScript.Exec(
		ctx,
		r.r,
		[]string{key},
		[]string{
			old,
			fmt.Sprintf("%d", new),
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
