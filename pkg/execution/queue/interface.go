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

type extendLeaseOptions struct {
	disableConstraintUpdates bool
}

func ExtendLeaseOptionDisableConstraintUpdates(disableUpdates bool) ExtendLeaseOptionFn {
	return func(o *extendLeaseOptions) {
		o.disableConstraintUpdates = disableUpdates
	}
}

type ExtendLeaseOptionFn func(o *extendLeaseOptions)

type PartitionLeaseOptions struct {
	DisableLeaseChecks bool
}

type PartitionLeaseOpt func(o *PartitionLeaseOptions)

func PartitionLeaseOptionDisableLeaseChecks(disableLeaseChecks bool) PartitionLeaseOpt {
	return func(o *PartitionLeaseOptions) {
		o.DisableLeaseChecks = disableLeaseChecks
	}
}

type QueueProcessor interface {
	Options() QueueOptions

	EnqueueItem(ctx context.Context, shard QueueShard, i QueueItem, at time.Time, opts EnqueueOpts) (QueueItem, error)
	Peek(ctx context.Context, partition *QueuePartition, until time.Time, limit int64) ([]*QueueItem, error)
	Lease(ctx context.Context, item QueueItem, leaseDuration time.Duration, now time.Time, denies *LeaseDenies, options ...LeaseOptionFn) (*ulid.ULID, error)
	ExtendLease(ctx context.Context, i QueueItem, leaseID ulid.ULID, duration time.Duration, opts ...ExtendLeaseOptionFn) (*ulid.ULID, error)
	Requeue(ctx context.Context, queueShard QueueShard, i QueueItem, at time.Time, opts ...RequeueOptionFn) error
	RequeueByJobID(ctx context.Context, queueShard QueueShard, jobID string, at time.Time) error
	Dequeue(ctx context.Context, queueShard QueueShard, i QueueItem, opts ...DequeueOptionFn) error

	PartitionPeek(ctx context.Context, sequential bool, until time.Time, limit int64) ([]*QueuePartition, error)
	PartitionLease(ctx context.Context, p *QueuePartition, duration time.Duration, opts ...PartitionLeaseOpt) (*ulid.ULID, int, error)
	PartitionRequeue(ctx context.Context, shard QueueShard, p *QueuePartition, at time.Time, forceAt bool) error

	ItemsByPartition(ctx context.Context, shard QueueShard, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error)

	SetFunctionMigrate(ctx context.Context, shard QueueShard, fnID uuid.UUID, migrateLockUntil *time.Time)
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
