package queue

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
)

// ItemHintSource serves notifications for the owned shard until ctx is canceled.
// offer is nonblocking and acknowledges buffer admission, not execution.
type ItemHintSource func(ctx context.Context, shard QueueShard, offer func(itemID string) bool) error

type ItemHintOptions struct {
	Source         ItemHintSource
	BufferSize     int
	MaxActive      int
	AttemptTimeout time.Duration
}

func WithItemHints(opts ItemHintOptions) QueueOpt {
	return func(o *QueueOptions) { o.itemHints = &opts }
}

// ItemHintShard opts a backend into direct processing with its own eligibility
// checks. LoadReadyItem must preserve ready/backlog, pause, and migration rules.
type ItemHintShard interface {
	LoadReadyItem(context.Context, string) (*QueueItem, error)
}

func (q *queueProcessor) hintsAllowed() bool {
	if q.hintsStopped.Load() || (!q.runMode.Partition && !q.runMode.Account) || q.scanningExcludedByRole() != "" {
		return false
	}
	if q.runMode.ShardGroup != "" {
		lease := q.shardLease()
		return lease != nil && ulid.Time(lease.Time()).After(q.Clock().Now())
	}
	return true
}

func (q *queueProcessor) startItemHints(ctx context.Context, dispatch DispatchFunc) func() {
	opts := q.itemHints
	shard, ok := q.Shard().(ItemHintShard)
	if opts == nil || opts.Source == nil || !ok || !q.hintsAllowed() {
		return func() {}
	}
	if opts.BufferSize <= 0 || opts.MaxActive <= 0 || opts.AttemptTimeout <= 0 {
		logger.StdlibLogger(ctx).Warn("item hints disabled: invalid limits")
		return func() {}
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	pending := make(chan string, opts.BufferSize)
	active := make(chan struct{}, opts.MaxActive)
	offer := func(id string) bool {
		if id == "" || ctx.Err() != nil || !q.hintsAllowed() {
			return false
		}
		select {
		case pending <- id:
			return true
		default:
			return false
		}
	}
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		defer wg.Wait()
		defer cancel()
		wg.Go(func() {
			defer cancel()
			if err := opts.Source(ctx, q.Shard(), offer); err != nil && ctx.Err() == nil {
				logger.StdlibLogger(ctx).Warn("item hint source stopped; scanning continues", "error", err)
			}
		})
		tick := q.Clock().NewTicker(q.pollTick)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.Chan():
				if !q.hintsAllowed() {
					return
				}
				// Snapshot the pending count so arrivals cannot extend this pass.
				for range len(pending) {
					id := <-pending
					if !q.hintsAllowed() {
						q.recordItemHint(ctx, "inactive")
						continue
					}
					select {
					case active <- struct{}{}:
						wg.Go(func() {
							defer func() { <-active }()
							q.processItemHint(ctx, shard, id, dispatch)
						})
					default:
						q.recordItemHint(ctx, "budget_full")
					}
				}
				metrics.RecordGaugeMetric(ctx, int64(len(active)), metrics.GaugeOpt{
					PkgName: pkgName, MetricName: "queue_item_hint_active", Tags: map[string]any{"queue_shard": q.Shard().Name()},
				})
				metrics.RecordGaugeMetric(ctx, int64(len(pending)), metrics.GaugeOpt{
					PkgName: pkgName, MetricName: "queue_item_hint_pending", Tags: map[string]any{"queue_shard": q.Shard().Name()},
				})
			}
		}
	}()
	return func() { cancel(); <-done }
}

func (q *queueProcessor) processItemHint(ctx context.Context, shard ItemHintShard, id string, dispatch DispatchFunc) {
	attemptCtx, cancel := context.WithTimeout(ctx, q.itemHints.AttemptTimeout)
	defer cancel()
	item, err := shard.LoadReadyItem(attemptCtx, id)
	if err != nil {
		status := "read_error"
		if errors.Is(err, ErrQueueItemNotFound) || errors.Is(err, ErrQueueItemNotReady) {
			status = "ineligible"
		}
		q.recordItemHint(ctx, status)
		return
	}
	if item.Data.Kind != KindStart || item.Data.Attempt != 0 || item.AtMS > q.Clock().Now().UnixMilli() || !q.hintsAllowed() {
		q.recordItemHint(ctx, "ineligible")
		return
	}
	account := item.Data.Identifier.AccountID
	if len(q.runMode.ExclusiveAccounts) > 0 && !slices.Contains(q.runMode.ExclusiveAccounts, account) {
		q.recordItemHint(ctx, "ineligible")
		return
	}
	if exists, err := q.accountExists(attemptCtx, account); err != nil || !exists {
		q.recordItemHint(ctx, "ineligible")
		return
	}
	partition := ItemPartition(ctx, *item)
	var dispatched DispatchedItem
	result, err := q.LeaseItem(attemptCtx, LeaseItemRequest{
		Item: item, RequireReady: true, StaticTime: q.Clock().Now(),
		Priority: q.PartitionPriorityFinder(attemptCtx, partition),
	}, func(ctx context.Context, item ProcessItem) (DispatchedItem, error) {
		var err error
		dispatched, err = dispatch(ctx, item)
		return dispatched, err
	})
	cancel()
	if err != nil || dispatched == nil || result.Status != LeaseItemStatusDispatched {
		q.recordItemHint(ctx, "not_dispatched")
		return
	}
	q.recordItemHint(ctx, "dispatched")
	select {
	case result := <-dispatched.Done():
		if q.runMode.Continuations && result.Err == nil && result.ScheduledImmediateJob {
			q.addContinue(ctx, &partition, 1)
		}
	case <-ctx.Done():
	}
}

func (q *queueProcessor) recordItemHint(ctx context.Context, outcome string) {
	metrics.RecordCounterMetric(ctx, 1, metrics.CounterOpt{
		PkgName: pkgName, MetricName: "queue_item_hint_total",
		Tags: map[string]any{"queue_shard": q.Shard().Name(), "outcome": outcome},
	})
}
