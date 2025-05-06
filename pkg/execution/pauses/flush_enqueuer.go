package pauses

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/execution/queue"
)

// BlockFlushEnqueuer is an interface that enqueues and runs block flush jobs.  This may be an
// in-memory implementation for testing, or a queue-backed inplementation in production.
type BlockFlushEnqueuer interface {
	Enqueue(ctx context.Context, index Index) error
}

func QueueFlushProcessor(q queue.Queue) BlockFlushEnqueuer {
	return flushQueue{q}
}

// InMemoryFlushProcessor runs block flushes in-process without enqueueng.
func InMemoryFlushProcessor(f BlockFlusher) BlockFlushEnqueuer {
	return flushInProcess{f: f}
}

type flushQueue struct {
	queue queue.Queue
}

func (f flushQueue) Enqueue(ctx context.Context, index Index) error {
	jid := fmt.Sprintf("%s-%s-flush", index.WorkspaceID, index.EventName)
	return f.queue.Enqueue(ctx, queue.Item{
		JobID:       &jid,
		WorkspaceID: index.WorkspaceID,
		Kind:        queue.KindPauseBlockFlush,
		Payload: queue.PayloadPauseBlockFlush{
			EventName: index.EventName,
		},
		QueueName: &BlockFlushQueueName,
	}, time.Now(), queue.EnqueueOpts{})
}

type flushInProcess struct {
	f BlockFlusher
}

func (f flushInProcess) Enqueue(ctx context.Context, index Index) error {
	return f.f.FlushIndexBlock(ctx, index)
}
