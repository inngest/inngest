package apiv2

import "context"

// RateLimitProvider checks whether an API request should be rate limited.
// Cloud implementations use per-account limits; the default noop allows all.
type RateLimitProvider interface {
	// CheckRateLimit determines if the request is allowed.
	// method is a gRPC full method name (e.g. apiv2.V2_InvokeFunction_FullMethodName).
	CheckRateLimit(ctx context.Context, method string) RateLimitResult
}

// RateLimitResult holds the outcome of a rate limit check.
type RateLimitResult struct {
	Limited bool
}

// noopRateLimitProvider always allows requests.
type noopRateLimitProvider struct{}

func (noopRateLimitProvider) CheckRateLimit(context.Context, string) RateLimitResult {
	return RateLimitResult{}
}
