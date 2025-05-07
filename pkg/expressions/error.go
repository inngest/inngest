package expressions

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
)

var (
	ErrCompileFailed    = fmt.Errorf("expression compilation failed")
	ErrValidationFailed = fmt.Errorf("validation failed")
)

type CompileError struct {
	err error
	msg string
}

func NewCompileError(err error) *CompileError {
	return &CompileError{
		err: multierror.Append(ErrCompileFailed, err),
		msg: err.Error(),
	}
}

func (c *CompileError) Error() string {
	return fmt.Sprintf("error compiling expression: %s", c.msg)
}

func (c *CompileError) Unwrap() error {
	return c.err
}

func (c *CompileError) Message() string {
	return c.msg
}

func (c *CompileError) Is(tgt error) bool {
	_, ok := tgt.(*CompileError)
	return ok
}

type validationError struct {
	err error
	msg string
}

func newValidationErr(err error) error {
	if err == nil {
		return &validationError{
			err: ErrValidationFailed,
			msg: ErrValidationFailed.Error(),
		}
	} else {
		return &validationError{
			err: multierror.Append(ErrValidationFailed, err),
			msg: err.Error(),
		}
	}
}

func (v *validationError) Error() string {
	return fmt.Sprintf("validation failed: %s", v.msg)
}

func (v *validationError) Unwrap() error {
	return v.err
}

func (v *validationError) Message() string {
	return v.msg
}
