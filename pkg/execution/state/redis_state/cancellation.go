package redis_state

import (
	"context"
	"errors"
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
	sequential := false
	key := ""
	cancellations, err := q.peekCancellationPartitions(ctx, key, sequential, CancellationPartitionPeekMax, until)
	if err != nil {
		return fmt.Errorf("error peeking cancellation partitions: %w", err)
	}

	for _, c := range cancellations {
		// TODO: lease the item to reduce contention

		cc <- cancellationChanMsg{
			item: c,
		}
	}

	return nil
}

func (q *queue) peekCancellationPartitions(ctx context.Context, partitionIndexKey string, sequential bool, peekLimit int64, until time.Time) ([]*QueueCancellation, error) {
	if q.isQueueShardKindAllowed(q.primaryQueueShard.Kind) {
		return nil, fmt.Errorf("unsupported queue shard kind for peekCancellationPartition: %s", q.primaryQueueShard.Kind)
	}

	p := peeker[QueueCancellation]{
		q:               q,
		opName:          "peekCancellation",
		keyMetadataHash: "", // TODO: replace this key
		max:             CancellationPartitionPeekMax,
		maker: func() *QueueCancellation {
			return &QueueCancellation{}
		},
		handleMissingItems: func(pointers []string) error {
			q.log.Warn("found missing cancellation partitions", "missing", pointers, "partitionKey", partitionIndexKey)
			return nil
		},
		isMillisecondPrecision: true,
	}

	res, err := p.peek(ctx, partitionIndexKey, sequential, until, peekLimit)
	if err != nil {
		if errors.Is(err, ErrPeekerPeekExceedsMaxLimits) {
			return nil, ErrCancellationPartitionPeekMaxExceedsLimits
		}
		return nil, fmt.Errorf("could not peek cancellation partition: %w", err)
	}

	if res.TotalCount > 0 {
		for _, item := range res.Items {
			q.log.Trace("peeked cancellation partition", "id", item.ID, "type", item.Type.String())
		}
	}

	return res.Items, nil
}

func (q *queue) processCancellation(ctx context.Context, qc *QueueCancellation) error {
	// TODO: extend lease

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
