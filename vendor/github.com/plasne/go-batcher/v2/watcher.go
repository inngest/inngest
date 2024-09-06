package batcher

import "time"

type Watcher interface {
	WithMaxAttempts(val uint32) Watcher
	WithMaxBatchSize(val uint32) Watcher
	WithMaxOperationTime(val time.Duration) Watcher
	MaxAttempts() uint32
	MaxBatchSize() uint32
	MaxOperationTime() time.Duration
	ProcessBatch(ops []Operation)
}

type watcher struct {
	maxAttempts      uint32
	maxBatchSize     uint32
	maxOperationTime time.Duration
	onReady          func(ops []Operation)
}

// This method creates a new Watcher with a callback function. This function will be called whenever a batch of Operations is ready to be
// processed. When the callback function is completed, it will reduce the Target by the cost of all Operations in the batch. If for some
// reason the processing is "stuck" in this function, they Target will be reduced after MaxOperationTime. Every time this function is called
// with a batch it is run as a new goroutine so anything inside could cause race conditions with the rest of your code - use atomic, sync,
// etc. as appropriate.
func NewWatcher(onReady func(batch []Operation)) Watcher {
	return &watcher{
		onReady: onReady,
	}
}

// If there are transient errors, you can enqueue the same Operation again. If you do not provide MaxAttempts, it will allow you to enqueue
// as many times as you like. Instead, if you specify MaxAttempts, the Enqueue() method will return `TooManyAttemptsError` if you attempt
// to enqueue it too many times.
func (w *watcher) WithMaxAttempts(val uint32) Watcher {
	w.maxAttempts = val
	return w
}

// This determines the maximum number of Operations that will be raised in a single batch. This does not guarantee that batches will be of
// this size (constraints such rate limiting might reduce the size), but it does guarantee they will not be larger.
func (w *watcher) WithMaxBatchSize(val uint32) Watcher {
	w.maxBatchSize = val
	return w
}

// This determines how long the system should wait for the callback function to be completed on the batch before it assumes it is done and
// decreases the Target anyway. It is critical that the Target reflect the current cost of outstanding Operations. The MaxOperationTime
// ensures that a batch isn't orphaned and continues reserving capacity long after it is no longer needed. If MaxOperationTime is not provided
// on the Watcher, the Batcher MaxOperationTime is used.
func (w *watcher) WithMaxOperationTime(val time.Duration) Watcher {
	w.maxOperationTime = val
	return w
}

// If there are transient errors, you can enqueue the same Operation again. If you do not provide MaxAttempts, it will allow you to enqueue
// as many times as you like. Instead, if you specify MaxAttempts, the Enqueue() method will return `TooManyAttemptsError` if you attempt
// to enqueue it too many times.
func (w *watcher) MaxAttempts() uint32 {
	return w.maxAttempts
}

// This determines the maximum number of Operations that will be raised in a single batch. This does not guarantee that batches will be of
// this size (constraints such rate limiting might reduce the size), but it does guarantee they will not be larger.
func (w *watcher) MaxBatchSize() uint32 {
	return w.maxBatchSize
}

// This determines how long the system should wait for the callback function to be completed on the batch before it assumes it is done and
// decreases the Target anyway. It is critical that the Target reflect the current cost of outstanding Operations. The MaxOperationTime
// ensures that a batch isn't orphaned and continues reserving capacity long after it is no longer needed. If MaxOperationTime is not provided
// on the Watcher, the Batcher MaxOperationTime is used.
func (w *watcher) MaxOperationTime() time.Duration {
	return w.maxOperationTime
}

// This is used internally by Batcher to process a batch of Operations using the callback function. You should generally not call this method,
// but you might mock it for unit tests.
func (w *watcher) ProcessBatch(batch []Operation) {
	w.onReady(batch)
}
