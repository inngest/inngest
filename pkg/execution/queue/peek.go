package queue

type PeekResult[T any] struct {
	Items        []*T
	TotalCount   int
	RemovedCount int

	// Cursor represents the score of the last item in the peek result.
	// This can be used for pagination within iterators
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
