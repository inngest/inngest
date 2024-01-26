package errors

import (
	"encoding/json"
	"errors"
	"time"
)

// StepError is an error returned when a step permanently fails
type StepError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	// Data is the data from state.UserError, used to store
	// the resulting value when step errors occur with an additional
	// response type.
	Data json.RawMessage `json:"data,omitempty"`
}

func (e StepError) Error() string {
	return e.Message
}

func (e StepError) Is(err error) bool {
	switch err.(type) {
	case *StepError, StepError:
		return true
	default:
		return false
	}
}

func IsStepError(err error) bool {
	return errors.Is(err, StepError{})
}

// NoRetryError wraps an error, preventing retries in the SDK.  This permanently
// fails a step and function.
func NoRetryError(err error) error {
	return noRetryError{Err: err}
}

// IsNoRetryError returns whether an error is a NoRetryError
func IsNoRetryError(err error) bool {
	return errors.Is(err, noRetryError{})
}

// noRetryError represents an error that will not be retried.
type noRetryError struct {
	Err error
}

// Error fulfils the builtin go error interface.  This returns the original root cause
// for debugging via logging - all logging interfaces & tracing packages use Error() to
// capture error information.
func (e noRetryError) Error() string {
	return e.Err.Error()
}

// Unwrap fulfils errors.Unwrap, allowing us to use the builtin error functionality to
// manage the original error, or to chain public errors.
func (e noRetryError) Unwrap() error {
	return e.Err
}

// Unwrap fulfils errors.Unwrap, allowing us to use the builtin error functionality to
// manage the original error, or to chain public errors.
func (e noRetryError) Is(target error) bool {
	switch target.(type) {
	case *noRetryError, noRetryError:
		return true
	default:
		return false
	}
}

// RetryAtError allows you to specify the time at which the next retry should occur. This
// wraps your error, leaving the original cause and error message available.
func RetryAtError(err error, at time.Time) error {
	return retryAtError{Err: err, At: at}
}

// GetRetryAtTime returns the time from a retryAtError, or nil.
func GetRetryAtTime(err error) *time.Time {
	retryAt := &retryAtError{}
	if ok := errors.As(err, retryAt); ok {
		return &retryAt.At
	}
	return nil
}

type retryAtError struct {
	Err error
	At  time.Time
}

// Error fulfils the builtin go error interface.  This returns the original root cause
// for debugging via logging - all logging interfaces & tracing packages use Error() to
// capture error information.
func (e retryAtError) Error() string {
	return e.Err.Error()
}

// Unwrap fulfils errors.Unwrap, allowing us to use the builtin error functionality to
// manage the original error, or to chain public errors.
func (e retryAtError) Unwrap() error {
	return e.Err
}

// Unwrap fulfils errors.Unwrap, allowing us to use the builtin error functionality to
// manage the original error, or to chain public errors.
func (e retryAtError) Is(target error) bool {
	switch target.(type) {
	case *retryAtError, retryAtError:
		return true
	default:
		return false
	}
}
