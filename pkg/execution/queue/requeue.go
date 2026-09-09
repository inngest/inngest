package queue

import (
	"context"
	"errors"
	"time"
)

func (q *queueProducer) Requeue(ctx context.Context, shardName string, i QueueItem, at time.Time, opts ...RequeueOptionFn) error {
	// Account routing may have changed while this item was leased. Prefer the
	// current shard so a migrated copy is requeued at its destination. During
	// the copy window the destination may not contain the item yet; falling back
	// to the named source preserves it for the migration reader.
	current, resolveErr := q.selectShard(ctx, "", i)
	if resolveErr == nil {
		err := current.Requeue(ctx, i, at, opts...)
		if err == nil || current.Name() == shardName || !errors.Is(err, ErrQueueItemNotFound) {
			return err
		}
	}

	source, err := q.shards.ByName(shardName)
	if err != nil {
		if resolveErr != nil {
			return resolveErr
		}
		return err
	}
	return source.Requeue(ctx, i, at, opts...)
}

// RequeueByJobID requires scope to include account, environment, and function
// IDs. It prefers the account's current shard and falls back to shardName while
// a migration has not copied the item yet.
func (q *queueProducer) RequeueByJobID(ctx context.Context, scope Scope, shardName string, jobID string, at time.Time) error {
	if err := scope.ValidateIDs(); err != nil {
		return err
	}

	current, resolveErr := q.shards.Resolve(ctx, scope, nil)
	if resolveErr == nil {
		err := current.RequeueByJobID(ctx, jobID, at)
		if err == nil || current.Name() == shardName || !errors.Is(err, ErrQueueItemNotFound) {
			return err
		}
	}

	source, err := q.shards.ByName(shardName)
	if err != nil {
		if resolveErr != nil {
			return resolveErr
		}
		return err
	}
	return source.RequeueByJobID(ctx, jobID, at)
}
