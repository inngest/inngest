package queue

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
)

type DequeueOptionFn func(o *DequeueOptions)

type DequeueOptions struct {
	DisableConstraintUpdates bool
}

type LeaseOptions struct {
	disableConstraintChecks bool

	backlog     QueueBacklog
	sp          QueueShadowPartition
	constraints PartitionConstraintConfig
}

func LeaseOptionDisableConstraintChecks(disableChecks bool) LeaseOptionFn {
	return func(o *LeaseOptions) {
		o.disableConstraintChecks = disableChecks
	}
}

func LeaseBacklog(b QueueBacklog) LeaseOptionFn {
	return func(o *LeaseOptions) {
		o.backlog = b
	}
}

func LeaseShadowPartition(sp QueueShadowPartition) LeaseOptionFn {
	return func(o *LeaseOptions) {
		o.sp = sp
	}
}

func LeaseConstraints(constraints PartitionConstraintConfig) LeaseOptionFn {
	return func(o *LeaseOptions) {
		o.constraints = constraints
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

type QueueProcessor interface {
	EnqueueItem(ctx context.Context, shard QueueShard, i QueueItem, at time.Time, opts EnqueueOpts) (QueueItem, error)
	Peek(ctx context.Context, partition *QueuePartition, until time.Time, limit int64) ([]*QueueItem, error)
	Lease(ctx context.Context, item QueueItem, leaseDuration time.Duration, now time.Time, denies *LeaseDenies, options ...LeaseOptionFn) (*ulid.ULID, error)
	ExtendLease(ctx context.Context, i QueueItem, leaseID ulid.ULID, duration time.Duration, opts ...ExtendLeaseOptionFn) (*ulid.ULID, error)
	Requeue(ctx context.Context, queueShard QueueShard, i QueueItem, at time.Time, opts ...requeueOptionFn) error
	RequeueByJobID(ctx context.Context, queueShard QueueShard, jobID string, at time.Time) error
	Dequeue(ctx context.Context, queueShard QueueShard, i QueueItem, opts ...DequeueOptionFn) error

	PartitionPeek(ctx context.Context, sequential bool, until time.Time, limit int64) ([]*QueuePartition, error)
	PartitionLease(ctx context.Context, p *QueuePartition, duration time.Duration, opts ...partitionLeaseOpt) (*ulid.ULID, int, error)
	PartitionRequeue(ctx context.Context, shard QueueShard, p *QueuePartition, at time.Time, forceAt bool) error
}
