package redis_state

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

var (
	errCancellationLeaseExpired  = fmt.Errorf("cancellation lease expired")
	errCancellationAlreadyLeased = fmt.Errorf("cancellation already leased")
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
	cancellations, err := q.peekCancellationPartitions(
		ctx,
		q.primaryQueueShard.RedisClient.kg.CancellationPartitionSet(),
		sequential,
		CancellationPartitionPeekMax,
		until,
	)
	if err != nil {
		return fmt.Errorf("error peeking cancellation partitions: %w", err)
	}

	for _, qc := range cancellations {
		// lease the item to reduce contention
		_, err := duration(ctx, q.primaryQueueShard.Name, "cancellation_lease", q.clock.Now(), func(ctx context.Context) (any, error) {
			return nil, q.leaseCancellation(ctx, qc)
		})
		if err != nil {
			if errors.Is(err, errCancellationAlreadyLeased) {
				continue
			}
			return fmt.Errorf("error leasing cancellation: %w", err)
		}

		metrics.IncrCancellationScannedCounter(ctx, metrics.CounterOpt{PkgName: pkgName})

		cc <- cancellationChanMsg{
			item: qc,
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
		keyMetadataHash: q.primaryQueueShard.RedisClient.kg.CancellationPartitionMeta(),
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
	l := q.log.With("cancellation", qc)

	// extend lease
	extendLeaseCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		for {
			select {
			case <-extendLeaseCtx.Done():
				return
			case <-time.Tick(CancellationLeaseDuration / 2):
				if err := q.extendCancellationLease(ctx, q.clock.Now(), qc); err != nil {
					switch err {
					// can't extend lease since it's already expired
					case errCancellationLeaseExpired:
						return
					}
					l.Error("error extending cancellation lease", "error", err, "cancellation", qc)
					return
				}
			}
		}
	}()

	// TODO: implement handling of cancellation
	switch qc.Type {
	case enums.CancellationTypeBacklog:
	case enums.CancellationTypeRun:
		// no-op, don't know what to do with these options
	default:
	}

	return nil
}

func (q *queue) leaseCancellation(ctx context.Context, qc *QueueCancellation) error {
	leaseExpiry := q.clock.Now().Add(CancellationLeaseDuration)
	leaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return fmt.Errorf("could not generate lease ID for cancellation: %w", err)
	}

	shard := q.primaryQueueShard
	rc := shard.RedisClient.Client()
	cmd := rc.B().
		Set().
		Key(shard.RedisClient.kg.CancellationLease(qc.CancelID)).
		Value(leaseID.String()).
		Nx().
		Get().
		Exat(leaseExpiry).
		Build()

	_, err = rc.Do(ctx, cmd).ToString()
	if err == rueidis.Nil {
		// successfully leased since prior value was nil
		return nil
	}
	if err != nil {
		return err
	}
	return errCancellationAlreadyLeased
}

func (q *queue) extendCancellationLease(ctx context.Context, now time.Time, qc *QueueCancellation) error {
	leaseExpiry := now.Add(CancellationLeaseDuration)
	leaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return fmt.Errorf("could not generate extended lease ID for cancellation: %w", err)
	}

	shard := q.primaryQueueShard
	rc := shard.RedisClient.Client()
	cmd := rc.B().
		Set().
		Key(shard.RedisClient.kg.CancellationLease(qc.CancelID)).
		Value(leaseID.String()).
		Xx().
		Get().
		Exat(leaseExpiry).
		Build()

	_, err = rc.Do(ctx, cmd).ToString()
	if err == rueidis.Nil {
		return errCancellationLeaseExpired
	}
	if err != nil {
		return err
	}
	// successfully extended lease
	return nil
}

// QueueCancellation represents an object that needs to be cancelled by the queue
type QueueCancellation struct {
	// CancelID is the identifier for this cancellation job
	CancelID string
	// TargetID represents the identifier of the target that needs to be cancelled.
	// This should be used with coordination of the `Type` attribute.
	//
	// e.g. Types
	// - run -> runID
	// - backlog -> backlogID
	TargetID string `json:"id"`
	// Type indicates what type of cancellation is this for
	Type enums.CancellationType `json:"t"`
	// ReferenceID is the reference or source that initiated this cancellation
	ReferenceID string `json:"ref"`
	// Cause shows how this cancellation was initiated
	Cause enums.CancellationCause `json:"cause"`
}
