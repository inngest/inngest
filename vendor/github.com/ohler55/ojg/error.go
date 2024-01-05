// Copyright (c) 2021, Peter Ohler, All rights reserved.

package ojg

import (
	"fmt"
	"runtime/debug"
)

// ErrorWithStack if true the Error() call will include the stack.
var ErrorWithStack = false

// Error struct to hold an error message and a stack trace.
type Error struct {
	msg   string
	stack []byte
}

// NewError creates a new Error instance, capturing the stack when created.
func NewError(r any) *Error {
	return &Error{
		msg:   fmt.Sprintf("%v", r),
		stack: debug.Stack(),
	}
}

// Error returns a string representation of the instance.
func (err *Error) Error() string {
	if ErrorWithStack {
		return string(append(append([]byte(err.msg), '\n'), err.stack...))
	}
	return err.msg
}

// Stack returns the stack.
func (err *Error) Stack() []byte {
	return err.stack
}
