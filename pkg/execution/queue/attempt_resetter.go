package queue

import "context"

type attemptResetter struct {
	shards QueueShardRegistry
}

func newAttemptResetter(shards QueueShardRegistry) AttemptResetter {
	return &attemptResetter{shards: shards}
}

func (r *attemptResetter) ResetAttemptsByJobID(ctx context.Context, shardName string, scope Scope, jobID string) error {
	shard, err := r.shards.ByName(shardName)
	if err != nil {
		return err
	}

	return shard.ResetAttemptsByJobID(ctx, scope, jobID)
}
