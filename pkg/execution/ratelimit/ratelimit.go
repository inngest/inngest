package ratelimit

import (
	"context"
	"fmt"
	"log"
	"strconv"

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
	RateLimit(ctx context.Context, key string, c inngest.RateLimit) (bool, error)
}

// RateLimitKey returns the rate limiting key given a function ID, rate limit config,
// and incoming event data.
func RateLimitKey(ctx context.Context, id uuid.UUID, c inngest.RateLimit, evt map[string]any) (string, error) {
	if c.Key == nil {
		return id.String(), nil
	}
	eval, err := expressions.NewExpressionEvaluator(ctx, *c.Key)
	if err != nil {
		return "", fmt.Errorf("unable to parse rate limit expression: %w", err)
	}
	res, _, err := eval.Evaluate(ctx, expressions.NewData(map[string]any{"event": evt}))
	if err != nil {
		return "", ErrEvaluatingRateLimitExpression
	}

	// Take a checksum of this data.  It doesn't matter if this is a map or a string;
	// as long as we're consistent here.
	ui := xxhash.Sum64String(fmt.Sprintf("%v", res))
	sum := strconv.FormatUint(ui, 36)

	// Use this function and checksum as the rate limit key.
	return fmt.Sprintf("%s-%s", id, sum), nil
}

// RateLimit checks the given key against the specified rate limit, returning true if limited.
func rateLimit(ctx context.Context, store throttled.GCRAStoreCtx, key string, c inngest.RateLimit) (bool, error) {
	dur, err := str2duration.ParseDuration(c.Period)
	if err != nil {
		return true, err
	}

	quota := throttled.RateQuota{
		MaxRate: throttled.PerDuration(int(c.Limit), dur),
	}

	limiter, err := throttled.NewGCRARateLimiterCtx(store, quota)
	if err != nil {
		log.Fatal(err)
	}

	ok, _, err := limiter.RateLimitCtx(ctx, key, 1)
	return ok, err
}
