package queue

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
)

type ShardAssignmentConfig struct {
	ShardGroup   string
	NumExecutors int
}

type QueueShard interface {
	ShardOperations

	Name() string
	Kind() enums.QueueShardKind
	ShardAssignmentConfig() ShardAssignmentConfig
}

func (q *queueProcessor) selectShard(ctx context.Context, shardName string, qi QueueItem) (QueueShard, error) {
	l := logger.StdlibLogger(ctx)

	// If the caller wants us to enqueue the job to a specific queue shard, use that.
	if shardName != "" {
		shard, err := q.shards.ByName(shardName)
		if err != nil {
			return q.Shard(), fmt.Errorf("tried to force invalid queue shard %q", shardName)
		}
		return shard, nil
	}

	// QueueName should be consistently specified on both levels. This safeguard ensures
	// we'll check for both places, just in case.
	qn := qi.Data.QueueName
	if qn == nil {
		qn = qi.QueueName
	}

	selected, err := q.shards.Resolve(ctx, qi.Data.Identifier.AccountID, qn)
	if err != nil {
		l.Error("error selecting shard", "error", err, "item", qi)
		return q.Shard(), fmt.Errorf("could not select shard: %w", err)
	}
	return selected, nil
}
