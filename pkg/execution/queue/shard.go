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

func (q *queueProducer) defaultQueueNameForItemKind(kind string) *string {
	var queueName *string
	if name, ok := q.queueKindMapping[kind]; ok {
		queueName = &name
	}
	return queueName
}

func (q *queueProducer) selectShard(ctx context.Context, shardName string, qi QueueItem) (QueueShard, error) {
	l := logger.StdlibLogger(ctx)

	// If the caller wants us to enqueue the job to a specific queue shard, use that.
	if shardName != "" {
		shard, err := q.shards.ByName(shardName)
		if err != nil {
			return nil, fmt.Errorf("tried to force invalid queue shard %q", shardName)
		}
		return shard, nil
	}

	var queueItemKind *string
	if q.defaultQueueNameForItemKind(qi.Data.Kind) != nil {
		queueItemKind = &qi.Data.Kind
	}

	selected, err := q.shards.Resolve(ctx, ScopeFromQueueItem(qi), queueItemKind)
	if err != nil {
		l.Error("error selecting shard", "error", err, "item", qi)
		return nil, fmt.Errorf("could not select shard: %w", err)
	}
	return selected, nil
}
