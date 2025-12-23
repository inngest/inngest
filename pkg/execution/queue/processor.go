package queue

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

type QueueShard interface {
	Name() string
	Kind() enums.QueueShardKind
}

type DequeueOptionFn func(o *DequeueOptions)

type DequeueOptions struct {
	DisableConstraintUpdates bool
}

type QueueProcessor interface {
	EnqueueItem(ctx context.Context, shard QueueShard, i QueueItem, at time.Time, opts EnqueueOpts) (QueueItem, error)
	Peek(ctx context.Context, partition *QueuePartition, until time.Time, limit int64) ([]*osqueue.QueueItem, error)
	Lease(ctx context.Context, item QueueItem, leaseDuration time.Duration, now time.Time, denies *leaseDenies, options ...leaseOptionFn) (*ulid.ULID, error)
	ExtendLease(ctx context.Context, i QueueItem, leaseID ulid.ULID, duration time.Duration, opts ...extendLeaseOptionFn) (*ulid.ULID, error)
	Requeue(ctx context.Context, queueShard QueueShard, i QueueItem, at time.Time, opts ...requeueOptionFn) error
	RequeueByJobID(ctx context.Context, queueShard QueueShard, jobID string, at time.Time) error
	Dequeue(ctx context.Context, queueShard QueueShard, i QueueItem, opts ...DequeueOptionFn) error

	PartitionPeek(ctx context.Context, sequential bool, until time.Time, limit int64) ([]*QueuePartition, error)
	PartitionLease(ctx context.Context, p *QueuePartition, duration time.Duration, opts ...partitionLeaseOpt) (*ulid.ULID, int, error)
	PartitionRequeue(ctx context.Context, shard QueueShard, p *QueuePartition, at time.Time, forceAt bool) error
}

// QueuePartition represents an individual queue for a workflow.  It stores the
// time of the earliest job within the workflow.
type QueuePartition struct {
	// ID represents the key used within the global Partition hash and global pointer set
	// which represents this QueuePartition.  This is the function ID for enums.PartitionTypeDefault,
	// or the entire key returned from the key generator for other types.
	ID string `json:"id,omitempty"`
	// QueueName is used for manually overriding queue items to be enqueued for
	// system jobs like pause events and timeouts, batch timeouts, and replays.
	//
	// NOTE: This field is required for backwards compatibility, as old system partitions
	// simply set the queue name.
	//
	// This should almost always be nil.
	QueueName *string `json:"queue,omitempty"`
	// FunctionID represents the function ID that this partition manages.
	// NOTE:  If this partition represents many fns (eg. acct or env), this may be nil
	FunctionID *uuid.UUID `json:"wid,omitempty"`
	// EnvID represents the environment ID for the partition, either from the
	// function ID or the environment scope itself.
	EnvID *uuid.UUID `json:"wsID,omitempty"`
	// AccountID represents the account ID for the partition
	AccountID uuid.UUID `json:"aID,omitempty"`
	// LeaseID represents a lease on this partition.  If the LeaseID is not nil,
	// this partition can be claimed by a shared-nothing worker to work on the
	// queue items within this partition.
	//
	// A lease is shortly held (eg seconds).  It should last long enough for
	// workers to claim QueueItems only.
	LeaseID *ulid.ULID `json:"leaseID,omitempty"`
	// Last represents the time that this partition was last leased, as a millisecond
	// unix epoch.  In essence, we need this to track how frequently we're leasing and
	// attempting to run items in the partition's queue.
	// Without this, we cannot track sojourn latency.
	Last int64 `json:"last"`
	// ForcedAtMS records the time that the partition is forced to, in milliseconds, if
	// the partition has been forced into the future via concurrency issues. This means
	// that it was requeued due to concurrency issues and should not be brought forward
	// when a new step is enqueued, if now < ForcedAtMS.
	ForceAtMS int64 `json:"forceAtMS"`
}
