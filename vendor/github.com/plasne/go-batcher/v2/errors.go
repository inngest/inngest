package batcher

import "errors"

const (
	AuditMsgFailureOnTargetAndInflight = "an audit revealed that the target and inflight should both be zero but neither was."
	AuditMsgFailureOnTarget            = "an audit revealed that the target should be zero but was not."
	AuditMsgFailureOnInflight          = "an audit revealed that inflight should be zero but was not."
)

var (
	NoWatcherError               = errors.New("the operation must have a watcher assigned.")
	TooManyAttemptsError         = errors.New("the operation exceeded the maximum number of attempts.")
	TooExpensiveError            = errors.New("the operation costs more than the maximum capacity.")
	BufferFullError              = errors.New("the buffer is full, try to enqueue again later.")
	BufferIsShutdown             = errors.New("the buffer is shutdown, you may no longer enqueue.")
	ImproperOrderError           = errors.New("methods can only be called in this order Start() > Stop().")
	NoOperationError             = errors.New("no operation was provided.")
	InitializationOnlyError      = errors.New("this property can only be set before Start() is called.")
	SharedCapacityNotProvisioned = errors.New("shared capacity cannot be set if it was not provisioned.")
)
