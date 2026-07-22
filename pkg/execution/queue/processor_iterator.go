package queue

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"golang.org/x/sync/errgroup"
)

type ProcessorIterator struct {
	Partition *QueuePartition
	Items     []*QueueItem
	// PartitionContinueCtr is the number of times the partition has currently been
	// continued already in the chain.  we must record this such that a partition isn't
	// forced indefinitely.
	PartitionContinueCtr uint

	// Queue is the Queue that owns this processor.
	Queue QueueProcessor
	// Leaser attempts to lease queue items before dispatch.
	Leaser QueueItemLeaser
	// Dispatch sends a leased item to the execution layer.
	Dispatch DispatchFunc

	// error returned when processing
	Err error

	// StaticTime is used as the processing time for all items in the queue.
	// We process queue items sequentially, and time progresses linearly as each
	// queue item is processed.  We want to use a static time to prevent out-of-order
	// processing with regards to things like rate limiting;  if we use time.Now(),
	// queue items later in the array may be processed before queue items earlier in
	// the array depending on eg. a rate limit becoming available half way through
	// iteration.
	StaticTime time.Time

	// Parallel indicates whether the partition's jobs can be processed in Parallel.
	// Parallel processing breaks best effort fifo but increases throughput.
	Parallel bool

	// These flags are used to handle partition requeueing.
	// These counters must be accessed atomically as they may be incremented
	// concurrently when Parallel=true.
	CtrSuccess     atomic.Int32
	CtrConcurrency atomic.Int32
	CtrRateLimit   atomic.Int32

	// IsCustomKeyLimitOnly records whether we ONLY hit custom concurrency key limits.
	// This lets us know whether to peek from a random offset if we have FIFO disabled
	// to attempt to find other possible functions outside of the key(s) with issues.
	// This field must be accessed atomically as it may be modified concurrently when Parallel=true.
	IsCustomKeyLimitOnly atomic.Bool

	// IsSemaphoreLimitOnly records whether all concurrency hits were from semaphore limits only.
	// When true, we use a shorter partition requeue delay since semaphore-blocked items stay in
	// the ready queue and can be picked up quickly once capacity is freed.
	IsSemaphoreLimitOnly atomic.Bool
}

func (p *ProcessorIterator) Iterate(ctx context.Context) error {
	var err error

	// set flags to true to begin with
	p.IsCustomKeyLimitOnly.Store(true)
	p.IsSemaphoreLimitOnly.Store(true)

	eg := errgroup.Group{}
	for idx, i := range p.Items {
		if i == nil {
			// THIS SHOULD NEVER HAPPEN. Skip gracefully and log error
			logger.StdlibLogger(ctx).Error("nil queue item in partition", "partition", p.Partition)
			continue
		}

		if p.Parallel {
			item := *i
			eg.Go(func() error {
				err := p.LeaseItem(ctx, &item)
				if err != nil {
					// NOTE: ignore if the queue item is not found
					if errors.Is(err, ErrQueueItemNotFound) {
						return nil
					}
				}
				return err
			})
			continue
		}

		// non-parallel (sequential fifo) processing.
		if err = p.LeaseItem(ctx, i); err != nil {
			// NOTE: ignore if the queue item is not found
			if errors.Is(err, ErrQueueItemNotFound) {
				continue
			}

			// If item processing was terminated early due to user constraints,
			// set the earliest peek time for the remaining items.
			// This ensures accurate tracking of peeked items waiting for
			// user constraint capacity to become available (user latency).
			if errors.Is(err, ErrProcessNoUserConstraintCapacity) {
				config := p.Queue.Options().ItemEarliestPeekTimeConfig(ctx, p.Queue.Shard().Name(), *i)
				if config.Enabled {
					// Stamp remaining items from next item on (idx+1)
					p.stampRemainingEarliestPeekTimes(ctx, idx+1, config.BulkStampLimit)
				}
			}
			// always break on the first error;  if processing returns an error we
			// always assume that we stop iterating.
			//
			// we return errors when:
			// * there's no capacity (so dont continue, because FIFO)
			// * we hit fn concurrency limits (so don't continue, because FIFO too)
			// * some other error, which means something went wrong.
			break
		}
	}

	if p.Parallel {
		// normalize errors from parallel
		err = eg.Wait()
	}

	if errors.Is(err, ErrProcessStopIterator) {
		// This is safe;  it's stopping safely but isn't an error.
		return nil
	}
	if errors.Is(err, ErrProcessNoCapacity) {
		// This is safe;  it's stopping safely but isn't an error.
		return nil
	}

	// someting went wrong.  report the error.
	return err
}

// stampRemainingEarliestPeekTimes iterates over a slice of items starting from start and up to limit.
func (p *ProcessorIterator) stampRemainingEarliestPeekTimes(ctx context.Context, start, limit int) {
	if limit <= 0 || start >= len(p.Items) {
		return
	}

	peekTime := p.StaticTime
	if peekTime.IsZero() {
		peekTime = p.Queue.Clock().Now()
	}

	l := logger.StdlibLogger(ctx).With("partition", p.Partition)
	attempts := 0
	for _, item := range p.Items[start:] {
		if attempts >= limit {
			return
		}
		if item == nil || item.EarliestPeekTime != 0 {
			continue
		}

		attempts++
		earliestPeekTime, err := p.Queue.Shard().SetEarliestPeekTime(ctx, *item, peekTime)
		if err != nil {
			l.Warn("could not set earliest peek time for remaining item", "item", item, "error", err)
			continue
		}

		item.EarliestPeekTime = earliestPeekTime.UnixMilli()
	}
}

// Process leases a single queue item and dispatches it to the execution layer.
//
// Deprecated: use LeaseItem.
func (p *ProcessorIterator) Process(ctx context.Context, item *QueueItem) error {
	return p.LeaseItem(ctx, item)
}

func (p *ProcessorIterator) LeaseItem(ctx context.Context, item *QueueItem) error {
	leaser := p.Leaser
	if leaser == nil {
		var ok bool
		leaser, ok = p.Queue.(QueueItemLeaser)
		if !ok {
			return ErrProcessStopIterator
		}
	}

	result, err := leaser.LeaseItem(ctx, LeaseItemRequest{
		Item:                 item,
		Partition:            p.Partition,
		PartitionContinueCtr: p.PartitionContinueCtr,
		StaticTime:           p.StaticTime,
	}, p.Dispatch)
	p.applyLeaseItemResult(result)
	return err
}

func (p *ProcessorIterator) applyLeaseItemResult(result LeaseItemResult) {
	switch result.Status {
	case LeaseItemStatusDispatched, LeaseItemStatusNotFound, LeaseItemStatusLeaseContention:
		p.CtrSuccess.Add(1)
	case LeaseItemStatusThrottled:
		p.IsCustomKeyLimitOnly.Store(false)
		p.IsSemaphoreLimitOnly.Store(false)
		p.CtrRateLimit.Add(1)
	case LeaseItemStatusConcurrencyLimited:
		p.IsCustomKeyLimitOnly.Store(false)
		p.IsSemaphoreLimitOnly.Store(false)
		p.CtrConcurrency.Add(1)
	case LeaseItemStatusCustomConcurrencyLimited:
		p.IsSemaphoreLimitOnly.Store(false)
		p.CtrConcurrency.Add(1)
	case LeaseItemStatusSemaphoreLimited:
		p.CtrConcurrency.Add(1)
	case LeaseItemStatusLeaseError:
		p.Err = result.Err
	}
}

func (p *ProcessorIterator) IsRequeuable() bool {
	// if we have concurrency OR we hit rate limiting/throttling.
	ctrConcurrency := p.CtrConcurrency.Load()
	ctrRateLimit := p.CtrRateLimit.Load()
	ctrSuccess := p.CtrSuccess.Load()
	return ctrConcurrency > 0 || (ctrRateLimit > 0 && ctrConcurrency == 0 && ctrSuccess == 0)
}
