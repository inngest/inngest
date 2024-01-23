package queue

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

type Queue interface {
	Producer
	Consumer

	JobQueueReader
}

type RunInfo struct {
	Latency      time.Duration
	SojournDelay time.Duration
	Priority     uint
}

type RunFunc func(context.Context, RunInfo, Item) error

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

// RetryAtSpecifier specifies the next retry time.  If this returns a nil pointer,
// the default retry iwll be used for the current attempt.
type RetryAtSpecifier interface {
	NextRetryAt() *time.Time
}

func RetryAtError(err error, at *time.Time) error {
	return retryAtError{cause: err, at: at}
}

type retryAtError struct {
	cause error
	at    *time.Time
}

func (r retryAtError) Error() string {
	return r.cause.Error()
}

func (r retryAtError) Unwrap() error {
	return r.cause
}

func (r retryAtError) NextRetryAt() *time.Time {
	return r.at
}

// ShouldRetry returns whether we need to retry an error.
func ShouldRetry(err error, attempt int, max int) bool {
	unwrapped := err
	for unwrapped != nil {
		if _, ok := unwrapped.(AlwaysRetryableError); ok {
			return true
		}

		// This error specifies internally whether it should be retried.
		retryable, isRetry := unwrapped.(RetryableError)
		if isRetry && !retryable.Retryable() {
			// The error says no;  cannot be bypassed.
			return false
		}

		unwrapped = errors.Unwrap(unwrapped)
	}

	// So, at this point we either have a basic, non-interface error OR
	// a retryable error which returns Retryable() true.
	//
	// Note that attempt is 0-indexed, hence attempt+1.
	return (attempt + 1) < max
}

func NeverRetryError(err error) error {
	return nonRetryable{error: err}
}

type nonRetryable struct {
	error
}

func (nonRetryable) Retryable() bool { return false }

// AlwaysRetryError always retries, ignoring max retry counts
func AlwaysRetryError(err error) error {
	return alwaysRetry{error: err}
}

type alwaysRetry struct {
	error
}

func (a alwaysRetry) AlwaysRetryable() {}

type JobResponse struct {
	// At represents the time the job is scheduled for.
	At time.Time `json:"at"`
	// Position represents the position for the job in the queue
	Position int64 `json:"position"`
	// Kind represents the kind of job in the queue.
	Kind string `json:"kind"`
	// Attempt
	Attempt int `json:"attempt"`
}

// JobQueueReader
type JobQueueReader interface {
	// OutstandingJobCount returns the number of jobs in progress
	// or scheduled for a given run.
	OutstandingJobCount(
		ctx context.Context,
		workspaceID uuid.UUID,
		workflowID uuid.UUID,
		runID ulid.ULID,
	) (int, error)

	// StatusCount returns the total number of items in the function
	// status queue.
	StatusCount(
		ctx context.Context,
		workflowID uuid.UUID,
		status string,
	) (int64, error)

	RunJobs(
		ctx context.Context,
		workspaceID uuid.UUID,
		workflowID uuid.UUID,
		runID ulid.ULID,
		limit,
		offset int64,
	) ([]JobResponse, error)
}
