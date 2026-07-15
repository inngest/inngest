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
// IDs. The scope is validated before dispatching to the shard so callers cannot
// accidentally requeue by ID without tenant/function context.
// Scope is not used for shard selection; shardName always selects the shard.
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
