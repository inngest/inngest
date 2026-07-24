package queue

import "sync"

type DispatchedItem interface {
	Done() <-chan DispatchedItemResult
}

type DispatchedItemResult struct {
	ScheduledImmediateJob bool
	Err                   error
}

type dispatchedItemHandle struct {
	once sync.Once
	done chan DispatchedItemResult
}

func newDispatchedItemHandle() *dispatchedItemHandle {
	return &dispatchedItemHandle{
		// Buffered so worker completion never blocks if no scanner is waiting.
		done: make(chan DispatchedItemResult, 1),
	}
}

// NewCompletedDispatchedItem returns a dispatched item that has already completed.
// This is useful for scanner implementations or tests that synchronously process dispatch.
func NewCompletedDispatchedItem(result DispatchedItemResult) DispatchedItem {
	handle := newDispatchedItemHandle()
	handle.complete(result)
	return handle
}

func (h *dispatchedItemHandle) Done() <-chan DispatchedItemResult {
	return h.done
}

func (h *dispatchedItemHandle) complete(result DispatchedItemResult) {
	h.once.Do(func() {
		h.done <- result
		close(h.done)
	})
}
