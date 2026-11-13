package redis_state

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
)

// HasPendingMigrationItems checks the pending indexes atomically after a batch.
// In particular, normalization moves backlogs out of the ordinary shadow index;
// the ordinary item iterator must not mistake that state for a drained source.
// Leased jobs are handled separately by the migration's in-flight transfer.
func (q *queue) HasPendingMigrationItems(ctx context.Context, scope osqueue.Scope, partitionID string) (bool, error) {
	if err := scope.Validate(); err != nil {
		return false, err
	}
	if partitionID == "" || !scope.IsSystem && partitionID != scope.FunctionID.String() {
		return false, fmt.Errorf("migration partition does not match scope")
	}
	kg := q.RedisClient.kg
	rc := q.RedisClient.Client()
	count, err := rc.Do(ctx, rc.B().Eval().Script(`return redis.call('ZCARD', KEYS[1]) + redis.call('ZCARD', KEYS[2]) + redis.call('ZCARD', KEYS[3])`).Numkeys(3).Key(
		kg.PartitionQueueSet(enums.PartitionTypeDefault, partitionID, ""),
		kg.ShadowPartitionSet(partitionID),
		kg.PartitionNormalizeSet(partitionID),
	).Build()).AsInt64()
	if err != nil {
		return false, fmt.Errorf("cannot inspect pending migration items: %w", err)
	}
	return count > 0, nil
}
