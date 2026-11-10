package domain

import "github.com/inngest/inngest/pkg/util/errs"

// External holds a result or an error from an external source such as an SDK.
// the zero value is a successful result with a zero value of T.  callers must
// check Err before using Value so a failed SDK request is not treated as a success.
type External[T any] struct {
	value T
	err   errs.UserError
}

// ExternalValue returns a successful result, including when value is zero.
func ExternalValue[T any](value T) External[T] {
	return External[T]{value: value}
}

// ExternalError returns a failed result with the external error's retry policy.
// a nil error returns the successful zero value.
func ExternalError[T any](err errs.UserError) External[T] {
	return External[T]{err: err}
}

// Valid reports whether the result is successful, including a zero value of T.
func (e External[T]) Valid() bool {
	return e.err == nil
}

// HasError reports whether the result contains an external error.
func (e External[T]) HasError() bool {
	return e.err != nil
}

// Value returns the successful value, or the zero value of T after a failure.
func (e External[T]) Value() T {
	return e.value
}

// Err returns the external error so callers can apply its retry policy.
func (e External[T]) Err() errs.UserError {
	return e.err
}

// Or returns the successful value, or def after a failure.
func (e External[T]) Or(def T) T {
	if e.HasError() {
		return def
	}
	return e.value
}

// Unpack returns the value and external error for callers that use separate returns.
func (e External[T]) Unpack() (T, errs.UserError) {
	return e.value, e.err
}

// UnpackErr returns the value and external error for callers that use separate returns,
// as an error interface.
func (e External[T]) UnpackErr() (T, error) {
	return e.value, e.err
}
