package pauses

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/execution/queue"
)

// BlockFlushEnqueuer is an interface that enqueues and runs block flush jobs.  This may be an
// in-memory implementation for testing, or a queue-backed inplementation in production.
type BlockFlushEnqueuer interface {
	Enqueue(ctx context.Context, index Index) error
}

// QueueFlushProcessor enqueues jobs to the given queue so that flushing happens in some
// background process, handled outside of this package.
func QueueFlushProcessor(q queue.Queue) BlockFlushEnqueuer {
	return flushQueue{q}
}

// InMemoryFlushProcessor runs block flushes in-process without enqueueng.
func InMemoryFlushProcessor(f BlockFlusher) BlockFlushEnqueuer {
	return &flushInProcess{f: f}
}

type flushQueue struct {
	queue queue.Queue
}

func (f flushQueue) Enqueue(ctx context.Context, index Index) error {
	// It's fine to enqueue another flush job as soon as the last one
	// was done but we can't do less than a second as Redis expiration.
	blockFlushIdempotencyPeriod := time.Second * 1

	jid := fmt.Sprintf("%s-%s-flush", index.WorkspaceID, index.EventName)
	return f.queue.Enqueue(ctx, queue.Item{
		JobID:       &jid,
		WorkspaceID: index.WorkspaceID,
		Kind:        queue.KindPauseBlockFlush,
		Payload: queue.PayloadPauseBlockFlush{
			EventName: index.EventName,
		},
		QueueName: &BlockFlushQueueName,
	}, time.Now(), queue.EnqueueOpts{
		IdempotencyPeriod: &blockFlushIdempotencyPeriod,
	})
}

type flushInProcess struct {
	counter int32
	f       BlockFlusher
}

func (f *flushInProcess) Enqueue(ctx context.Context, index Index) error {
	atomic.AddInt32(&f.counter, 1)
	return f.f.FlushIndexBlock(ctx, index)
}
