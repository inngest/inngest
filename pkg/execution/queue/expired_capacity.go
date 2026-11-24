package queue

import (
	"context"
	"errors"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

// Bound Redis mutations so repairing a large ready queue does not monopolize
// the partition lease. Subsequent scans continue removing the expired prefix.
const expiredCapacityLeaseRequeueLimit = 100

type capacityLeaseRepairResult struct {
	attempts int
	retryAt  time.Time
	reason   string
}

func (q *queueProcessor) capacityLeaseRepairRetryAt(partitionID string) time.Time {
	if value, ok := q.repairRetry.Load(partitionID); ok {
		until := value.(time.Time)
		if until.After(q.Clock().Now()) {
			return until
		}
		q.repairRetry.CompareAndDelete(partitionID, value)
	}
	return time.Time{}
}

func (q *queueProcessor) requeueExpiredCapacityLeases(ctx context.Context, p *QueuePartition, items []*QueueItem) ([]*QueueItem, capacityLeaseRepairResult, error) {
	result := capacityLeaseRepairResult{}
	defer func() {
		if result.attempts > 0 {
			metrics.IncrQueueCapacityLeaseRepairCounter(ctx, int64(result.attempts), metrics.CounterOpt{PkgName: pkgName,
				Tags: map[string]any{"queue_shard": q.Shard().Name(), "outcome": "attempted"}})
		}
		if !result.retryAt.IsZero() {
			metrics.IncrQueueCapacityLeaseRepairCounter(ctx, 1, metrics.CounterOpt{PkgName: pkgName,
				Tags: map[string]any{"queue_shard": q.Shard().Name(), "outcome": "deferred", "reason": result.reason}})
		}
	}()
	if q.Shard().Kind() != enums.QueueShardKindRedis || q.CapacityManager == nil || p.LeaseID == nil {
		return items, result, nil
	}
	remaining := p.LeaseID.Timestamp().Sub(q.Clock().Now())
	if remaining <= 0 {
		return nil, result, ErrPartitionLeaseMismatch
	}
	budget := q.keyQueueCapacityLeaseRepairBudget
	if budget == nil {
		budget = defaultKeyQueueCapacityLeaseRepairBudget
	}
	// Even a late scan keeps most of its remaining lease for normal admission.
	duration := min(budget.limits.MaxDuration, remaining/4)
	deadline := q.Clock().Now().Add(duration)
	repairCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	kept := make([]*QueueItem, 0, len(items))
	var shardBudget *capacityLeaseRepairShardBudget
	var acquired bool
	defer func() {
		if acquired {
			shardBudget.sem.Release(1)
		}
	}()
	deferRepair := func(wait time.Duration, reason string) {
		// Stay beyond second-granularity pointer rounding and scanner lookahead.
		// A very low repair rate must not park healthy execution for its entire
		// token refill period; later scans can dispatch valid owners and recheck.
		wait = min(max(wait, PartitionLookahead+2*time.Second), 5*time.Second)
		result.retryAt = q.Clock().Now().Add(wait + time.Duration(rnd.Uint64n(uint64(time.Second))))
		result.reason = reason
	}
	for _, item := range items {
		now := q.Clock().Now()
		// Legacy throttle payloads need the ordinary admission path to refresh
		// their routing before requeueing. Never refresh/delete from a stale
		// cleanup snapshot outside the conditional Redis mutation.
		legacyThrottle := item != nil && item.Data.Throttle != nil && item.Data.Throttle.KeyExpressionHash == ""
		if item == nil || legacyThrottle || item.RefilledFrom == "" || item.CapacityLease == nil ||
			!item.CapacityLease.LeaseID.Timestamp().Before(now) ||
			item.IsLeased(now) || item.AtMS > now.UnixMilli() ||
			!q.ItemAllowKeyQueueCapacityLeaseRepair(ctx, *item) {
			kept = append(kept, item)
			continue
		}
		if result.retryAt.IsZero() {
			result.retryAt = q.capacityLeaseRepairRetryAt(p.ID)
			if !result.retryAt.IsZero() {
				result.reason = "cooldown"
			}
		}
		if !result.retryAt.IsZero() {
			continue
		}
		if result.attempts >= expiredCapacityLeaseRequeueLimit {
			deferRepair(0, "pass_limit")
			continue
		}
		if !now.Before(deadline) || repairCtx.Err() != nil {
			deferRepair(0, "time")
			continue
		}
		if shardBudget == nil {
			shardBudget = budget.shard(q.Shard().Name())
			acquired = shardBudget.sem.TryAcquire(1)
			if !acquired {
				deferRepair(0, "concurrency")
				continue
			}
		}
		if !shardBudget.limiter.AllowN(now, 1) {
			deferRepair(shardBudget.retryAfter(now), "rate")
			continue
		}
		result.attempts++
		if err := q.Shard().Requeue(repairCtx, *item, now, RequeueIfExpiredCapacityLease(*p.LeaseID)); err != nil {
			if errors.Is(err, context.DeadlineExceeded) && repairCtx.Err() != nil && ctx.Err() == nil {
				// The mutation may have committed before the local timeout. Skip its
				// snapshot, and keep the parent context available for healthy work.
				deferRepair(0, "time")
				continue
			}
			return nil, result, err
		}
		// Always skip expired snapshots, including those deferred or rejected by
		// Redis. Acquiring capacity for them here would recreate the starvation.
	}
	if err := ctx.Err(); err != nil {
		return nil, result, err
	}
	if !p.LeaseID.Timestamp().After(q.Clock().Now()) {
		return nil, result, ErrPartitionLeaseMismatch
	}
	return kept, result, nil
}
