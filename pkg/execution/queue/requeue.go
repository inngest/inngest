package queue

import (
	"context"
	"time"
)

func (q *queueProducer) Requeue(ctx context.Context, shardName string, i QueueItem, at time.Time, opts ...RequeueOptionFn) error {
	shard, err := q.shards.ByName(shardName)
	if err != nil {
		return err
	}

	return shard.Requeue(ctx, i, at, opts...)
}

// RequeueByJobID requires scope to include account, environment, and function
// IDs, preserving the producer interface contract used by wrappers. This
// producer does not use scope for lookup or shard selection: shardName selects
// the shard and jobID identifies the item within that shard. Other producer
// implementations, such as Cloud rollout wrappers, may use scope before
// delegating for account-level feature flag or routing decisions.
func (q *queueProducer) RequeueByJobID(ctx context.Context, scope Scope, shardName string, jobID string, at time.Time) error {
	if err := scope.ValidateIDs(); err != nil {
		return err
	}

	shard, err := q.shards.ByName(shardName)
	if err != nil {
		return err
	}

	return shard.RequeueByJobID(ctx, jobID, at)
}
