package queue

type PeekResult[T any] struct {
	Items        []*T
	TotalCount   int
	RemovedCount int

	// Cursor represents the score of the last item in the peek result.
	// This can be used for pagination within iterators
	Cursor int64
}
