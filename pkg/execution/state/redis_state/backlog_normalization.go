package redis_state

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

func (q *queue) leaseBacklogForNormalization(ctx context.Context, bl *QueueBacklog) error {
	leaseExpiry := q.clock.Now().Add(BacklogNormalizeLeaseDuration)
	leaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return fmt.Errorf("could not generate leaseID: %w", err)
	}

	shard := q.primaryQueueShard

	rc := shard.RedisClient.Client()
	cmd := rc.B().
		Set().
		Key(shard.RedisClient.kg.BacklogNormalizationLease(bl.BacklogID)).
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

	return errBacklogAlreadyLeasedForNormalization
}

func (q *queue) extendBacklogNormalizationLease(ctx context.Context, now time.Time, bl *QueueBacklog) error {
	leaseExpiry := now.Add(BacklogNormalizeLeaseDuration)
	newLeaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return fmt.Errorf("could not generate newLeaseID: %w", err)
	}

	shard := q.primaryQueueShard

	rc := shard.RedisClient.Client()
	cmd := rc.B().
		Set().
		Key(shard.RedisClient.kg.BacklogNormalizationLease(bl.BacklogID)).
		Value(newLeaseID.String()).
		Xx().
		Get().
		Exat(leaseExpiry).
		Build()

	_, err = rc.Do(ctx, cmd).ToAny()
	if err == rueidis.Nil {
		return errBacklogNormalizationLeaseExpired
	}
	if err != nil {
		return err
	}

	// successfully extended lease
	return nil
}

func (q *queue) ShadowPartitionPeekNormalizeBacklogs(ctx context.Context, sp *QueueShadowPartition, limit int64) ([]*QueueBacklog, error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for ShadowPartitionPeekNormalizeBacklogs: %s", q.primaryQueueShard.Kind)
	}

	rc := q.primaryQueueShard.RedisClient

	partitionNormalizeSet := rc.kg.PartitionNormalizeSet(sp.PartitionID)

	p := peeker[QueueBacklog]{
		q:               q,
		opName:          "ShadowPartitionPeekNormalizeBacklogs",
		keyMetadataHash: q.primaryQueueShard.RedisClient.kg.BacklogMeta(),
		max:             NormalizePartitionPeekMax,
		maker: func() *QueueBacklog {
			return &QueueBacklog{}
		},
		handleMissingItems: CleanupMissingPointers(ctx, partitionNormalizeSet, rc.Client(), q.log.With("sp", sp)),
		// faster option: load items regardless of zscore
		ignoreUntil:            true,
		isMillisecondPrecision: true,
	}

	res, err := p.peek(ctx, partitionNormalizeSet, false, q.clock.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("could not peek backlogs for normalization: %w", err)
	}

	return res.Items, nil
}

func (q *queue) BacklogNormalizePeek(ctx context.Context, b *QueueBacklog, limit int64) (*peekResult[osqueue.QueueItem], error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for BacklogNormalizePeek: %s", q.primaryQueueShard.Kind)
	}

	rc := q.primaryQueueShard.RedisClient

	backlogSet := rc.kg.BacklogSet(b.BacklogID)

	p := peeker[osqueue.QueueItem]{
		q:               q,
		opName:          "BacklogNormalizePeek",
		keyMetadataHash: q.primaryQueueShard.RedisClient.kg.QueueItem(),
		max:             NormalizeBacklogPeekMax,
		maker: func() *osqueue.QueueItem {
			return &osqueue.QueueItem{}
		},
		handleMissingItems: CleanupMissingPointers(ctx, backlogSet, rc.Client(), q.log.With("backlog", b)),
		// faster option: load items regardless of zscore
		ignoreUntil:            true,
		isMillisecondPrecision: true,
	}

	// this is essentially +inf as no queue items should ever be scheduled >2y out
	normalizeLookahead := q.clock.Now().Add(time.Hour * 24 * 365 * 2)

	res, err := p.peek(ctx, backlogSet, false, normalizeLookahead, limit)
	if err != nil {
		return nil, fmt.Errorf("could not peek backlog items for normalization: %w", err)
	}

	return res, nil
}
