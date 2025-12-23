package queue

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
)

type QueueShard interface {
	Name() string
	Kind() enums.QueueShardKind

	Processor() QueueProcessor
}

func (q *queueProcessor) selectShard(ctx context.Context, shardName string, qi QueueItem) (QueueShard, error) {
	l := logger.StdlibLogger(ctx)

	shard := q.PrimaryQueueShard
	switch {
	// If the caller wants us to enqueue the job to a specific queue shard, use that.
	case shardName != "":
		foundShard, ok := q.QueueShardClients[shardName]
		if !ok {
			return shard, fmt.Errorf("tried to force invalid queue shard %q", shardName)
		}

		shard = foundShard
	// Otherwise, invoke the shard selector, if configured.
	case q.shardSelector != nil:
		// QueueName should be consistently specified on both levels. This safeguard ensures
		// we'll check for both places, just in case.
		qn := qi.Data.QueueName
		if qn == nil {
			qn = qi.QueueName
		}

		selected, err := q.shardSelector(ctx, qi.Data.Identifier.AccountID, qn)
		if err != nil {
			l.Error("error selecting shard", "error", err, "item", qi)
			return shard, fmt.Errorf("could not select shard: %w", err)
		}

		shard = selected
	}
	return shard, nil
}
