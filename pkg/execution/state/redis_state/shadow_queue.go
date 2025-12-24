package redis_state

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

func (q *queue) IsMigrationLocked(ctx context.Context, fnID uuid.UUID) (*time.Time, error) {
	client := q.RedisClient.Client()
	kg := q.RedisClient.KeyGenerator()
	cmd := client.B().Get().Key(kg.QueueMigrationLock(fnID)).Build()
	exists, err := client.Do(ctx, cmd).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not check for migration lock: %w", err)
	}

	parsed, err := ulid.Parse(exists)
	if err != nil {
		return nil, fmt.Errorf("invalid lock format: %w", err)
	}

	lockUntil := parsed.Timestamp()
	return &lockUntil, nil
}

// peekShadowPartitions returns pending shadow partitions within the global shadow partition pointer _or_ account shadow partition pointer ZSET.
func (q *queue) peekShadowPartitions(ctx context.Context, partitionIndexKey string, sequential bool, peekLimit int64, until time.Time) ([]*QueueShadowPartition, error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for peekShadowPartitions: %s", q.primaryQueueShard.Kind)
	}

	p := peeker[QueueShadowPartition]{
		q:               q,
		opName:          "peekShadowPartitions",
		keyMetadataHash: q.primaryQueueShard.RedisClient.kg.ShadowPartitionMeta(),
		max:             ShadowPartitionPeekMax,
		maker: func() *QueueShadowPartition {
			return &QueueShadowPartition{}
		},
		handleMissingItems: func(pointers []string) error {
			return nil
		},
		isMillisecondPrecision: true,
	}

	res, err := p.peek(ctx, partitionIndexKey, sequential, until, peekLimit)
	if err != nil {
		if errors.Is(err, ErrPeekerPeekExceedsMaxLimits) {
			return nil, ErrShadowPartitionPeekMaxExceedsLimits
		}
		return nil, fmt.Errorf("could not peek shadow partitions: %w", err)
	}

	if res.TotalCount > 0 {
		for _, p := range res.Items {
			q.log.Trace("peeked shadow partition", "partition_id", p.PartitionID, "until", until.Format(time.StampMilli))
		}
	}

	return res.Items, nil
}

func (q *queue) ShadowPartitionPeek(ctx context.Context, sp *QueueShadowPartition, sequential bool, until time.Time, limit int64, opts ...PeekOpt) ([]*QueueBacklog, int, error) {
	opt := peekOption{}
	for _, apply := range opts {
		apply(&opt)
	}

	shadowPartitionSet := rc.kg.ShadowPartitionSet(sp.PartitionID)

	p := peeker[QueueBacklog]{
		q:               q,
		opName:          "ShadowPartitionPeek",
		keyMetadataHash: rc.kg.BacklogMeta(),
		max:             ShadowPartitionPeekMaxBacklogs,
		maker: func() *QueueBacklog {
			return &QueueBacklog{}
		},
		handleMissingItems:     CleanupMissingPointers(ctx, shadowPartitionSet, rc.Client(), q.log.With("sp", sp)),
		isMillisecondPrecision: true,
	}

	res, err := p.peek(ctx, shadowPartitionSet, sequential, until, limit, opts...)
	if err != nil {
		if errors.Is(err, ErrPeekerPeekExceedsMaxLimits) {
			return nil, 0, ErrShadowPartitionBacklogPeekMaxExceedsLimits
		}
		return nil, 0, fmt.Errorf("could not peek shadow partition backlogs: %w", err)
	}

	return res.Items, res.TotalCount, nil
}

func (q *queue) ShadowPartitionExtendLease(ctx context.Context, sp *QueueShadowPartition, leaseID ulid.ULID, duration time.Duration) (*ulid.ULID, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ShadowPartitionExtendLease"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for ShadowPartitionExtendLease: %s", q.primaryQueueShard.Kind)
	}

	now := q.clock.Now()
	leaseExpiry := now.Add(duration)
	newLeaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("could not generate new leaseID: %w", err)
	}

	sp.LeaseID = &newLeaseID

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	keys := []string{
		q.primaryQueueShard.RedisClient.kg.ShadowPartitionMeta(),
		q.primaryQueueShard.RedisClient.kg.GlobalShadowPartitionSet(),
		q.primaryQueueShard.RedisClient.kg.GlobalAccountShadowPartitions(),
		q.primaryQueueShard.RedisClient.kg.AccountShadowPartitions(accountID),
	}
	args, err := StrSlice([]any{
		sp.PartitionID,
		accountID,
		leaseID,
		newLeaseID,
		now.UnixMilli(),
		leaseExpiry.UnixMilli(),
	})
	if err != nil {
		return nil, fmt.Errorf("could not serialize args: %w", err)
	}

	status, err := scripts["queue/shadowPartitionExtendLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "shadowPartitionExtendLease"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error extending shadow partition lease: %w", err)
	}
	switch status {
	case 0:
		return &newLeaseID, nil
	case -1:
		return nil, ErrShadowPartitionNotFound
	case -2:
		return nil, ErrShadowPartitionLeaseNotFound
	case -3:
		return nil, ErrShadowPartitionAlreadyLeased
	default:
		return nil, fmt.Errorf("unknown response extending shadow partition lease: %v (%T)", status, status)
	}
}

func (q *queue) ShadowPartitionRequeue(ctx context.Context, sp *QueueShadowPartition, requeueAt *time.Time) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ShadowPartitionRequeue"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for ShadowPartitionRequeue: %s", q.primaryQueueShard.Kind)
	}

	sp.LeaseID = nil

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	var requeueAtMS int64
	var requeueAtStr string
	if requeueAt != nil {
		requeueAtMS = requeueAt.UnixMilli()
		requeueAtStr = requeueAt.Format(time.StampMilli)
	}

	keys := []string{
		q.primaryQueueShard.RedisClient.kg.ShadowPartitionMeta(),
		q.primaryQueueShard.RedisClient.kg.GlobalShadowPartitionSet(),
		q.primaryQueueShard.RedisClient.kg.GlobalAccountShadowPartitions(),
		q.primaryQueueShard.RedisClient.kg.AccountShadowPartitions(accountID),
		q.primaryQueueShard.RedisClient.kg.ShadowPartitionSet(sp.PartitionID),
	}
	args, err := StrSlice([]any{
		sp.PartitionID,
		accountID,
		q.clock.Now().UnixMilli(),
		requeueAtMS,
	})
	if err != nil {
		return fmt.Errorf("could not serialize args: %w", err)
	}

	status, err := scripts["queue/shadowPartitionRequeue"].Exec(
		redis_telemetry.WithScriptName(ctx, "shadowPartitionRequeue"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error returning shadow partition lease: %w", err)
	}

	q.log.Trace("requeued shadow partition",
		"id", sp.PartitionID,
		"time", requeueAtStr,
		"status", status,
	)

	switch status {
	case 0:
		return nil
	case -1:
		metrics.IncrQueueShadowPartitionLeaseContentionCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"queue_shard": q.primaryQueueShard.Name,
				// "partition_id": sp.PartitionID,
				"action": "not_found",
			},
		})

		return ErrShadowPartitionNotFound
	default:
		return fmt.Errorf("unknown response returning shadow partition lease: %v (%T)", status, status)
	}
}

func (q *queue) ShadowPartitionLease(ctx context.Context, sp *QueueShadowPartition, duration time.Duration) (*ulid.ULID, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ShadowPartitionLease"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for ShadowPartitionLease: %s", q.primaryQueueShard.Kind)
	}

	now := q.clock.Now()
	leaseExpiry := now.Add(duration)
	leaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("could not generate leaseID: %w", err)
	}

	sp.LeaseID = &leaseID

	accountID := uuid.Nil
	if sp.AccountID != nil {
		accountID = *sp.AccountID
	}

	keys := []string{
		q.primaryQueueShard.RedisClient.kg.ShadowPartitionMeta(),
		q.primaryQueueShard.RedisClient.kg.GlobalShadowPartitionSet(),
		q.primaryQueueShard.RedisClient.kg.GlobalAccountShadowPartitions(),
		q.primaryQueueShard.RedisClient.kg.AccountShadowPartitions(accountID),
	}
	args, err := StrSlice([]any{
		sp.PartitionID,
		accountID,
		leaseID,
		now.UnixMilli(),
		leaseExpiry.UnixMilli(),
	})
	if err != nil {
		return nil, fmt.Errorf("could not serialize args: %w", err)
	}

	status, err := scripts["queue/shadowPartitionLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "shadowPartitionLease"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error leasing shadow partition: %w", err)
	}
	switch status {
	case 0:
		return &leaseID, nil
	case -1:
		return nil, ErrShadowPartitionNotFound
	case -2:
		return nil, ErrShadowPartitionAlreadyLeased
	default:
		return nil, fmt.Errorf("unknown response leasing shadow partition: %v (%T)", status, status)
	}
}

func (q *queue) PeekGlobalShadowPartitionAccounts(ctx context.Context, sequential bool, until time.Time, limit int64) ([]uuid.UUID, error) {
	p := peeker[osqueue.QueueBacklog]{
		q:                      q,
		opName:                 "peekGlobalShadowPartitionAccounts",
		max:                    osqueue.ShadowPartitionAccountPeekMax,
		isMillisecondPrecision: true,
	}

	return p.peekUUIDPointer(ctx, q.RedisClient.kg.GlobalAccountShadowPartitions(), sequential, until, limit)
}
