// Package ierrors provides Inngest specific errors across all of our system, including
// syncs, internal errors, request/networking errors, execution errors, etc.
package ierrors

const (
	ErrorCodeUnknown = "0.001"
)

func New[T error](t T) InngestError[T] {
	return ierror[T]{Cause: t}
}

type InngestError[T error] interface {
	error

	Kind() T

	ErrorCode() string

	// Warning indicates whether the error is a warning, eg. a function's URL changes
	// in between runs *without* the run erroring.
	//
	// A warning should be marked as such;  it does not fail runs.
	Warning() bool

	// Metadata returns metadata regarding the error, eg. the status code and headers
	// if a request to the SDK throws a 404.
	//
	// Metadata changes depending on the type of error.
	Metadata() map[string]any

	// Retryable indicates whether this error is retryable in our queue.  If this error
	// is thrown in an exeuction path within the queue, this function dictates whether
	// the step is retried.
	Retryable() bool
}

// ierror represents an inngest error.
type ierror[T error] struct {
	Cause T
}

func (e ierror[T]) Error() string {
	return e.Cause.Error()
}

// Unwrap fulfils errors.Unwrap, allowing us to use the builtin error functionality to
// manage the original error, or to chain public errors.
func (e ierror[T]) Unwrap() error {
	return e.Cause
}

// TODO
// func (e ierror[T]) Is(err error) bool {
// 	_, nonptr := err.(ierror[T])
// 	_, ptr := err.(*ierror[T])
// 	return nonptr || ptr
// }

func (e ierror[T]) Kind() T {
	return e.Cause
}

func (e ierror[T]) ErrorCode() string {
	if c, ok := any(e.Cause).(ErrorCoder); ok {
		code := c.ErrorCode()
		if code == "" {
			return ErrorCodeUnknown
		}
	}
	return ErrorCodeUnknown
}

func (e ierror[T]) Warning() bool {
	// TODO: Not implemented.
	return false
}

func (e ierror[T]) Metadata() map[string]any {
	// TODO: Not implemented.
	return nil
}

func (e ierror[T]) Retryable() bool {
	switch v := any(e.Cause).(type) {
	case InternalExecutionError:
		return v.Retryable()
	default:
		return true
	}
}

// TODO: Marshalling/Unmarshalling.

// ErrorCoder represents an error that returns an error code.
type ErrorCoder interface {
	error
	ErrorCode() string
}

type InternalExecutionError struct {
	Err error
	// Code represents the error code for this error.  If left blank this defaults
	// to ErrorCodeUnknown.
	Code string
	// Final indicates whether the error is non-retryable.  Its flipped such
	// that the default is true, eg. always retry on a newly initialized error.
	Final bool
}

func (e InternalExecutionError) Retryable() bool {
	return !e.Final
}

func (e InternalExecutionError) Error() string {
	return e.Err.Error()
}

func (e InternalExecutionError) ErrorCode() string {
	return e.Code
}
