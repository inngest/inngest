package queue

import (
	"context"
	"math/rand"
	"time"
)

// peekSize returns the number of items to peek for the queue based on a couple of factors
// 1. EWMA of concurrency limit hits
// 2. configured min, max of peek size range
// 3. worker capacity
func (q *queueProcessor) peekSize(ctx context.Context, p *QueuePartition) int64 {
	if peekSize, ok := q.peekSizeForFunctions[p.ID]; ok {
		return peekSize
	}
	if q.usePeekEWMA {
		return q.ewmaPeekSize(ctx, p)
	}
	return q.peekSizeRandom(ctx, p)
}

func (q *queueProcessor) peekSizeRandom(_ context.Context, _ *QueuePartition) int64 {
	// set ranges
	pmin := q.PeekMin
	if pmin == 0 {
		pmin = q.PeekMin
	}
	pmax := q.PeekMax
	if pmax == 0 {
		pmax = q.PeekMax
	}

	// Take a random amount between our range.
	size := int64(rand.Intn(int(pmax-pmin))) + pmin
	// Limit to capacity
	cap := q.capacity()
	if size > cap {
		size = cap
	}
	return size
}

//nolint:unused // this code remains to be enabled on demand
func (q *queueProcessor) ewmaPeekSize(ctx context.Context, p *QueuePartition) int64 {
	if p.FunctionID == nil {
		return q.PeekMin
	}

	// retrieve the EWMA value
	ewma, err := q.primaryQueueShard.PeekEWMA(ctx, *p.FunctionID)
	if err != nil {
		// return the minimum if there's an error
		return q.PeekMin
	}

	// set multiplier
	multiplier := q.peekCurrMultiplier
	if multiplier == 0 {
		multiplier = QueuePeekCurrMultiplier
	}

	// set ranges
	pmin := q.PeekMin
	if pmin == 0 {
		pmin = DefaultQueuePeekMin
	}
	pmax := q.PeekMax
	if pmax == 0 {
		pmax = DefaultQueuePeekMax
	}

	// calculate size with EWMA and multiplier
	size := ewma * multiplier
	switch {
	case size < pmin:
		size = pmin
	case size > pmax:
		size = pmax
	}

	dur := time.Hour * 24
	qsize, _ := q.primaryQueueShard.PartitionSize(ctx, p.ID, q.Clock().Now().Add(dur))
	if qsize > size {
		size = qsize
	}

	// add 10% expecting for some workflow that will finish in the mean time
	cap := int64(float64(q.capacity()) * 1.1)
	if size > cap {
		size = cap
	}

	return size
}
