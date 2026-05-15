package queue

type PeekResult[T any] struct {
	Items        []*T
	TotalCount   int
	RemovedCount int

	// Cursor represents the score of the last item in the peek result.
	// This can be used for pagination within iterators
	Cursor int64
}

// BacklogPeekResult is the result of a BacklogPeek call.
type BacklogPeekResult struct {
	Items      []*QueueItem
	TotalCount int

	// Cursor is the sorted-set score (millisecond timestamp) of the last
	// item fetched from the backlog sorted set.  Callers must use this —
	// instead of item AtMS — to advance pagination cursors, because AtMS
	// can diverge from the sorted-set score when items are retried or
	// rescheduled.
	Cursor int64
}

type PeekOpt func(p *PeekOption)

type PeekOption struct {
	IgnoreCleanup bool
}

// WithPeekOptIgnoreCleanup will prevent missing items from being deleted.
func WithPeekOptIgnoreCleanup() PeekOpt {
	return func(p *PeekOption) {
		p.IgnoreCleanup = true
	}
}
