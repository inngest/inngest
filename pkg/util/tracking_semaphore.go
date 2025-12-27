package util

import (
	"context"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

// trackingSemaphore returns a semaphore that tracks closely - but not atomically -
// the total number of items in the semaphore.  This is best effort, and is loosely
// accurate to reduce further contention.
//
// This is only used as an indicator as to whether to scan.
type trackingSemaphore struct {
	*semaphore.Weighted
	counter int64
}

type TrackingSemaphore interface {
	TryAcquire(n int64) bool
	Acquire(ctx context.Context, n int64) error
	Release(n int64)
	Count() int64
}

func NewTrackingSemaphore(num int) TrackingSemaphore {
	return &trackingSemaphore{Weighted: semaphore.NewWeighted(int64(num))}
}

func (t *trackingSemaphore) TryAcquire(n int64) bool {
	if !t.Weighted.TryAcquire(n) {
		return false
	}
	atomic.AddInt64(&t.counter, n)
	return true
}

func (t *trackingSemaphore) Acquire(ctx context.Context, n int64) error {
	if err := t.Weighted.Acquire(ctx, n); err != nil {
		return err
	}
	atomic.AddInt64(&t.counter, n)
	return nil
}

func (t *trackingSemaphore) Release(n int64) {
	t.Weighted.Release(n)
	atomic.AddInt64(&t.counter, -n)
}

func (t *trackingSemaphore) Count() int64 {
	return atomic.LoadInt64(&t.counter)
}
