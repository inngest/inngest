package errs

import "fmt"

// InternalError represents an internal error that isn't from the user's servers.
type InternalError interface {
	error

	// ErrorCode returns the error code for this error.
	ErrorCode() int

	// Retryable indicates whether this is retryable.
	Retryable() bool

	// InternalError indicates that this is a user error.  This is a noop and allows
	// implementations of error classes to assert that they're a user error specifically.
	InternalError()
}

// UserError represents an error from the user's server, either as a 500 or because of
// some other error **directly out of our control**.
type UserError interface {
	error

	// ErrorCode returns the error code for this error.
	ErrorCode() int

	// Retryable indicates whether this is retryable.
	Retryable() bool

	// UserError indicates that this is a user error.  This is a noop and allows
	// implementations of error classes to assert that they're a user error specifically.
	UserError()

	// Raw returns the raw data for the error.  This may be the raw HTTP response,
	// the raw error string, and so on.
	Raw() []byte
}

// Wrap always wraps an error as an InternalError type.
func Wrap(code int, retryable bool, msg string, a ...any) InternalError {
	return internal{
		error:     fmt.Errorf(msg, a...),
		code:      code,
		retryable: retryable,
	}
}

// WrapUser always wraps an error as an InternalError type.
func WrapUser(code int, retryable bool, msg string, a ...any) UserError {
	return user{
		error:     fmt.Errorf(msg, a...),
		code:      code,
		retryable: retryable,
	}
}

// WrapResponseAsUser wraps a raw HTTP response as a user error, exposing the raw
// response via the `.Raw()` method.
func WrapResponseAsUser(code int, retryable bool, raw []byte, msg string, a ...any) UserError {
	return user{
		error:     fmt.Errorf(msg, a...),
		code:      code,
		retryable: retryable,
		raw:       raw,
	}
}

type internal struct {
	error
	retryable bool
	code      int
}

// Unwrap allows us to use errors.Is to determine the proper
// cause of errors.
func (i internal) Unwrap() error   { return i.error }
func (i internal) ErrorCode() int  { return i.code }
func (i internal) Retryable() bool { return i.retryable }
func (internal) InternalError()    {}

type user struct {
	error
	retryable bool
	code      int
	raw       []byte
}

// Unwrap allows us to use errors.Is to determine the proper
// cause of errors.
func (u user) Unwrap() error   { return u.error }
func (u user) ErrorCode() int  { return u.code }
func (u user) Retryable() bool { return u.retryable }
func (u user) Raw() []byte     { return u.raw }
func (user) UserError()        {}
