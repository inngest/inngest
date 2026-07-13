package queue

import "context"

type queueConsumer struct {
	shards QueueShardRegistry
}

func newQueueConsumer(shards QueueShardRegistry) Consumer {
	return &queueConsumer{shards: shards}
}

// Dequeue implements Consumer.
func (c *queueConsumer) Dequeue(ctx context.Context, shardName string, i QueueItem, opts ...DequeueOptionFn) error {
	shard, err := c.shards.ByName(shardName)
	if err != nil {
		return err
	}

	return shard.Dequeue(ctx, i, opts...)
}
