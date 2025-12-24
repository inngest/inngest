package redis_state

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

func (q *queue) LeaseBacklogForNormalization(ctx context.Context, bl *osqueue.QueueBacklog) error {
	leaseExpiry := q.Clock.Now().Add(osqueue.BacklogNormalizeLeaseDuration)
	leaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return fmt.Errorf("could not generate leaseID: %w", err)
	}

	rc := q.RedisClient.Client()
	cmd := rc.B().
		Set().
		Key(q.RedisClient.kg.BacklogNormalizationLease(bl.BacklogID)).
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

	return osqueue.ErrBacklogAlreadyLeasedForNormalization
}

func (q *queue) ExtendBacklogNormalizationLease(ctx context.Context, now time.Time, bl *osqueue.QueueBacklog) error {
	leaseExpiry := now.Add(osqueue.BacklogNormalizeLeaseDuration)
	newLeaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return fmt.Errorf("could not generate newLeaseID: %w", err)
	}

	rc := q.RedisClient.Client()
	cmd := rc.B().
		Set().
		Key(q.RedisClient.kg.BacklogNormalizationLease(bl.BacklogID)).
		Value(newLeaseID.String()).
		Xx().
		Get().
		Exat(leaseExpiry).
		Build()

	_, err = rc.Do(ctx, cmd).ToAny()
	if err == rueidis.Nil {
		return osqueue.ErrBacklogNormalizationLeaseExpired
	}
	if err != nil {
		return err
	}

	// successfully extended lease
	return nil
}

func (q *queue) ShadowPartitionPeekNormalizeBacklogs(ctx context.Context, sp *osqueue.QueueShadowPartition, limit int64) ([]*osqueue.QueueBacklog, error) {
	partitionNormalizeSet := q.RedisClient.kg.PartitionNormalizeSet(sp.PartitionID)

	p := peeker[osqueue.QueueBacklog]{
		q:               q,
		opName:          "ShadowPartitionPeekNormalizeBacklogs",
		keyMetadataHash: q.RedisClient.kg.BacklogMeta(),
		max:             osqueue.NormalizePartitionPeekMax,
		maker: func() *osqueue.QueueBacklog {
			return &osqueue.QueueBacklog{}
		},
		handleMissingItems: CleanupMissingPointers(ctx, partitionNormalizeSet, q.RedisClient.Client(), logger.StdlibLogger(ctx).With("sp", sp)),
		// faster option: load items regardless of zscore
		ignoreUntil:            true,
		isMillisecondPrecision: true,
	}

	res, err := p.peek(ctx, partitionNormalizeSet, false, q.Clock.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("could not peek backlogs for normalization: %w", err)
	}

	return res.Items, nil
}

func (q *queue) BacklogNormalizePeek(ctx context.Context, b *osqueue.QueueBacklog, limit int64) (*osqueue.PeekResult[osqueue.QueueItem], error) {
	backlogSet := q.RedisClient.kg.BacklogSet(b.BacklogID)

	p := peeker[osqueue.QueueItem]{
		q:               q,
		opName:          "BacklogNormalizePeek",
		keyMetadataHash: q.RedisClient.kg.QueueItem(),
		max:             osqueue.NormalizeBacklogPeekMax,
		maker: func() *osqueue.QueueItem {
			return &osqueue.QueueItem{}
		},
		handleMissingItems: CleanupMissingPointers(ctx, backlogSet, q.RedisClient.Client(), logger.StdlibLogger(ctx).With("backlog", b)),
		// faster option: load items regardless of zscore
		ignoreUntil:            true,
		isMillisecondPrecision: true,
	}

	// this is essentially +inf as no queue items should ever be scheduled >2y out
	normalizeLookahead := q.Clock.Now().Add(time.Hour * 24 * 365 * 2)

	res, err := p.peek(ctx, backlogSet, false, normalizeLookahead, limit)
	if err != nil {
		return nil, fmt.Errorf("could not peek backlog items for normalization: %w", err)
	}

	return res, nil
}
