package redis_state

import (
	"context"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/enums"
)

type cancellationChanMsg struct {
	item *QueueCancellation
}

func (q *queue) cancellationScan(ctx context.Context) error {
	cc := make(chan cancellationChanMsg)

	for i := int32(0); i < q.numCancellationWorkers; i++ {
		go q.cancellationWorker(ctx, cc)
	}

	tick := q.clock.NewTicker(q.cancelPollTick)
	q.log.Debug("starting cancellation scanner", "poll", q.cancelPollTick.String())

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return nil

		case <-tick.Chan():
			until := q.clock.Now().Add(2 * time.Second)
			if err := q.iterateCancellationPartition(ctx, until, cc); err != nil {
				return fmt.Errorf("error scanning cancellation partition")
			}
		}
	}
}

func (q *queue) cancellationWorker(ctx context.Context, cc chan cancellationChanMsg) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-cc:
			err := q.processCancellation(ctx, msg.item)
			if err != nil {
				q.log.Error("error cancelling item", "type", msg.item.Type.String(), "id", msg.item.ID)
			}
		}
	}
}

func (q *queue) iterateCancellationPartition(ctx context.Context, until time.Time, cc chan cancellationChanMsg) error {
	// TODO: implement scanning
	return nil
}

func (q *queue) processCancellation(ctx context.Context, qc *QueueCancellation) error {
	// TODO: implement handling of cancellation
	switch qc.Type {
	case enums.CancellationTypeBacklog:
	case enums.CancellationTypeRun:
	case enums.CancellationTypeNone, enums.CancellationTypeEvent, enums.CancellationTypeManual:
		// no-op, don't know what to do with these options
	default:
	}

	return nil
}

// QueueCancellation represents an object that needs to be cancelled by the queue
type QueueCancellation struct {
	// ID represents the identifier of the target that needs to be cancelled.
	// This should be used with coordination of the `Type` attribute.
	//
	// e.g. Types
	// - run -> runID
	// - backlog -> backlogID
	ID   string                 `json:"id"`
	Type enums.CancellationType `json:"t"`
}
