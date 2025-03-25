package util

import (
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

func ParallelDecode[T any](in []any, process func(any) (T, error)) ([]T, error) {
	var (
		ctr int32
		eg  = errgroup.Group{}
	)

	// parallel/concurrent JSON decoding.  this improves perf by *at least* 50% on a
	// same-machine lookup.  on networked machines, slightly less.
	for n, str := range in {
		item := str
		idx := n
		eg.Go(func() error {
			processed, err := process(item)
			if err != nil {
				// Unset the item in the slice
				in[idx] = nil
				return err
			}
			// Make the slice member the decoded/processed item
			in[idx] = processed
			// Increase counter of processed items.  This lets us allocate a resulting
			// array in the type of []T at the correct size and capacity.
			atomic.AddInt32(&ctr, 1)
			return nil
		})
	}

	// Wait for all aprsing to finish.
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	n := 0
	result := make([]T, ctr)
	for _, item := range in {
		if item == nil {
			continue
		}
		if v, ok := item.(T); ok {
			result[n] = v
			n++
		}
	}

	return result, nil
}
