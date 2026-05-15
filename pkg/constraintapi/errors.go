package constraintapi

// ConstraintAPIInternalErrorCode represents an internal error code
type ConstraintAPIInternalErrorCode int

const (
	ConstraintAPIErrorUnknown ConstraintAPIInternalErrorCode = iota
	ConstraintAPIErrorInvalidRequest
)
