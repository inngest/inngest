package queue

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
)

// ItemHintSource serves notifications for the owned shard until ctx is canceled.
// offer is nonblocking and acknowledges buffer admission, not execution.
type ItemHintSource func(ctx context.Context, shard QueueShard, offer func(QueueItem) bool) error

type ItemHintOptions struct {
	Source     ItemHintSource
	BufferSize int
	// MaxActive bounds concurrent eligibility/lease attempts, not running work.
	MaxActive      int
	AttemptTimeout time.Duration
}

func WithItemHints(opts ItemHintOptions) QueueOpt {
	return func(o *QueueOptions) { o.itemHints = &opts }
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
	if opts == nil || opts.Source == nil || !q.hintsAllowed() {
		return func() {}
	}
	if opts.BufferSize <= 0 || opts.MaxActive <= 0 || opts.AttemptTimeout <= 0 {
		logger.StdlibLogger(ctx).Warn("item hints disabled: invalid limits")
		return func() {}
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	pending := make(chan QueueItem, opts.BufferSize)
	active := make(chan struct{}, opts.MaxActive)
	offer := func(item QueueItem) bool {
		if item.ID == "" || ctx.Err() != nil || !q.hintsAllowed() {
			return false
		}
		select {
		case pending <- item:
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
					item := <-pending
					if !q.hintsAllowed() {
						q.recordItemHint(ctx, "inactive")
						continue
					}
					select {
					case active <- struct{}{}:
						wg.Go(func() {
							dispatched := q.processItemHint(ctx, item, dispatch)
							<-active
							// Normal worker capacity bounds execution. Completion
							// observation must not retain a hint attempt slot.
							if dispatched != nil && q.runMode.Continuations {
								q.observeItemHint(ctx, ItemPartition(ctx, item), dispatched)
							}
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

func (q *queueProcessor) processItemHint(ctx context.Context, item QueueItem, dispatch DispatchFunc) DispatchedItem {
	if ctx.Err() != nil {
		return nil
	}
	// Match the existing two-second lookahead. Only admission uses this future
	// time; leases use actual now and the worker waits until the item's AtMS.
	now := q.Clock().Now()
	if item.Data.Kind != KindStart || item.Data.Attempt != 0 || item.AtMS > now.Add(2*time.Second).UnixMilli() ||
		item.QueueName != nil || item.Data.QueueName != nil || !q.hintsAllowed() {
		q.recordItemHint(ctx, "ineligible")
		return nil
	}
	account := item.Data.Identifier.AccountID
	partition := ItemPartition(ctx, item)
	matches := func(name string) bool {
		return name == partition.Queue() || (strings.HasSuffix(name, "*") && strings.HasPrefix(partition.Queue(), strings.TrimSuffix(name, "*")))
	}
	if (len(q.AllowQueues) > 0 && !slices.ContainsFunc(q.AllowQueues, matches)) || slices.ContainsFunc(q.DenyQueues, matches) ||
		(len(q.runMode.ExclusiveAccounts) > 0 && !slices.Contains(q.runMode.ExclusiveAccounts, account)) {
		q.recordItemHint(ctx, "ineligible")
		return nil
	}
	attemptCtx, cancel := context.WithTimeout(ctx, q.itemHints.AttemptTimeout)
	defer cancel()
	// Direct hints bypass scanner eligibility, not pause/migration policy. Use
	// the same callbacks/operation without scanner pointer requeues or refills.
	if q.PartitionPausedGetter(attemptCtx, item.FunctionID).Paused || attemptCtx.Err() != nil {
		q.recordItemHint(ctx, "ineligible")
		return nil
	}
	locked, err := q.Shard().IsMigrationLocked(attemptCtx, Scope{AccountID: account, EnvID: item.WorkspaceID, FunctionID: item.FunctionID})
	if err != nil {
		q.recordItemHint(ctx, "read_error")
		return nil
	}
	if locked != nil && locked.After(q.Clock().Now()) {
		q.recordItemHint(ctx, "ineligible")
		return nil
	}
	if exists, err := q.accountExists(attemptCtx, account); err != nil || !exists {
		q.recordItemHint(ctx, "ineligible")
		return nil
	}
	// Scanners attach the stored queue ID to the worker-facing Item. Enqueue's
	// returned envelope may still carry the original (unhashed) producer JobID.
	item.Data.JobID = &item.ID
	var dispatched DispatchedItem
	result, err := q.LeaseItem(attemptCtx, LeaseItemRequest{
		Item: &item, StaticTime: q.Clock().Now(),
		Priority: q.PartitionPriorityFinder(attemptCtx, partition),
	}, func(ctx context.Context, item ProcessItem) (DispatchedItem, error) {
		var err error
		dispatched, err = dispatch(ctx, item)
		return dispatched, err
	})
	cancel()
	if err != nil || dispatched == nil || result.Status != LeaseItemStatusDispatched {
		q.recordItemHint(ctx, "not_dispatched")
		return nil
	}
	q.recordItemHint(ctx, "dispatched")
	return dispatched
}

func (q *queueProcessor) observeItemHint(ctx context.Context, partition QueuePartition, dispatched DispatchedItem) {
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
