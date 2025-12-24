package queue

import (
	"context"
	"iter"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

type DequeueOptionFn func(o *DequeueOptions)

type DequeueOptions struct {
	DisableConstraintUpdates bool
}

func DequeueOptionDisableConstraintUpdates(disableUpdates bool) DequeueOptionFn {
	return func(o *DequeueOptions) {
		o.DisableConstraintUpdates = disableUpdates
	}
}

type RequeueOptions struct {
	DisableConstraintUpdates bool
}

func RequeueOptionDisableConstraintUpdates(disableUpdates bool) RequeueOptionFn {
	return func(o *RequeueOptions) {
		o.DisableConstraintUpdates = disableUpdates
	}
}

type RequeueOptionFn func(o *RequeueOptions)

type LeaseOptions struct {
	DisableConstraintChecks bool

	Backlog         QueueBacklog
	ShadowPartition QueueShadowPartition
	Constraints     PartitionConstraintConfig
}

func LeaseOptionDisableConstraintChecks(disableChecks bool) LeaseOptionFn {
	return func(o *LeaseOptions) {
		o.DisableConstraintChecks = disableChecks
	}
}

func LeaseBacklog(b QueueBacklog) LeaseOptionFn {
	return func(o *LeaseOptions) {
		o.Backlog = b
	}
}

func LeaseShadowPartition(sp QueueShadowPartition) LeaseOptionFn {
	return func(o *LeaseOptions) {
		o.ShadowPartition = sp
	}
}

func LeaseConstraints(constraints PartitionConstraintConfig) LeaseOptionFn {
	return func(o *LeaseOptions) {
		o.Constraints = constraints
	}
}

type LeaseOptionFn func(o *LeaseOptions)

type ExtendLeaseOptions struct {
	DisableConstraintUpdates bool
}

func ExtendLeaseOptionDisableConstraintUpdates(disableUpdates bool) ExtendLeaseOptionFn {
	return func(o *ExtendLeaseOptions) {
		o.DisableConstraintUpdates = disableUpdates
	}
}

type ExtendLeaseOptionFn func(o *ExtendLeaseOptions)

type PartitionLeaseOptions struct {
	DisableLeaseChecks bool
}

type PartitionLeaseOpt func(o *PartitionLeaseOptions)

func PartitionLeaseOptionDisableLeaseChecks(disableLeaseChecks bool) PartitionLeaseOpt {
	return func(o *PartitionLeaseOptions) {
		o.DisableLeaseChecks = disableLeaseChecks
	}
}

type QueueManager interface {
	JobQueueReader
	Queue
	QueueDirectAccess

	DequeueByJobID(ctx context.Context, jobID string) error
	Dequeue(ctx context.Context, queueShard QueueShard, i QueueItem, opts ...DequeueOptionFn) error
	Requeue(ctx context.Context, queueShard QueueShard, i QueueItem, at time.Time, opts ...RequeueOptionFn) error
	RequeueByJobID(ctx context.Context, queueShard QueueShard, jobID string, at time.Time) error

	// ResetAttemptsByJobID sets retries to zero given a single job ID.  This is important for
	// checkpointing;  a single job becomes shared amongst many  steps.
	ResetAttemptsByJobID(ctx context.Context, shard string, jobID string) error

	// ItemsByPartition returns a queue item iterator for a function within a specific time range
	ItemsByPartition(ctx context.Context, queueShard QueueShard, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error)
	// ItemsByBacklog returns a queue item iterator for a backlog within a specific time range
	ItemsByBacklog(ctx context.Context, queueShard QueueShard, backlogID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error)
	// BacklogsByPartition returns an iterator for the partition's backlogs
	BacklogsByPartition(ctx context.Context, queueShard QueueShard, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueBacklog], error)
	// BacklogSize retrieves the number of items in the specified backlog
	BacklogSize(ctx context.Context, queueShard QueueShard, backlogID string) (int64, error)
	// PartitionByID retrieves the partition by the partition ID
	PartitionByID(ctx context.Context, queueShard QueueShard, partitionID string) (*PartitionInspectionResult, error)
	// ItemByID retrieves the queue item by the jobID
	ItemByID(ctx context.Context, jobID string) (*QueueItem, error)
	// ItemExists checks if an item with jobID exists in the queue
	ItemExists(ctx context.Context, jobID string) (bool, error)
	// ItemsByRunID retrieves all queue items via runID
	//
	// NOTE
	// The queue technically shouldn't know about runIDs, so we should make this more generic with certain type of indices in the future
	ItemsByRunID(ctx context.Context, runID ulid.ULID) ([]*QueueItem, error)

	// PartitionBacklogSize returns the point in time backlog size of the partition.
	// This will sum the size of all backlogs in that partition
	PartitionBacklogSize(ctx context.Context, partitionID string) (int64, error)

	// Total queue depth of all partitions including backlog and ready state items
	TotalSystemQueueDepth(ctx context.Context) (int64, error)
}

type QueueProcessor interface {
	EnqueueItem(ctx context.Context, i QueueItem, at time.Time, opts EnqueueOpts) (QueueItem, error)
	Peek(ctx context.Context, partition *QueuePartition, until time.Time, limit int64) ([]*QueueItem, error)
	Lease(ctx context.Context, item QueueItem, leaseDuration time.Duration, now time.Time, denies *LeaseDenies, options ...LeaseOptionFn) (*ulid.ULID, error)
	ExtendLease(ctx context.Context, i QueueItem, leaseID ulid.ULID, duration time.Duration, opts ...ExtendLeaseOptionFn) (*ulid.ULID, error)
	Requeue(ctx context.Context, i QueueItem, at time.Time, opts ...RequeueOptionFn) error
	RequeueByJobID(ctx context.Context, jobID string, at time.Time) error
	Dequeue(ctx context.Context, i QueueItem, opts ...DequeueOptionFn) error

	PartitionPeek(ctx context.Context, sequential bool, until time.Time, limit int64) ([]*QueuePartition, error)
	PartitionLease(ctx context.Context, p *QueuePartition, duration time.Duration, opts ...PartitionLeaseOpt) (*ulid.ULID, int, error)
	PartitionRequeue(ctx context.Context, p *QueuePartition, at time.Time, forceAt bool) error

	Scavenge(ctx context.Context, limit int) (int, error)
	ActiveCheck(ctx context.Context) (int, error)
	Instrument(ctx context.Context) error

	ItemsByPartition(ctx context.Context, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error)

	SetFunctionMigrate(ctx context.Context, fnID uuid.UUID, migrateLockUntil *time.Time) error
	ResetAttemptsByJobID(ctx context.Context, jobID string) error

	PeekEWMA(ctx context.Context, fnID uuid.UUID) (int64, error)
	SetPeekEWMA(ctx context.Context, fnID *uuid.UUID, val int64) error
	PartitionSize(ctx context.Context, partitionID string, until time.Time) (int64, error)

	ConfigLease(ctx context.Context, key string, duration time.Duration, existingLeaseID ...*ulid.ULID) (*ulid.ULID, error)

	AccountPeek(ctx context.Context, sequential bool, until time.Time, limit int64) ([]uuid.UUID, error)

	PeekAccountPartitions(
		ctx context.Context,
		accountID uuid.UUID,
		peekLimit int64,
		peekUntil time.Time,
		sequential bool,
	) ([]*QueuePartition, error)

	PeekGlobalPartitions(
		ctx context.Context,
		peekLimit int64,
		peekUntil time.Time,
		sequential bool,
	) ([]*QueuePartition, error)

	BacklogRefillConstraintCheck(
		ctx context.Context,
		shadowPart *QueueShadowPartition,
		backlog *QueueBacklog,
		constraints PartitionConstraintConfig,
		items []*QueueItem,
		operationIdempotencyKey string,
		now time.Time,
	) (*BacklogRefillConstraintCheckResult, error)

	ItemLeaseConstraintCheck(
		ctx context.Context,
		shadowPart *QueueShadowPartition,
		backlog *QueueBacklog,
		constraints PartitionConstraintConfig,
		item *QueueItem,
		now time.Time,
	) (ItemLeaseConstraintCheckResult, error)

	RemoveQueueItem(ctx context.Context, partitionID string, itemID string) error
	LoadQueueItem(ctx context.Context, itemID string) (*QueueItem, error)

	LeaseBacklogForNormalization(ctx context.Context, bl *QueueBacklog) error
	ExtendBacklogNormalizationLease(ctx context.Context, now time.Time, bl *QueueBacklog) error
	ShadowPartitionPeekNormalizeBacklogs(ctx context.Context, sp *QueueShadowPartition, limit int64) ([]*QueueBacklog, error)
	BacklogNormalizePeek(ctx context.Context, b *QueueBacklog, limit int64) (*PeekResult[QueueItem], error)
}

type BacklogRefillOptions struct {
	ConstraintCheckIdempotencyKey string
	DisableConstraintChecks       bool
	CapacityLeases                []CapacityLease
}

type BacklogRefillOptionFn func(o *BacklogRefillOptions)

func WithBacklogRefillConstraintCheckIdempotencyKey(idempotencyKey string) BacklogRefillOptionFn {
	return func(o *BacklogRefillOptions) {
		o.ConstraintCheckIdempotencyKey = idempotencyKey
	}
}

func WithBacklogRefillDisableConstraintChecks(disableConstraintChecks bool) BacklogRefillOptionFn {
	return func(o *BacklogRefillOptions) {
		o.DisableConstraintChecks = disableConstraintChecks
	}
}

func WithBacklogRefillItemCapacityLeases(itemCapacityLeases []CapacityLease) BacklogRefillOptionFn {
	return func(o *BacklogRefillOptions) {
		o.CapacityLeases = itemCapacityLeases
	}
}
