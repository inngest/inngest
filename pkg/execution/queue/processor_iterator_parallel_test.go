package queue

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProcessorIteratorConcurrentLeaseErrors(t *testing.T) {
	leaseErr := errors.New("injected Redis lease failure")
	items := make([]*QueueItem, 100)
	for i := range items {
		items[i] = &QueueItem{}
	}
	iterator := ProcessorIterator{
		Partition: &QueuePartition{},
		Queue:     &mockQueueProcessor{opts: NewQueueOptions()},
		Items:     items,
		Parallel:  true,
		Leaser: mockQueueItemLeaser{fn: func(context.Context, LeaseItemRequest, DispatchFunc) (LeaseItemResult, error) {
			return LeaseItemResult{Status: LeaseItemStatusLeaseError, Err: leaseErr}, leaseErr
		}},
	}
	require.ErrorIs(t, iterator.Iterate(context.Background()), leaseErr)
	require.ErrorIs(t, iterator.Err, leaseErr)
}

func TestProcessorIteratorParallelClaimLimit(t *testing.T) {
	for _, tc := range []struct {
		name     string
		parallel bool
		limit    int
		max      int32
	}{
		{name: "fifo", limit: 8, max: 1},
		{name: "one claim", parallel: true, limit: 1, max: 1},
		{name: "eight claims", parallel: true, limit: 8, max: 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var active, peak atomic.Int32
			items := make([]*QueueItem, 100)
			for i := range items {
				items[i] = &QueueItem{}
			}
			iterator := ProcessorIterator{
				Partition: &QueuePartition{}, Queue: &mockQueueProcessor{opts: NewQueueOptions()},
				Items: items, Parallel: tc.parallel, ParallelLimit: tc.limit,
				Leaser: mockQueueItemLeaser{fn: func(context.Context, LeaseItemRequest, DispatchFunc) (LeaseItemResult, error) {
					n := active.Add(1)
					for {
						old := peak.Load()
						if n <= old || peak.CompareAndSwap(old, n) {
							break
						}
					}
					time.Sleep(2 * time.Millisecond)
					active.Add(-1)
					return LeaseItemResult{Status: LeaseItemStatusDispatched}, nil
				}},
			}
			require.NoError(t, iterator.Iterate(context.Background()))
			require.LessOrEqual(t, peak.Load(), tc.max)
			if tc.parallel && tc.limit > 1 {
				require.Greater(t, peak.Load(), int32(1))
			}
			require.Equal(t, int32(100), iterator.CtrSuccess.Load())
		})
	}
}
