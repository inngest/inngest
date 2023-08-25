package inngestgo

import (
	"errors"
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
