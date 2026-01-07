package queue

import (
	"context"
	"errors"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

// worker runs a blocking process that listens to items being pushed into the
// worker channel.  This allows us to process an individual item from a queue.
func (q *queueProcessor) worker(ctx context.Context, f RunFunc) {
	for {
		select {
		case <-ctx.Done():
			return

		case i := <-q.workers:
			// Create a new context which isn't cancelled by the parent, when quit.
			// XXX: When jobs can have their own cancellation signals, move this into
			// process itself.
			processCtx, cancel := context.WithCancel(context.Background())
			err := q.ProcessItem(processCtx, i, f)
			q.sem.Release(1)
			metrics.WorkerQueueCapacityCounter(ctx, -1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name()}})
			cancel()
			if err == nil {
				continue
			}

			// We handle the error individually within process, requeueing
			// the item into the queue.  Here, the worker can continue as
			// usual to process the next item.
			logger.StdlibLogger(ctx).Error("error processing queue item", "error", err, "item", i)
		}
	}
}

type ShadowPartitionChanMsg struct {
	ShadowPartition   *QueueShadowPartition
	ContinuationCount uint
}

func (q *queueProcessor) ShadowPartitionWorkers() chan ShadowPartitionChanMsg {
	return q.qspc
}

// shadowWorker runs a blocking process that listens to item being pushed into the
// shadow queue partition channel. This allows us to process an individual shadow partition.
func (q *queueProcessor) shadowWorker(ctx context.Context, qspc chan ShadowPartitionChanMsg) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-qspc:
			_, err := DurationWithTags(
				ctx,
				q.primaryQueueShard.Name(),
				"shadow_partition_process_duration",
				q.Clock().Now(),
				func(ctx context.Context) (any, error) {
					err := q.ProcessShadowPartition(ctx, msg.ShadowPartition, msg.ContinuationCount)
					if errors.Is(err, context.Canceled) {
						return nil, nil
					}
					return nil, err
				},
				map[string]any{
					// 	"partition_id": msg.sp.PartitionID,
				},
			)
			if err != nil {
				logger.StdlibLogger(ctx).Error("could not scan shadow partition", "error", err, "shadow_part", msg.ShadowPartition, "continuation_count", msg.ContinuationCount)
			}
		}
	}
}
