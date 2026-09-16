package constraintapi

import "errors"

var ErrAccountNotFound = errors.New("constraint api: account not found")

// ErrConstraintShardNotFound indicates that a caller cannot route an account
// to its configured constraint shard. Unlike transport failures, retrying this
// error cannot succeed until the account or deployment configuration changes.
var ErrConstraintShardNotFound = errors.New("constraint api: configured shard not found")

// ConstraintAPIInternalErrorCode represents an internal error code
type ConstraintAPIInternalErrorCode int

const (
	ConstraintAPIErrorUnknown ConstraintAPIInternalErrorCode = iota
	ConstraintAPIErrorInvalidRequest
)
