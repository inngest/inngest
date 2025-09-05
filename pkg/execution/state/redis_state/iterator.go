package redis_state

import (
	"context"
	"errors"
	"iter"
	"time"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/redis_state/peek"
	"github.com/inngest/inngest/pkg/logger"
)

type queueSortedSetIterationOptions struct {
	keySortedSet  string
	partitionID   string
	from          time.Time
	until         time.Time
	pageSize      int
	includeLeased bool
	peeker        peek.Peeker[osqueue.QueueItem]
}

// iterateSortedSetQueue returns an iterator over queue items found in a provided sorted set. This has to be millisecond precision, and can be the queue set, an in-progress
func (q *queue) iterateSortedSetQueue(ctx context.Context, shard QueueShard, opts queueSortedSetIterationOptions) iter.Seq[*osqueue.QueueItem] {
	l := q.log.With(
		"method", "iterateSortedSet",
		"queue_shard", shard.Name,
		"partition_id", opts.partitionID,
		"sorted_set_key", opts.keySortedSet,
		"from", opts.from.UnixMilli(),
		"from_human", opts.from.Format(time.StampMilli),
		"until", opts.until.UnixMilli(),
		"until_human", opts.until.Format(time.StampMilli),
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
			l := l.With("cursor", cursor.UnixMilli())
			l.Trace("fetching sorted set page")

			// Fetch page of queue items
			res, err := opts.peeker.Peek(ctx, opts.keySortedSet,
				peek.From(cursor),
				peek.Until(opts.until),
				peek.Limit(limit),
				peek.Sequential(true),
			)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					l.ReportError(err, "error peeking items for partition iterator")
				}
				return
			}

			items := res.Items

			if len(items) == 0 {
				return
			}

			// If the set has more items, we have just retrieved one extra item (given the limit is page size + 1)
			hasExtra := len(items) == limit

			itemsWithoutExtra := items
			if hasExtra {
				itemsWithoutExtra = items[0 : len(items)-1]
			}

			// Yield all items except for the extra item
			// If the item was previously returned, ignore it. This is a best-effort deduplication implementation.
			for _, qi := range itemsWithoutExtra {
				if _, has := seenBuffer[qi.ID]; has {
					continue
				}

				if !yield(qi) {
					return
				}
				seenBuffer[qi.ID] = struct{}{}
			}

			// If there's an extra item and it has the same score as the last item on the current page,
			// this means there is at least one more item with the same score. Temporarily switch to
			// offset-based pagination for all items with this score, then continue with the next millisecond.
			hasDuplicateScores := hasExtra && items[len(items)-1].AtMS == items[len(items)-2].AtMS
			if hasDuplicateScores {
				l.Trace("starting offset based iteration")
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

					if !yield(qi) {
						return
					}
					seenBuffer[qi.ID] = struct{}{}
				}
			}

			// Carry on with next page by advancing the cursor by one millisecond (this is equivalent to AtMS + 1)
			nextCursor := time.UnixMilli(itemsWithoutExtra[len(itemsWithoutExtra)-1].AtMS).Add(time.Millisecond)
			if nextCursor.Equal(cursor) {
				return
			}
			cursor = nextCursor
			l.Trace("continuing to next cursor", "cursor", cursor.UnixMilli())
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

type iteratePartitionBacklogsOpt struct {
	from        time.Time
	until       time.Time
	partitionID string
	interval    time.Duration
	pageSize    int
}

func (q *queue) iteratePartitionBacklogs(ctx context.Context, shard QueueShard, opt iteratePartitionBacklogsOpt) iter.Seq[*osqueue.QueueItem] {
	from := opt.from
	until := opt.until

	l := logger.StdlibLogger(ctx)

	// NOTE: iterate through backlogs
	backlogFrom := from

	sp, err := q.ShadowPartitionByID(ctx, shard, opt.partitionID)
	if err != nil && !errors.Is(err, ErrShadowPartitionNotFound) {
		l.Warn("error retrieving shadow partition from queue", "error", err)
	}

	return func(yield func(*osqueue.QueueItem) bool) {
		if sp == nil {
			return
		}

		l = l.With("shadow_partition", sp)

		for {
			var iterated int

			// TODO: maybe provide a different limit?
			backlogs, _, err := q.ShadowPartitionPeek(ctx, shard, sp, true, until, ShadowPartitionPeekMaxBacklogs)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					l.ReportError(err, "error peeking backlogs for partition")
				}
				return
			}

			if len(backlogs) == 0 {
				l.Warn("no more backlogs to iterate")
				return
			}

			latestTimes := []time.Time{}
			for _, backlog := range backlogs {
				errTags := map[string]string{
					"backlog_id": backlog.BacklogID,
				}

				var last time.Time
				items, _, err := q.backlogPeek(ctx, shard, backlog, backlogFrom, until, int64(opt.pageSize))
				if err != nil {
					l.ReportError(err, "error retrieving queue items from backlog",
						logger.WithErrorReportTags(errTags),
					)
					return
				}

				var start, end time.Time
				for _, qi := range items {
					if qi == nil {
						continue
					}

					if !yield(qi) {
						return
					}
					iterated++

					at := time.UnixMilli(qi.AtMS)
					if start.IsZero() {
						start = at
					}
					end = at
					last = at
				}

				l.Debug("iterated items in backlog",
					"count", iterated,
					"start", start.Format(time.StampMilli),
					"end", end.Format(time.StampMilli),
				)
				latestTimes = append(latestTimes, last)

				// didn't process anything, meaning there's nothing left to do
				// exit loop
				if iterated == 0 {
					return
				}
			}

			// find the earliest time within the last item timestamp of the previously processed backlogs
			var earliest time.Time
			for _, t := range latestTimes {
				if earliest.IsZero() || t.Before(earliest) {
					earliest = t
				}
			}
			// shift the starting point 1ms so it doesn't try to grab the same stuff again
			// NOTE: this could result skipping items if the previous batch of items are all on
			// the same millisecond
			backlogFrom = earliest.Add(time.Millisecond)

			// wait a little before proceeding
			<-time.After(opt.interval)
		}
	}
}
