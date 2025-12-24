package queue

import (
	"context"

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
			err := q.process(processCtx, i, f)
			q.sem.Release(1)
			metrics.WorkerQueueCapacityCounter(ctx, -1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.PrimaryQueueShard.Name()}})
			cancel()
			if err == nil {
				continue
			}

			// We handle the error individually within process, requeueing
			// the item into the queue.  Here, the worker can continue as
			// usual to process the next item.
			q.log.Error("error processing queue item", "error", err, "item", i)
		}
	}
}
