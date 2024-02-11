package runstate

import "context"

type PauseIterator interface {
	// Next returns the next batch of pauses in the iterator.  The number of
	// pauses in each batch depends on the backing store implementation.
	//
	// This is useable with Go 1.22's iterator range support:
	//
	//	for batch, err := range iter.Next(ctx) {
	//		if err != nil {
	//			return err
	//		}
	//		// work with batch.
	//	}
	Next(ctx context.Context) func(yield func([]any, error) bool)
	// Count returns either an approximation or the total number of items
	// to be returned in the iterator.
	Count(ctx context.Context) int64
}
