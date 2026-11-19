package queue

import (
	"context"
	"github.com/inngest/inngest/pkg/enums"
)

// Bound Redis mutations so repairing a large ready queue does not monopolize
// the partition lease. Subsequent scans continue removing the expired prefix.
const expiredCapacityLeaseRequeueLimit = 100

func (q *queueProcessor) requeueExpiredCapacityLeases(ctx context.Context, p *QueuePartition, items []*QueueItem) ([]*QueueItem, bool, error) {
	if q.Shard().Kind() != enums.QueueShardKindRedis || q.CapacityManager == nil || p.LeaseID == nil {
		return items, false, nil
	}
	remaining := p.LeaseID.Timestamp().Sub(q.Clock().Now())
	if remaining <= 0 {
		return nil, false, ErrPartitionLeaseMismatch
	}
	ctx, cancel := context.WithTimeout(ctx, remaining)
	defer cancel()

	kept := make([]*QueueItem, 0, len(items))
	attempts := 0
	for _, item := range items {
		now := q.Clock().Now()
		// Legacy throttle payloads need the ordinary admission path to refresh
		// their routing before requeueing. Never refresh/delete from a stale
		// cleanup snapshot outside the conditional Redis mutation.
		legacyThrottle := item != nil && item.Data.Throttle != nil && item.Data.Throttle.KeyExpressionHash == ""
		if item == nil || legacyThrottle || !q.ItemEnableKeyQueues(ctx, *item) ||
			item.RefilledFrom == "" || item.CapacityLease == nil ||
			!item.CapacityLease.LeaseID.Timestamp().Before(now) ||
			item.IsLeased(now) || item.AtMS > now.UnixMilli() {
			kept = append(kept, item)
			continue
		}
		if attempts < expiredCapacityLeaseRequeueLimit {
			if err := q.Shard().Requeue(ctx, *item, now, RequeueIfExpiredCapacityLease(*p.LeaseID)); err != nil {
				return nil, false, err
			}
			attempts++
		}
		// Skip rejected/stale snapshots as well as expired items beyond this
		// pass's repair budget. Acquiring capacity for them recreates the cycle.
	}
	return kept, attempts > 0, nil
}
