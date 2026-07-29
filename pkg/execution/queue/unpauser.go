package queue

import "context"

type queueUnpauser struct {
	shards QueueShardRegistry
}

func newQueueUnpauser(shards QueueShardRegistry) Unpauser {
	return &queueUnpauser{shards: shards}
}

func (u *queueUnpauser) UnpauseFunction(ctx context.Context, shardName string, scope Scope) error {
	shard, err := u.shards.ByName(shardName)
	if err != nil {
		return err
	}
	return shard.UnpauseFunction(ctx, scope)
}
