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

func (q *queueProducer) RequeueByJobID(ctx context.Context, scope Scope, shardName string, jobID string, at time.Time) error {
	if err := scope.Validate(); err != nil {
		return err
	}

	shard, err := q.shards.ByName(shardName)
	if err != nil {
		return err
	}

	return shard.RequeueByJobID(ctx, jobID, at)
}
