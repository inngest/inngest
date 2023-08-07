package ratelimit

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/throttled/throttled/v2"
	"github.com/xhit/go-str2duration/v2"
)

var (
	ErrRateLimitExceeded             = fmt.Errorf("rate limit exceeded")
	ErrEvaluatingRateLimitExpression = fmt.Errorf("rate limit expression evaluation failed")
)

type RateLimiter interface {
	RateLimit(ctx context.Context, key string, c inngest.RateLimit) (bool, time.Duration, error)
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
	res, _, err := eval.Evaluate(ctx, expressions.NewData(map[string]any{"event": evt}))
	if err != nil {
		return "", ErrEvaluatingRateLimitExpression
	}

	// Take a checksum of this data.  It doesn't matter if this is a map or a string;
	// as long as we're consistent here.
	return hash(res, id), nil
}

func hash(res any, id uuid.UUID) string {
	ui := xxhash.Sum64String(fmt.Sprintf("%v", res))
	sum := strconv.FormatUint(ui, 36)
	return fmt.Sprintf("%s-%s", id, sum)
}

// RateLimit checks the given key against the specified rate limit, returning true if limited.
//
// This allows bursts of up to 1/10th the given rate limit, by default.
//
// Tihs returns the duration until the next request will be permitted, or -1 if the rate limit
// has not been exceeded.
func rateLimit(ctx context.Context, store throttled.GCRAStoreCtx, key string, c inngest.RateLimit) (bool, time.Duration, error) {
	dur, err := str2duration.ParseDuration(c.Period)
	if err != nil {
		return true, -1, err
	}

	quota := throttled.RateQuota{
		MaxRate:  throttled.PerDuration(int(c.Limit), dur),
		MaxBurst: int(c.Limit) / 10,
	}

	limiter, err := throttled.NewGCRARateLimiterCtx(store, quota)
	if err != nil {
		log.Fatal(err)
	}

	ok, res, err := limiter.RateLimitCtx(ctx, key, 1)
	if err != nil {
		return ok, -1, err
	}

	return ok, res.RetryAfter, err
}
