package inngestgo

import (
	"errors"
	"time"
)

// NoRetryError wraps an error, preventing retries in the SDK.  This permanently
// fails a step and function.
func NoRetryError(err error) error {
	return noRetryError{Err: err}
}

func IsNoRetryError(err error) bool {
	return errors.Is(err, noRetryError{})
}

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

func IsRetryAtError(err error) bool {
	return errors.Is(err, retryAtError{})
}

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
