package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/inngest"
	"github.com/redis/rueidis"
	"github.com/xhit/go-str2duration/v2"
)

type luaGCRARateLimiter struct {
	r      rueidis.Client
	prefix string
}

func newLuaGCRARateLimiter(ctx context.Context, r rueidis.Client, prefix string) RateLimiter {
	return &luaGCRARateLimiter{
		r:      r,
		prefix: prefix,
	}
}

// RateLimit implements RateLimiter, returning (limited, retryAfter, error).
func (l *luaGCRARateLimiter) RateLimit(ctx context.Context, key string, c inngest.RateLimit, options ...RateLimitOptionFn) (*RateLimitResult, error) {
	o := &rateLimitOptions{
		now:                  time.Now(),
		useLuaImplementation: true,
		idempotencyKey:       "-",
	}
	for _, opt := range options {
		opt(o)
	}

	dur, err := str2duration.ParseDuration(c.Period)
	if err != nil {
		return nil, err // limited = true on error
	}

	burst := int(c.Limit / 10)
	nowNS := o.now.UnixNano()
	periodNS := dur.Nanoseconds()
	limit := c.Limit

	keys := []string{
		l.prefix + key,
		l.prefix + o.idempotencyKey,
	}
	args := []string{
		fmt.Sprintf("%d", nowNS),
		fmt.Sprintf("%d", periodNS),
		fmt.Sprintf("%d", limit),
		fmt.Sprintf("%d", burst),
		fmt.Sprintf("%d", int(o.idempotencyTTL.Seconds())),
	}

	res, err := scripts["ratelimit"].Exec(ctx, l.r, keys, args).AsIntSlice()
	if err != nil {
		return nil, fmt.Errorf("could not invoke rate limit: %w", err) // limited = true on error
	}

	if len(res) != 2 {
		return nil, fmt.Errorf("invalid rate limit response: %w", err) // limited = true on error
	}

	switch res[0] {
	// rate limited
	case 0:
		retryAfterNS := res[1] - nowNS
		if retryAfterNS < 0 {
			retryAfterNS = 0
		}
		return &RateLimitResult{
			Limited:    true,
			RetryAfter: time.Duration(retryAfterNS),
		}, nil
		// ok
	case 1:
		return &RateLimitResult{
			Limited: false,
		}, nil // limited = false
		// idempotency
	case 2:
		return &RateLimitResult{
			Limited:        false,
			IdempotencyHit: true,
		}, nil
	default:
		return nil, fmt.Errorf("invalid return status %v", res[0]) // limited = true on error
	}
}
