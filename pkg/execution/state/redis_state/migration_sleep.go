package redis_state

import (
	"context"
	"fmt"
	"time"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/oklog/ulid/v2"
)

// EnqueueSleepForMigration preserves both runs when a legacy sleep ID is reused
// after shard routing changes. Legacy duplicates and completion markers are
// checked atomically with the copy, as are ordinary retries of migrated sleeps.
func (q *queue) EnqueueSleepForMigration(ctx context.Context, source osqueue.QueueItem, at time.Time, opts osqueue.EnqueueOpts) (osqueue.QueueItem, error) {
	if opts.ForceQueueShardName != "" && opts.ForceQueueShardName != q.Name() {
		return source, fmt.Errorf("sleep migration must target shard %q", q.Name())
	}
	scope := osqueue.ScopeFromQueueItem(source)
	if err := scope.Validate(); err != nil {
		return source, err
	}
	if source.Data.Kind != osqueue.KindSleep || scope.IsSystem || source.ID == "" || source.Data.Identifier.RunID == (ulid.ULID{}) || !opts.PassthroughJobId || opts.NormalizeFromBacklogID != "" {
		return source, fmt.Errorf("run-scoped migration requires a function sleep item with a preserved ID")
	}
	item := source
	item.ID = osqueue.MigrationSleepID(source)
	item.Data.JobID = &item.ID
	item.LeaseID = nil
	item.CapacityLease = nil
	item.RefilledFrom = ""
	item.RefilledAt = 0
	item.EarliestPeekTime = 0
	item.GenerationID = 0
	return q.enqueueItem(ctx, item, at, opts, source.ID)
}
