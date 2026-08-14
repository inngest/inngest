package queue

import (
	"context"
	"fmt"
	"time"
)

type queueMigrationLocker struct {
	shards QueueShardRegistry
}

func newQueueMigrationLocker(shards QueueShardRegistry) MigrationLocker {
	return &queueMigrationLocker{
		shards: shards,
	}
}

func (m *queueMigrationLocker) SetFunctionMigrate(ctx context.Context, sourceShard string, scope Scope, migrateLockUntil *time.Time) error {
	shard, err := m.shards.ByName(sourceShard)
	if err != nil {
		return fmt.Errorf("could not find shard %q", sourceShard)
	}

	return shard.SetFunctionMigrate(ctx, scope, migrateLockUntil)
}
