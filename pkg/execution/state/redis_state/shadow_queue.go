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
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"go.opentelemetry.io/otel/attribute"
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
func (q *queue) PeekShadowPartitions(ctx context.Context, accountID *uuid.UUID, sequential bool, peekLimit int64, until time.Time) ([]*osqueue.QueueShadowPartition, error) {
	l := logger.StdlibLogger(ctx)

	key := q.RedisClient.kg.GlobalShadowPartitionSet()
	if accountID != nil {
		key = q.RedisClient.kg.AccountShadowPartitions(*accountID)
	}

	p := peeker[osqueue.QueueShadowPartition]{
		q:               q,
		opName:          "peekShadowPartitions",
		keyMetadataHash: q.RedisClient.kg.ShadowPartitionMeta(),
		max:             osqueue.ShadowPartitionPeekMax,
		maker: func() *osqueue.QueueShadowPartition {
			return &osqueue.QueueShadowPartition{}
		},
		handleMissingItems: func(pointers []string) error {
			return nil
		},
		isMillisecondPrecision: true,
	}

	res, err := p.peek(ctx, key, sequential, until, peekLimit)
	if err != nil {
		if errors.Is(err, ErrPeekerPeekExceedsMaxLimits) {
			return nil, osqueue.ErrShadowPartitionPeekMaxExceedsLimits
		}
		return nil, fmt.Errorf("could not peek shadow partitions: %w", err)
	}

	if res.TotalCount > 0 {
		for _, p := range res.Items {
			l.Trace("peeked shadow partition", "partition_id", p.PartitionID, "until", until.Format(time.StampMilli))
		}
	}

	return res.Items, nil
}

func (q *queue) ShadowPartitionPeek(ctx context.Context, sp *osqueue.QueueShadowPartition, sequential bool, until time.Time, limit int64, opts ...osqueue.PeekOpt) ([]*osqueue.QueueBacklog, int, error) {
	opt := osqueue.PeekOption{}
	for _, apply := range opts {
		apply(&opt)
	}

	shadowPartitionSet := q.RedisClient.kg.ShadowPartitionSet(sp.PartitionID)

	p := peeker[osqueue.QueueBacklog]{
		q:               q,
		opName:          "ShadowPartitionPeek",
		keyMetadataHash: q.RedisClient.kg.BacklogMeta(),
		max:             osqueue.ShadowPartitionPeekMaxBacklogs,
		maker: func() *osqueue.QueueBacklog {
			return &osqueue.QueueBacklog{}
		},
		handleMissingItems:     CleanupMissingPointers(ctx, shadowPartitionSet, q.RedisClient.Client(), logger.StdlibLogger(ctx).With("sp", sp)),
		isMillisecondPrecision: true,
	}

	res, err := p.peek(ctx, shadowPartitionSet, sequential, until, limit, opts...)
	if err != nil {
		if errors.Is(err, ErrPeekerPeekExceedsMaxLimits) {
			return nil, 0, osqueue.ErrShadowPartitionBacklogPeekMaxExceedsLimits
		}
		return nil, 0, fmt.Errorf("could not peek shadow partition backlogs: %w", err)
	}

	return res.Items, res.TotalCount, nil
}

func (q *queue) ShadowPartitionExtendLease(ctx context.Context, sp *osqueue.QueueShadowPartition, leaseID ulid.ULID, duration time.Duration) (*ulid.ULID, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ShadowPartitionExtendLease"), redis_telemetry.ScopeQueue)

	now := q.Clock.Now()
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
		q.RedisClient.kg.ShadowPartitionMeta(),
		q.RedisClient.kg.GlobalShadowPartitionSet(),
		q.RedisClient.kg.GlobalAccountShadowPartitions(),
		q.RedisClient.kg.AccountShadowPartitions(accountID),
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
		q.RedisClient.unshardedRc,
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
		return nil, osqueue.ErrShadowPartitionNotFound
	case -2:
		return nil, osqueue.ErrShadowPartitionLeaseNotFound
	case -3:
		return nil, osqueue.ErrShadowPartitionAlreadyLeased
	default:
		return nil, fmt.Errorf("unknown response extending shadow partition lease: %v (%T)", status, status)
	}
}

func (q *queue) ShadowPartitionRequeue(ctx context.Context, sp *osqueue.QueueShadowPartition, requeueAt *time.Time) error {
	l := logger.StdlibLogger(ctx)

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ShadowPartitionRequeue"), redis_telemetry.ScopeQueue)

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

	partitionID := sp.Identifier()
	ctx, span := q.ConditionalTracer.NewSpan(ctx, "queue.ShadowPartitionRequeue", partitionID.AccountID, partitionID.EnvID)
	defer span.End()
	span.SetAttributes(attribute.String("partition_id", sp.PartitionID))

	keys := []string{
		q.RedisClient.kg.ShadowPartitionMeta(),
		q.RedisClient.kg.GlobalShadowPartitionSet(),
		q.RedisClient.kg.GlobalAccountShadowPartitions(),
		q.RedisClient.kg.AccountShadowPartitions(accountID),
		q.RedisClient.kg.ShadowPartitionSet(sp.PartitionID),
	}
	args, err := StrSlice([]any{
		sp.PartitionID,
		accountID,
		q.Clock.Now().UnixMilli(),
		requeueAtMS,
	})
	if err != nil {
		return fmt.Errorf("could not serialize args: %w", err)
	}

	status, err := scripts["queue/shadowPartitionRequeue"].Exec(
		redis_telemetry.WithScriptName(ctx, "shadowPartitionRequeue"),
		q.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error returning shadow partition lease: %w", err)
	}

	l.Trace("requeued shadow partition",
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
				"queue_shard": q.name,
				// "partition_id": sp.PartitionID,
				"action": "not_found",
			},
		})

		return osqueue.ErrShadowPartitionNotFound
	default:
		return fmt.Errorf("unknown response returning shadow partition lease: %v (%T)", status, status)
	}
}

func (q *queue) ShadowPartitionLease(ctx context.Context, sp *osqueue.QueueShadowPartition, duration time.Duration) (*ulid.ULID, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ShadowPartitionLease"), redis_telemetry.ScopeQueue)

	now := q.Clock.Now()
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
		q.RedisClient.kg.ShadowPartitionMeta(),
		q.RedisClient.kg.GlobalShadowPartitionSet(),
		q.RedisClient.kg.GlobalAccountShadowPartitions(),
		q.RedisClient.kg.AccountShadowPartitions(accountID),
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
		q.RedisClient.unshardedRc,
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
		return nil, osqueue.ErrShadowPartitionNotFound
	case -2:
		return nil, osqueue.ErrShadowPartitionAlreadyLeased
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
