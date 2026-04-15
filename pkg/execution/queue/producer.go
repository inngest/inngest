package queue

import (
	"context"
	"time"

	"github.com/jonboulle/clockwork"
)

type ProducerOpts struct {
	clock              clockwork.Clock
	queueKindMapping   map[string]string
	enableJobPromotion bool
}

// QueueProducer implements the Producer interface, providing the ability to enqueue
// and requeue items.
type QueueProducer struct {
	clock              clockwork.Clock
	queueKindMapping   map[string]string
	enableJobPromotion bool

	queueShards   map[string]QueueShard
	shardSelector ShardSelector
}

func NewProducer(
	queueShards map[string]QueueShard,
	shardSelector ShardSelector,
	opts ProducerOpts,
) *QueueProducer {
	return &QueueProducer{
		clock:              opts.clock,
		queueKindMapping:   opts.queueKindMapping,
		enableJobPromotion: opts.enableJobPromotion,
		queueShards:        queueShards,
		shardSelector:      shardSelector,
	}
}

func (q *QueueProducer) Clock() clockwork.Clock {
	return q.clock
}

// RequeueByJobID implements Producer.
func (q *QueueProducer) RequeueByJobID(ctx context.Context, shard QueueShard, jobID string, at time.Time) error {
	return shard.RequeueByJobID(ctx, jobID, at)
}
