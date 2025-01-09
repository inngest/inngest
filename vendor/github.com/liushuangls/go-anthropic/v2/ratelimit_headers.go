package anthropic

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type rateLimitHeaderKey string

const (
	requestsLimit     rateLimitHeaderKey = "anthropic-ratelimit-requests-limit"
	requestsRemaining rateLimitHeaderKey = "anthropic-ratelimit-requests-remaining"
	requestsReset     rateLimitHeaderKey = "anthropic-ratelimit-requests-reset"
	tokensLimit       rateLimitHeaderKey = "anthropic-ratelimit-tokens-limit"
	tokensRemaining   rateLimitHeaderKey = "anthropic-ratelimit-tokens-remaining"
	tokensReset       rateLimitHeaderKey = "anthropic-ratelimit-tokens-reset"
	retryAfter        rateLimitHeaderKey = "retry-after"
)

type RateLimitHeaders struct {
	// The maximum number of requests allowed within the rate limit window.
	RequestsLimit int `json:"anthropic-ratelimit-requests-limit"`
	// The number of requests remaining within the current rate limit window.
	RequestsRemaining int `json:"anthropic-ratelimit-requests-remaining"`
	// The time when the request rate limit window will reset, provided in RFC 3339 format.
	RequestsReset time.Time `json:"anthropic-ratelimit-requests-reset"`
	// The maximum number of tokens allowed within the rate limit window.
	TokensLimit int `json:"anthropic-ratelimit-tokens-limit"`
	// The number of tokens remaining, rounded to the nearest thousand, within the current rate limit window.
	TokensRemaining int `json:"anthropic-ratelimit-tokens-remaining"`
	// The time when the token rate limit window will reset, provided in RFC 3339 format.
	TokensReset time.Time `json:"anthropic-ratelimit-tokens-reset"`
	// The number of seconds until the rate limit window resets.
	RetryAfter int `json:"retry-after"`
}

func newRateLimitHeaders(h http.Header) (RateLimitHeaders, error) {
	var errs []error

	parseIntHeader := func(key rateLimitHeaderKey, required bool) int {
		value, err := strconv.Atoi(h.Get(string(key)))
		if err != nil {
			if !required {
				return -1
			}
			errs = append(errs, fmt.Errorf("failed to parse %s: %w", key, err))
			return 0
		}
		return value
	}

	parseTimeHeader := func(key rateLimitHeaderKey, required bool) time.Time {
		value, err := time.Parse(time.RFC3339, h.Get(string(key)))
		if err != nil {
			if !required {
				return time.Time{}
			}
			errs = append(errs, fmt.Errorf("failed to parse %s: %w", key, err))
			return time.Time{}
		}
		return value
	}

	headers := RateLimitHeaders{}
	headers.RequestsLimit = parseIntHeader(requestsLimit, true)
	headers.RequestsRemaining = parseIntHeader(requestsRemaining, true)
	headers.RequestsReset = parseTimeHeader(requestsReset, true)

	headers.TokensLimit = parseIntHeader(tokensLimit, true)
	headers.TokensRemaining = parseIntHeader(tokensRemaining, true)
	headers.TokensReset = parseTimeHeader(tokensReset, true)

	headers.RetryAfter = parseIntHeader(retryAfter, false) // optional

	if len(errs) > 0 {
		return headers, fmt.Errorf("error(s) parsing rate limit headers: %w",
			errors.Join(errs...),
		)
	}

	return headers, nil
}
