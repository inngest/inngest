package batcher

import (
	"sync/atomic"
)

type Operation interface {
	Payload() interface{}
	Attempt() uint32
	Cost() uint32
	Watcher() Watcher
	IsBatchable() bool
	MakeAttempt()
}

type operation struct {
	cost      uint32
	attempt   uint32
	batchable bool
	watcher   Watcher
	payload   interface{}
}

// This method creates a new Operation with a Watcher, cost, payload, and a flag determining whether or not the Operation is batchable.
// An Operation will be Enqueued into a Batcher.
func NewOperation(watcher Watcher, cost uint32, payload interface{}, batchable bool) Operation {
	return &operation{
		watcher:   watcher,
		cost:      cost,
		payload:   payload,
		batchable: batchable,
	}
}

// This will return the payload object for the Operation.
func (o *operation) Payload() interface{} {
	return o.payload
}

// This will return the number of times this Operation has been returned to its Watcher (for instance, the first time a Watcher sees the
// Operation in a batch, Attempt() will be equal to 1). This is used by MaxAttempts on a Watcher to ensure that the Operation is not retried
// more times than is allowed.
func (o *operation) Attempt() uint32 {
	return atomic.LoadUint32(&o.attempt)
}

// This is used internally by Batcher to increment the Attempts on the Operation. You should generally not call this method, but you might mock
// it for unit tests.
func (o *operation) MakeAttempt() {
	atomic.AddUint32(&o.attempt, 1)
}

// This is the cost of the Operation. The cost of a single Operation cannot exceed the rate limiter's MaxCapacity or a `TooExpensiveError`
// error will be thrown.
func (o *operation) Cost() uint32 {
	return o.cost
}

// This is the Watcher associated with this Operation. Operations are batched by Watcher.
func (o *operation) Watcher() Watcher {
	return o.watcher
}

// This is TRUE if the Operation can be batched with other Operations. If FALSE, it will be raised to the Watcher callback in a batch by itself.
func (o *operation) IsBatchable() bool {
	return o.batchable
}
