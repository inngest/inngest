package errs

import (
	"fmt"
	"time"
)

type Error interface {
	error

	// ErrorCode returns the error code for this error.
	ErrorCode() int

	// Retryable indicates whether this is retryable.
	Retryable() bool

	RetryAfter() time.Duration
}

// InternalError represents an internal error that isn't from the user's servers.
type InternalError interface {
	Error

	// InternalError indicates that this is a user error.  This is a noop and allows
	// implementations of error classes to assert that they're a user error specifically.
	InternalError()
}

// UserError represents an error from the user's server, either as a 500 or because of
// some other error **directly out of our control**.
type UserError interface {
	Error

	// UserError indicates that this is a user error.  This is a noop and allows
	// implementations of error classes to assert that they're a user error specifically.
	UserError()

	// Raw returns the raw data for the error.  This may be the raw HTTP response,
	// the raw error string, and so on.
	Raw() []byte
}

type InternalRetriableError interface {
	InternalError
	RetryAfter() time.Duration
}

type UserRetriableError interface {
	UserError
	RetryAfter() time.Duration
}

// Wrap always wraps an error as an InternalError type.
func Wrap(code int, retryable bool, msg string, a ...any) InternalError {
	return &internal{
		error:     fmt.Errorf(msg, a...),
		code:      code,
		retryable: retryable,
	}
}

// WrapAfter always wraps an error as an InternalError type.
func WrapAfter(code int, retryAfter time.Duration, msg string, a ...any) InternalError {
	return &internal{
		error:      fmt.Errorf(msg, a...),
		code:       code,
		retryable:  true,
		retryAfter: retryAfter,
	}
}

// WrapUser always wraps an error as an InternalError type.
func WrapUser(code int, retryable bool, msg string, a ...any) UserError {
	return &user{
		error:     fmt.Errorf(msg, a...),
		code:      code,
		retryable: retryable,
	}
}

func WrapAfterUser(code int, retryAfter time.Duration, msg string, a ...any) UserError {
	return &user{
		error:      fmt.Errorf(msg, a...),
		code:       code,
		retryable:  true,
		retryAfter: retryAfter,
	}
}

// WrapResponseAsUser wraps a raw HTTP response as a user error, exposing the raw
// response via the `.Raw()` method.
func WrapResponseAsUser(code int, retryable bool, raw []byte, msg string, a ...any) UserError {
	return &user{
		error:     fmt.Errorf(msg, a...),
		code:      code,
		retryable: retryable,
		raw:       raw,
	}
}

type internal struct {
	error
	retryable  bool
	code       int
	retryAfter time.Duration
}

// Unwrap allows us to use errors.Is to determine the proper
// cause of errors.
func (i *internal) Unwrap() error             { return i.error }
func (i *internal) ErrorCode() int            { return i.code }
func (i *internal) Retryable() bool           { return i.retryable }
func (i *internal) RetryAfter() time.Duration { return i.retryAfter }
func (*internal) InternalError()              {}

type user struct {
	error
	retryable  bool
	code       int
	raw        []byte
	retryAfter time.Duration
}

// Unwrap allows us to use errors.Is to determine the proper
// cause of errors.
func (u *user) Unwrap() error             { return u.error }
func (u *user) ErrorCode() int            { return u.code }
func (u *user) Retryable() bool           { return u.retryable }
func (u *user) RetryAfter() time.Duration { return u.retryAfter }
func (u *user) Raw() []byte               { return u.raw }
func (*user) UserError()                  {}

func NewError(
	isUser bool,
	code int,
	retryable bool,
	retryAfter time.Duration,
	msg string,
	a ...any,
) error {
	switch {
	case isUser && retryable:
		return WrapAfterUser(code, retryAfter, msg, a...)
	case isUser && !retryable:
		return WrapUser(code, false, msg, a...)
	case !isUser && retryable:
		return WrapAfter(code, retryAfter, msg, a...)
	default:
		return Wrap(code, false, msg, a...)
	}
}
