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
func (l *luaGCRARateLimiter) RateLimit(ctx context.Context, key string, c inngest.RateLimit, now time.Time) (bool, time.Duration, error) {
	dur, err := str2duration.ParseDuration(c.Period)
	if err != nil {
		return true, -1, err  // limited = true on error
	}

	burst := int(c.Limit / 10)
	nowNS := now.UnixNano()
	periodNS := dur.Nanoseconds()
	limit := c.Limit

	keys := []string{}
	args := []string{
		l.prefix + key,
		fmt.Sprintf("%d", nowNS),
		fmt.Sprintf("%d", periodNS),
		fmt.Sprintf("%d", limit),
		fmt.Sprintf("%d", burst),
	}

	res, err := scripts["ratelimit"].Exec(ctx, l.r, keys, args).AsIntSlice()
	if err != nil {
		return true, 0, fmt.Errorf("could not invoke rate limit: %w", err)  // limited = true on error
	}

	if len(res) != 2 {
		return true, 0, fmt.Errorf("invalid rate limit response: %w", err)  // limited = true on error
	}

	switch res[0] {
	// rate limited
	case 0:
		retryAfterNS := res[1] - nowNS
		if retryAfterNS < 0 {
			retryAfterNS = 0
		}
		return true, time.Duration(retryAfterNS), nil  // limited = true
		// ok
	case 1:
		return false, 0, nil  // limited = false
	default:
		return true, 0, fmt.Errorf("invalid return status %v", res[0])  // limited = true on error
	}
}
