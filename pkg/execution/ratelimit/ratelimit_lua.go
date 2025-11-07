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

// RateLimit implements RateLimiter.
func (l *luaGCRARateLimiter) RateLimit(ctx context.Context, key string, c inngest.RateLimit, now time.Time) (bool, time.Duration, error) {
	dur, err := str2duration.ParseDuration(c.Period)
	if err != nil {
		return true, -1, err
	}

	burst := int(c.Limit / 10)
	nowNS := now.UnixNano()
	periodNS := dur.Nanoseconds()
	limit := c.Limit

	keys := []string{}
	args := []string{
		key,
		fmt.Sprintf("%d", nowNS),
		fmt.Sprintf("%d", periodNS),
		fmt.Sprintf("%d", limit),
		fmt.Sprintf("%d", burst),
	}

	res, err := scripts["ratelimit"].Exec(ctx, l.r, keys, args).AsIntSlice()
	if err != nil {
		return false, 0, fmt.Errorf("could not invoke rate limit: %w", err)
	}

	if len(res) != 2 {
		return false, 0, fmt.Errorf("invalid rate limit response: %w", err)
	}

	switch res[0] {
	// rate limited
	case 0:
		return false, time.Until(time.Unix(0, res[1])), nil
		// ok
	case 1:
		return true, 0, nil
	default:
		return false, 0, fmt.Errorf("invalid return status %v", res[0])
	}
}
