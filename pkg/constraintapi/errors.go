package constraintapi

import "errors"

var ErrAccountNotFound = errors.New("constraint api: account not found")

// ConstraintAPIInternalErrorCode represents an internal error code
type ConstraintAPIInternalErrorCode int

const (
	ConstraintAPIErrorUnknown ConstraintAPIInternalErrorCode = iota
	ConstraintAPIErrorInvalidRequest
)
