package queue

import (
	"context"
	"time"
)

const (
	// DefaultRetryCount is used when no retry count for a job is specified.
	DefaultRetryCount = 3
)

type Queue interface {
	Producer
	Consumer
}

type RunFunc func(context.Context, Item) error

type Producer interface {
	// Enqueue allows an item to be enqueued ton run at the given time.
	Enqueue(context.Context, Item, time.Time) error
}

type Consumer interface {
	// Run is a blocking function which listens to the queue and executes the
	// given function each time a new Item becomes available.
	//
	// If the error from RunFunc is of type QuitError, the Run function will
	// always requeue the job as a retry and terminate.
	//
	// If the error from RunFunc is of type RetryableError, the job will be
	// re-enqueued if Retryable() returns true.  For all other errors, the
	// job will automatically be retried.
	Run(context.Context, RunFunc) error
}

// QuitError is an error that, when returned, quits the queue.  This always retries
// an error.
type QuitError interface {
	AlwaysRetryableError
	Quit()
}

// RetryableError represents an error that, when returned, optionally specifies
// whether the job can be retried.
type RetryableError interface {
	Retryable() bool
}

// AlwaysRetryableError ignores MaxAttempts and always retries a job.
type AlwaysRetryableError interface {
	AlwaysRetryable()
}

// ShouldRetry returns whether we need to retry an error.
func ShouldRetry(err error, attempt int, max int) bool {
	if _, ok := err.(AlwaysRetryableError); ok {
		return true
	}

	// This error specifies internally whether it should be retried.
	retryable, isRetry := err.(RetryableError)
	if isRetry && !retryable.Retryable() {
		// The error says no;  cannot be bypassed.
		return false
	}

	// So, at this point we either have a basic, non-interface error OR
	// a retryable error which returns Retryable() true.
	// Check max attempts.
	return attempt < max
}
