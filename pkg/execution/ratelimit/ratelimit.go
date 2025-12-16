package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
)

var (
	ErrRateLimitExceeded             = fmt.Errorf("rate limit exceeded")
	ErrEvaluatingRateLimitExpression = fmt.Errorf("rate limit expression evaluation failed")
	ErrNotRateLimited                = fmt.Errorf("not rate limited")
)

type rateLimitOptions struct {
	now time.Time

	idempotencyKey string
	idempotencyTTL time.Duration
}

type RateLimitOptionFn func(o *rateLimitOptions)

func WithNow(now time.Time) RateLimitOptionFn {
	return func(o *rateLimitOptions) {
		o.now = now
	}
}

func WithIdempotency(key string, ttl time.Duration) RateLimitOptionFn {
	return func(o *rateLimitOptions) {
		o.idempotencyKey = key
		o.idempotencyTTL = ttl
	}
}

type RateLimitResult struct {
	Limited        bool
	RetryAfter     time.Duration
	IdempotencyHit bool
}

type RateLimiter interface {
	RateLimit(ctx context.Context, key string, c inngest.RateLimit, options ...RateLimitOptionFn) (*RateLimitResult, error)
}

// RateLimitKey returns the rate limiting key given a function ID, rate limit config,
// and incoming event data.
func RateLimitKey(ctx context.Context, id uuid.UUID, c inngest.RateLimit, evt map[string]any) (string, error) {
	if c.Key == nil {
		return id.String(), nil
	}
	eval, err := expressions.NewExpressionEvaluator(ctx, *c.Key)
	if err != nil {
		return "", ErrEvaluatingRateLimitExpression
	}
	res, err := eval.Evaluate(ctx, expressions.NewData(map[string]any{"event": evt}))
	if err != nil {
		return "", ErrEvaluatingRateLimitExpression
	}
	if v, ok := res.(bool); ok && !v {
		return "", ErrNotRateLimited
	}

	// Take a checksum of this data.  It doesn't matter if this is a map or a string;
	// as long as we're consistent here.
	return hash(res, id), nil
}

func hash(res any, id uuid.UUID) string {
	sum := util.XXHash(res)
	return fmt.Sprintf("%s-%s", id, sum)
}
