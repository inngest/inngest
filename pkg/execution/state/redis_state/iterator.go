package redis_state

import (
	"context"
	"errors"
	"iter"
	"time"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
)

type queueSortedSetIterationOptions struct {
	keySortedSet  string
	partitionID   string
	from          time.Time
	until         time.Time
	pageSize      int
	includeLeased bool
	peek          func(ctx context.Context, shard QueueShard, from, until time.Time, limit, offset int) ([]*osqueue.QueueItem, error)
}

// iterateSortedSetQueue returns an iterator over queue items found in a provided sorted set. This has to be millisecond precision, and can be the queue set, an in-progress
func (q *queue) iterateSortedSetQueue(ctx context.Context, shard QueueShard, opts queueSortedSetIterationOptions) iter.Seq[*osqueue.QueueItem] {
	l := q.log.With(
		"method", "iterateSortedSet",
		"queue_shard", shard.Name,
		"partition_id", opts.partitionID,
		"sorted_set_key", opts.keySortedSet,
		"from", opts.from.UnixMilli(),
		"until", opts.until.UnixMilli(),
		"page_size", opts.pageSize,
	)

	// TODO: Switch to a circular buffer data structure
	seenBuffer := map[string]struct{}{}

	// Always look up one more item than the requested page size to check for duplicate scores.
	// A duplicate score is assumed when the extra item has the same score as the last item of the current page.
	//
	// This is relevant to ensure we never miss queue items while iterating. If we simply skip to the next millisecond,
	// we could miss items on the previous milliseconds outside the current page.
	limit := opts.pageSize + 1

	return func(yield func(*osqueue.QueueItem) bool) {
		// Cursor tracks the current position within the sorted set. This advances with each page, based on the last item's score.
		// Queue sets are assumed to use timestamps as scores with millisecond precision. This means the next item may be as close as the next millisecond.
		//
		// The important edge case of multiple items with the same score is explained above.
		cursor := opts.from
		for {
			// Fetch page of queue items
			items, err := opts.peek(ctx, shard, &cursor, opts.until, int64(limit), 0,
				peekOpts{
					PartitionID:   opts.partitionID,
					PartitionKey:  opts.keySortedSet,
					IncludeLeased: opts.includeLeased,
					From:          &cursor,
					Until:         opts.until,
					Limit:         int64(limit),
					Offset:        0,
				})
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					l.ReportError(err, "error peeking items for partition iterator")
				}
				return
			}

			if len(items) == 0 {
				break
			}

			// If the set has more items, we have just retrieved one extra item (given the limit is page size + 1)
			hasExtra := len(items) == limit

			itemsWithoutExtra := items
			if hasExtra {
				itemsWithoutExtra = items[0 : len(items)-2]
			}

			// Yield all items except for the extra item
			// If the item was previously returned, ignore it. This is a best-effort deduplication implementation.
			for _, qi := range itemsWithoutExtra {
				if _, has := seenBuffer[qi.ID]; has {
					continue
				}

				yield(qi)
				seenBuffer[qi.ID] = struct{}{}
			}

			// If there's an extra item and it has the same score as the last item on the current page,
			// this means there is at least one more item with the same score. Temporarily switch to
			// offset-based pagination for all items with this score, then continue with the next millisecond.
			hasDuplicateScores := hasExtra && items[len(items)-1].AtMS == items[len(items)-2].AtMS
			if hasDuplicateScores {
				items := q.offsetBasedIteration(ctx, shard, offsetBasedIterationOptions{
					keySortedSet:  opts.keySortedSet,
					partitionID:   opts.partitionID,
					score:         time.UnixMilli(items[len(items)-1].AtMS),
					pageSize:      opts.pageSize,
					includeLeased: opts.includeLeased,
				})
				for qi := range items {
					if _, has := seenBuffer[qi.ID]; has {
						continue
					}

					yield(qi)
					seenBuffer[qi.ID] = struct{}{}
				}
			}

			// Carry on with next page by advancing the cursor by one millisecond (this is equivalent to AtMS + 1)
			cursor = time.UnixMilli(itemsWithoutExtra[len(items)-1].AtMS).Add(time.Millisecond)
		}
	}
}

type offsetBasedIterationOptions struct {
	keySortedSet  string
	partitionID   string
	score         time.Time
	pageSize      int
	includeLeased bool
}

// offsetBasedIteration performs offset-based pagination over all queue items with the same score.
func (q *queue) offsetBasedIteration(ctx context.Context, shard QueueShard, opts offsetBasedIterationOptions) iter.Seq[*osqueue.QueueItem] {
	l := q.log.With(
		"method", "offsetBasedIteration",
		"queue_shard", shard.Name,
		"partition_id", opts.partitionID,
		"sorted_set_key", opts.keySortedSet,
		"score", opts.score.UnixMilli(),
		"page_size", opts.pageSize,
	)

	return func(yield func(*osqueue.QueueItem) bool) {
		var offset int64
		for {
			// Fetch page of queue items
			items, err := q.peek(ctx, shard, peekOpts{
				PartitionID:   opts.partitionID,
				PartitionKey:  opts.keySortedSet,
				IncludeLeased: opts.includeLeased,
				From:          &opts.score,
				Until:         opts.score,
				Limit:         int64(opts.pageSize),
				Offset:        offset,
			})
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					l.ReportError(err, "error peeking items for partition iterator")
				}
				return
			}

			if len(items) == 0 {
				break
			}

			for _, qi := range items {
				yield(qi)
			}

			offset += int64(len(items))
		}
	}
}
