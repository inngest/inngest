package queue

import (
	"context"
	"math"
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
		size, _ := q.peekSizeCache.Fetch(p.Queue(), 10*time.Second, func() (int64, error) {
			size := q.ewmaPeekSize(ctx, p)
			return size, nil
		})
		return size.Value()
	}
	return q.peekSizeWeightedDistribution(ctx, p, q.PeekSizeExponent)
}

// SkewedRand returns a random number in [min, max] following
// the distribution: min + (max-min) * r^n, where r ~ Uniform(0,1).
// Higher n values skew more heavily toward min.
func (q *queueProcessor) peekSizeWeightedDistribution(_ context.Context, _ *QueuePartition, n float64) int64 {
	r := rand.Float64()

	min := float64(q.PeekMin)
	if min == 0 {
		min = float64(DefaultQueuePeekMin)
	}
	max := float64(q.PeekMax)
	if max == 0 {
		max = float64(DefaultQueuePeekMax)
	}

	return int64(min + (max-min)*math.Pow(r, n))
}

func (q *queueProcessor) ewmaPeekSize(ctx context.Context, p *QueuePartition) int64 {
	if p.FunctionID == nil {
		return q.PeekMin
	}
	shard := q.Shard()

	// retrieve the EWMA value
	ewma, err := shard.PeekEWMA(ctx, *p.FunctionID)
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
	qsize, _ := shard.PartitionSize(ctx, p.ID, q.Clock().Now().Add(dur))
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
