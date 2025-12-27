package queue

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/redis/rueidis"
	"golang.org/x/sync/errgroup"
)

func (q *queueProcessor) Migrate(ctx context.Context, sourceShardName string, fnID uuid.UUID, limit int64, concurrency int, handler QueueMigrationHandler) (int64, error) {
	l := logger.StdlibLogger(ctx)
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "MigrationPeek"), redis_telemetry.ScopeQueue)

	shard, ok := q.queueShardClients[sourceShardName]
	if !ok {
		return -1, fmt.Errorf("no queue shard available for '%s'", sourceShardName)
	}

	from := time.Time{}
	// setting it to 5 years ahead should be enough to cover all queue items in the partition
	until := q.Clock().Now().Add(24 * time.Hour * 365 * 5)
	items, err := shard.ItemsByPartition(ctx, fnID.String(), from, until,
		WithQueueItemIterBatchSize(limit),
	)
	if err != nil {
		// the partition doesn't exist, meaning there are no workloads
		if errors.Is(err, rueidis.Nil) {
			return 0, nil
		}

		return -1, fmt.Errorf("error preparing partition iteration: %w", err)
	}

	// Should process in order because we don't want out of order execution when moved over
	var processed int64

	process := func(qi *QueueItem) error {
		if err := handler(ctx, qi); err != nil {
			return err
		}

		if err := shard.Dequeue(ctx, *qi); err != nil {
			l.ReportError(err, "error dequeueing queue item after migration")
		}

		atomic.AddInt64(&processed, 1)
		return nil
	}

	if concurrency > 0 {
		eg := errgroup.Group{}
		eg.SetLimit(concurrency)

		for qi := range items {
			i := qi
			eg.Go(func() error {
				return process(i)
			})
		}

		err := eg.Wait()
		if err != nil {
			return atomic.LoadInt64(&processed), err
		}

		return atomic.LoadInt64(&processed), nil
	}

	for qi := range items {
		if err := process(qi); err != nil {
			return processed, err
		}
	}

	return atomic.LoadInt64(&processed), nil
}
