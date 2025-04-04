package util

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/logger"
)

var ErrMaxAttemptReached = errors.New("maximum retry attempts reached")

// Retryable represents a function that can be retried
type Retryable[T any] func(ctx context.Context) (T, error)

// RetryConf specifies the control of how a function should be retried
type RetryConf struct {
	MaxAttempts     int              `json:"max_attempts"`
	InitialBackoff  time.Duration    `json:"initial_backoff"`
	MaxBackoff      time.Duration    `json:"max_backoff"`
	BackoffFactor   int              `json:"backoff_factor"`
	RetryableErrors func(error) bool `json:"-"`
}

type RetryConfSetting func(rc *RetryConf)

func WithRetryConfMaxAttempts(i int) RetryConfSetting {
	return func(rc *RetryConf) {
		rc.MaxAttempts = i
	}
}

func WithRetryConfInitialBackoff(dur time.Duration) RetryConfSetting {
	return func(rc *RetryConf) {
		rc.InitialBackoff = dur
	}
}

func WithRetryConfMaxBackoff(dur time.Duration) RetryConfSetting {
	return func(rc *RetryConf) {
		rc.MaxBackoff = dur
	}
}

func WithRetryConfBackoffFactor(i int) RetryConfSetting {
	return func(rc *RetryConf) {
		rc.BackoffFactor = i
	}
}

func WithRetryConfRetryableErrors(fn func(error) bool) RetryConfSetting {
	return func(rc *RetryConf) {
		rc.RetryableErrors = fn
	}
}

func NewRetryConf(opts ...RetryConfSetting) RetryConf {
	conf := RetryConf{
		MaxAttempts:     5,
		InitialBackoff:  100 * time.Millisecond,
		MaxBackoff:      5 * time.Second,
		BackoffFactor:   2,
		RetryableErrors: nil, // by default retry all errors
	}

	for _, apply := range opts {
		apply(&conf)
	}

	return conf
}

// WithRetry wraps a function with retry logic and returns the result of the successful call,
// or the error after retries have been exhausted
func WithRetry[T any](ctx context.Context, fn Retryable[T], conf RetryConf) (T, error) {
	var (
		result  T
		lastErr error
	)

	l := logger.StdlibLogger(ctx)
	backoff := conf.InitialBackoff

	for attempt := 1; attempt <= conf.MaxAttempts; attempt++ {
		// run the inner function
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		l.Warn("error on retriable function attempt",
			"error", err,
			"attempt", attempt,
			"conf", conf,
		)

		lastErr = err
		if attempt == conf.MaxAttempts {
			break
		}

		// check if the error returned should be retried.
		// if not, return as is
		if conf.RetryableErrors != nil && !conf.RetryableErrors(err) {
			return result, err
		}

		// calculate next backoff
		nextBackoff := backoff * time.Duration(conf.BackoffFactor)
		if nextBackoff >= conf.MaxBackoff {
			nextBackoff = conf.MaxBackoff
		}

		// create a timer for backoff
		timer := time.NewTimer(backoff)
		select {
		case <-timer.C:
			// continue to next attempt
		case <-ctx.Done():
			timer.Stop()
			return result, fmt.Errorf("stopping retry due to error: %w. last error: %w", ctx.Err(), lastErr)
		}

		backoff = nextBackoff
	}

	l.Error("retriable function failed",
		"error", lastErr,
		"conf", conf,
	)
	return result, fmt.Errorf("%w: %v", ErrMaxAttemptReached, lastErr)
}
