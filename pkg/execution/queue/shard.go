package queue

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
)

type QueueShard interface {
	Name() string
	Kind() enums.QueueShardKind

	Processor() QueueProcessor
}

func (q *queueProcessor) selectShard(ctx context.Context, shardName string, qi QueueItem) (QueueShard, error) {
	shard := q.options.PrimaryQueueShard
	switch {
	// If the caller wants us to enqueue the job to a specific queue shard, use that.
	case shardName != "":
		foundShard, ok := q.options.queueShardClients[shardName]
		if !ok {
			return shard, fmt.Errorf("tried to force invalid queue shard %q", shardName)
		}

		shard = foundShard
	// Otherwise, invoke the shard selector, if configured.
	case q.options.shardSelector != nil:
		// QueueName should be consistently specified on both levels. This safeguard ensures
		// we'll check for both places, just in case.
		qn := qi.Data.QueueName
		if qn == nil {
			qn = qi.QueueName
		}

		selected, err := q.options.shardSelector(ctx, qi.Data.Identifier.AccountID, qn)
		if err != nil {
			q.options.log.Error("error selecting shard", "error", err, "item", qi)
			return shard, fmt.Errorf("could not select shard: %w", err)
		}

		shard = selected
	}
	return shard, nil
}
