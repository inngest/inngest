package redis_state

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
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

	// Nothing left to normalize in this partition: drop the account -> partition pointer so
	// the scanner stops revisiting it.
	if len(res.Items) == 0 {
		accountID := uuid.Nil
		if sp.AccountID != nil {
			accountID = *sp.AccountID
		}
		if err := q.normalizeCleanupPointers(ctx, &accountID, sp.PartitionID, "", false); err != nil {
			return nil, err
		}
	}

	return res.Items, nil
}

// PeekAccountNormalizePartitions returns shadow partitions in the account that have
// backlogs pending normalization.
func (q *queue) PeekAccountNormalizePartitions(ctx context.Context, accountID uuid.UUID, until time.Time, limit int64) ([]*osqueue.QueueShadowPartition, error) {
	l := logger.StdlibLogger(ctx).With("account_id", accountID)

	p := peeker[osqueue.QueueShadowPartition]{
		q:               q,
		opName:          "peekAccountNormalizePartitions",
		keyMetadataHash: q.RedisClient.kg.ShadowPartitionMeta(),
		max:             osqueue.ShadowPartitionPeekMax,
		maker: func() *osqueue.QueueShadowPartition {
			return &osqueue.QueueShadowPartition{}
		},
		handleMissingItems: func(partitionIDs []string) error {
			// Without partition metadata the partition can never be normalized, so drop the
			// pointer rather than rescanning it forever.
			for _, partitionID := range partitionIDs {
				if err := q.normalizeCleanupPointers(ctx, &accountID, partitionID, "", true); err != nil {
					l.Warn("could not clean up normalize pointer for missing partition", "err", err, "partition_id", partitionID)
				}
			}
			return nil
		},
		isMillisecondPrecision: true,
	}

	res, err := p.peek(ctx, q.RedisClient.kg.AccountNormalizeSet(accountID), false, until, limit)
	if err != nil {
		if errors.Is(err, ErrPeekerPeekExceedsMaxLimits) {
			return nil, osqueue.ErrShadowPartitionPeekMaxExceedsLimits
		}
		return nil, fmt.Errorf("could not peek account normalize partitions: %w", err)
	}

	// The account is in the global normalize set but has no partitions to normalize: drop
	// the global -> account pointer if the account set is empty.
	if res.TotalCount == 0 {
		if err := q.normalizeCleanupPointers(ctx, &accountID, "", "", false); err != nil {
			return nil, err
		}
	}

	return res.Items, nil
}

// normalizeCleanupPointers drops normalization pointers whose target sets are empty. With
// force, the partition pointer is dropped from the account normalize set unconditionally.
// A nil accountID skips the account and global pointers.
func (q *queue) normalizeCleanupPointers(ctx context.Context, accountID *uuid.UUID, partitionID string, backlogID string, force bool) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "normalizeCleanupPointers"), redis_telemetry.ScopeQueue)

	kg := q.RedisClient.kg
	accountArg := ""
	keyAccountNormalizeSet := kg.AccountNormalizeSet(uuid.Nil)
	if accountID != nil {
		accountArg = accountID.String()
		keyAccountNormalizeSet = kg.AccountNormalizeSet(*accountID)
	}
	keys := []string{
		kg.BacklogSet(backlogID),
		kg.PartitionNormalizeSet(partitionID),
		keyAccountNormalizeSet,
		kg.GlobalAccountNormalizeSet(),
	}

	forceArg := 0
	if force {
		forceArg = 1
	}
	args, err := StrSlice([]any{
		backlogID,
		partitionID,
		accountArg,
		forceArg,
	})
	if err != nil {
		return fmt.Errorf("could not serialize args: %w", err)
	}

	err = scripts["queue/normalizeCleanupPointers"].Exec(
		redis_telemetry.WithScriptName(ctx, "normalizeCleanupPointers"),
		q.RedisClient.unshardedRc,
		keys,
		args,
	).Error()
	if err != nil {
		return fmt.Errorf("could not clean up normalize pointers: %w", err)
	}

	return nil
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

	// Enqueue only cleans up normalize pointers when it moves the last item out of the
	// backlog. A backlog that was already empty would otherwise stay in the normalize sets.
	if res.TotalCount == 0 {
		// The account is unknown here, so only the partition -> backlog pointer is dropped.
		// The next ShadowPartitionPeekNormalizeBacklogs cleans up the remaining pointers.
		if err := q.normalizeCleanupPointers(ctx, nil, b.ShadowPartitionID, b.BacklogID, false); err != nil {
			return nil, err
		}
	}

	return res, nil
}
