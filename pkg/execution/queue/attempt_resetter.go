package queue

import "context"

type attemptResetter struct {
	shards ShardRegistry
}

// NewAttemptResetter returns an attempt resetter backed by the provided shard registry.
func NewAttemptResetter(shards ShardRegistry) AttemptResetter {
	return &attemptResetter{shards: shards}
}

func (r *attemptResetter) ResetAttemptsByJobID(ctx context.Context, shardName string, scope Scope, jobID string) error {
	shard, err := r.shards.ByName(shardName)
	if err != nil {
		return err
	}

	return shard.ResetAttemptsByJobID(ctx, scope, jobID)
}
