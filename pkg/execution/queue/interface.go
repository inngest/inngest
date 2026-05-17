package queue

import (
	"context"
	"iter"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
)

type DequeueOptionFn func(o *DequeueOptions)

type DequeueOptions struct{}

type RequeueOptions struct{}

type RequeueOptionFn func(o *RequeueOptions)

type LeaseOptions struct {
	Backlog         QueueBacklog
	ShadowPartition QueueShadowPartition
	Constraints     PartitionConstraintConfig
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

type ExtendLeaseOptions struct{}

type ExtendLeaseOptionFn func(o *ExtendLeaseOptions)

type PartitionLeaseOptions struct{}

type PartitionLeaseOpt func(o *PartitionLeaseOptions)

type QueueManager interface {
	JobQueueReader
	Queue

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
	// BacklogByID retrieves a single backlog by its ID
	BacklogByID(ctx context.Context, queueShard QueueShard, backlogID string) (*QueueBacklog, error)
	// PartitionByID retrieves the partition by the partition ID
	PartitionByID(ctx context.Context, queueShard QueueShard, partitionID string) (*PartitionInspectionResult, error)
	// LoadQueueItem retrieves the queue item by the item ID.
	LoadQueueItem(ctx context.Context, shardName string, itemID string) (*QueueItem, error)

	// ItemExists checks if an item with jobID exists in the queue
	ItemExists(ctx context.Context, queueShard QueueShard, jobID string) (bool, error)
	// ItemsByRunID retrieves all queue items via runID
	//
	// NOTE
	// The queue technically shouldn't know about runIDs, so we should make this more generic with certain type of indices in the future
	ItemsByRunID(ctx context.Context, queueShard QueueShard, runID ulid.ULID) ([]*QueueItem, error)

	// PartitionBacklogSize returns the point in time backlog size of the partition.
	// This will sum the size of all backlogs in that partition
	PartitionBacklogSize(ctx context.Context, partitionID string) (int64, error)

	// Total queue depth of all partitions including backlog and ready state items
	TotalSystemQueueDepth(ctx context.Context, queueShard QueueShard) (int64, error)

	NormalizeBacklog(ctx context.Context, backlog *QueueBacklog, sp *QueueShadowPartition, latestConstraints PartitionConstraintConfig) error
	NormalizeItem(
		ctx context.Context,
		sp *QueueShadowPartition,
		latestConstraints PartitionConstraintConfig,
		sourceBacklog *QueueBacklog,
		item QueueItem,
	) (QueueItem, error)

	ProcessItem(
		ctx context.Context,
		i ProcessItem,
		f RunFunc,
	) error
	ProcessPartition(ctx context.Context, p *QueuePartition, continuationCount uint, randomOffset bool) error

	ScanShadowPartitions(ctx context.Context, until time.Time, qspc chan ShadowPartitionChanMsg) error
	ProcessShadowPartition(ctx context.Context, shadowPart *QueueShadowPartition, continuationCount uint) error

	ProcessShadowPartitionBacklog(
		ctx context.Context,
		shadowPart *QueueShadowPartition,
		backlog *QueueBacklog,
		refillUntil time.Time,
		constraints PartitionConstraintConfig,
	) (*BacklogRefillResult, enums.QueueConstraint, error)
}

type QueueProcessor interface {
	Shard() QueueShard
	Clock() clockwork.Clock
	Semaphore() util.TrackingSemaphore
	Options() *QueueOptions
	Workers() chan ProcessItem
	SequentialLease() *ulid.ULID

	ShadowPartitionWorkers() chan ShadowPartitionChanMsg
	AddShadowContinue(ctx context.Context, p *QueueShadowPartition, ctr uint)
	GetShadowContinuations() map[string]ShadowContinuation
	ClearShadowContinuations()

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
}

// Scope identifies the tenant/function namespace for queue-owned state.
type Scope struct {
	IsSystem   bool
	AccountID  uuid.UUID
	EnvID      uuid.UUID
	FunctionID uuid.UUID
}

// SingletonOperations is the per-shard surface for singleton lock state.
// Construct singleton clients against a shard resolved through ShardRegistry.
type SingletonOperations interface {
	// SingletonGetRunID returns the run ID currently holding the singleton
	// lock for key, or nil if no lock is held.
	SingletonGetRunID(ctx context.Context, scope Scope, key string) (*ulid.ULID, error)
	// SingletonReleaseRunID atomically gets and deletes the singleton lock
	// for key, returning the released run ID or nil if no lock was held.
	SingletonReleaseRunID(ctx context.Context, scope Scope, key string) (*ulid.ULID, error)
}

// DebounceUpdateStatus describes the outcome of DebounceUpdate.
type DebounceUpdateStatus int

const (
	// DebounceUpdateOK indicates the debounce was updated; the returned TTL
	// (in seconds) is the new lifetime to requeue against.
	DebounceUpdateOK DebounceUpdateStatus = iota
	// DebounceUpdateInProgress indicates the debounce has begun executing
	// or is just about to; the caller should retry.
	DebounceUpdateInProgress
	// DebounceUpdateOutOfOrder indicates a newer event has already updated
	// the debounce; the caller should drop the update.
	DebounceUpdateOutOfOrder
	// DebounceUpdateNotFound indicates the timeout queue item is missing;
	// the caller should enqueue a fresh timeout job.
	DebounceUpdateNotFound
)

// DebounceStartStatus describes the outcome of DebounceStartExecution.
type DebounceStartStatus int

const (
	// DebounceStartStarted indicates execution started successfully.
	DebounceStartStarted DebounceStartStatus = iota
	// DebounceStartMigrating indicates a concurrent migration disabled
	// execution on this shard; the caller must abort.
	DebounceStartMigrating
)

// DebounceOperations is the per-shard surface for debounce state. Each
// debounce lives on a single shard alongside its timeout queue item; route
// to the right shard via ShardRegistry before invoking these.
type DebounceOperations interface {
	// DebounceCreate atomically creates a new debounce for scope/key. If a
	// debounce already exists, it returns the existing debounce ID and
	// no error.
	DebounceCreate(ctx context.Context, scope Scope, key string, debounceID ulid.ULID, item []byte, ttl time.Duration) (existingID *ulid.ULID, err error)

	// DebounceUpdate atomically updates the currently pending debounce.
	// On status DebounceUpdateOK, ttlSeconds is the new TTL to requeue
	// against. Other statuses describe special outcomes; ttlSeconds is
	// undefined for those.
	DebounceUpdate(ctx context.Context, scope Scope, key string, debounceID ulid.ULID, item []byte, ttl time.Duration, jobID string, now time.Time, eventTimestamp int64) (ttlSeconds int64, status DebounceUpdateStatus, err error)

	// DebounceStartExecution atomically begins execution of a debounce,
	// rotating the pointer to newDebounceID.
	DebounceStartExecution(ctx context.Context, scope Scope, key string, newDebounceID, debounceID ulid.ULID) (DebounceStartStatus, error)

	// DebouncePrepareMigration atomically replaces the debounce pointer
	// with fakeDebounceID to disable execution on this shard, returning
	// the existing debounce ID and timeout (millis) so the caller can
	// re-create the debounce on another shard. Returns (nil, 0, nil)
	// when no debounce exists.
	DebouncePrepareMigration(ctx context.Context, scope Scope, key string, fakeDebounceID ulid.ULID) (existingID *ulid.ULID, timeoutMillis int64, err error)

	// DebounceGetItem retrieves the serialized debounce item from the
	// hash. Returns ErrDebounceNotFound when absent.
	DebounceGetItem(ctx context.Context, scope Scope, debounceID ulid.ULID) ([]byte, error)

	// DebounceDeleteItems removes one or more debounce items from the
	// hash. A no-op on an empty list.
	DebounceDeleteItems(ctx context.Context, scope Scope, debounceIDs ...ulid.ULID) error

	// DebounceDeleteMigratingFlag clears the in-progress migration flag
	// for debounceID.
	DebounceDeleteMigratingFlag(ctx context.Context, scope Scope, debounceID ulid.ULID) error

	// DebounceGetPointer reads the current debounce ID for scope/key.
	// Returns ErrDebounceNotFound when no debounce is active.
	DebounceGetPointer(ctx context.Context, scope Scope, key string) (string, error)

	// DebounceDeletePointer removes the pointer for scope/key.
	DebounceDeletePointer(ctx context.Context, scope Scope, key string) error
}

// PeekOperations is the per-shard surface for queue peeking.
type PeekOperations interface {
	Peek(ctx context.Context, partition *QueuePartition, until time.Time, limit int64) ([]*QueueItem, error)
	PeekRandom(ctx context.Context, partition *QueuePartition, until time.Time, limit int64) ([]*QueueItem, error)
	PartitionPeek(ctx context.Context, sequential bool, until time.Time, limit int64) ([]*QueuePartition, error)
	PeekEWMA(ctx context.Context, fnID uuid.UUID) (int64, error)
	SetPeekEWMA(ctx context.Context, fnID *uuid.UUID, val int64) error
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
}

// ShadowProcessingOperations is the per-shard surface for shadow partition,
// backlog, and backlog normalization processing.
type ShadowProcessingOperations interface {
	LeaseBacklogForNormalization(ctx context.Context, bl *QueueBacklog) error
	ExtendBacklogNormalizationLease(ctx context.Context, now time.Time, bl *QueueBacklog) error

	ShadowPartitionRequeue(ctx context.Context, sp *QueueShadowPartition, requeueAt *time.Time) error
	ShadowPartitionLease(ctx context.Context, sp *QueueShadowPartition, duration time.Duration) (*ulid.ULID, error)
	ShadowPartitionExtendLease(ctx context.Context, sp *QueueShadowPartition, leaseID ulid.ULID, duration time.Duration) (*ulid.ULID, error)

	BacklogPrepareNormalize(ctx context.Context, b *QueueBacklog, sp *QueueShadowPartition) error
	BacklogRefill(
		ctx context.Context,
		b *QueueBacklog,
		sp *QueueShadowPartition,
		refillUntil time.Time,
		refillItems []string,
		options ...BacklogRefillOptionFn,
	) (*BacklogRefillResult, error)
	BacklogRequeue(ctx context.Context, backlog *QueueBacklog, sp *QueueShadowPartition, requeueAt time.Time) error
	BacklogsByPartition(ctx context.Context, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueBacklog], error)
	BacklogSize(ctx context.Context, backlogID string) (int64, error)
	BacklogByID(ctx context.Context, backlogID string) (*QueueBacklog, error)

	ItemsByBacklog(ctx context.Context, backlogID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error)

	ShadowPartitionPeekNormalizeBacklogs(ctx context.Context, sp *QueueShadowPartition, limit int64) ([]*QueueBacklog, error)
	BacklogNormalizePeek(ctx context.Context, b *QueueBacklog, limit int64) (*PeekResult[QueueItem], error)
	PeekGlobalNormalizeAccounts(ctx context.Context, until time.Time, limit int64) ([]uuid.UUID, error)
	PeekGlobalShadowPartitionAccounts(ctx context.Context, sequential bool, until time.Time, limit int64) ([]uuid.UUID, error)
	ShadowPartitionPeek(ctx context.Context, sp *QueueShadowPartition, sequential bool, until time.Time, limit int64, opts ...PeekOpt) ([]*QueueBacklog, int, error)
	PeekShadowPartitions(ctx context.Context, accountID *uuid.UUID, sequential bool, peekLimit int64, until time.Time) ([]*QueueShadowPartition, error)
	BacklogPeek(ctx context.Context, b *QueueBacklog, from time.Time, until time.Time, limit int64, opts ...PeekOpt) (*BacklogPeekResult, error)
}

// InsightsOperations is the per-shard surface for queue inspection and metrics.
type InsightsOperations interface {
	Instrument(ctx context.Context) error
	ItemsByPartition(ctx context.Context, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error)

	// Total queue depth of all partitions including backlog and ready state items
	TotalSystemQueueDepth(ctx context.Context) (int64, error)

	PartitionBacklogSize(ctx context.Context, partitionID string) (int64, error)
	OutstandingJobCount(ctx context.Context, workspaceID, workflowID uuid.UUID, runID ulid.ULID) (int, error)
	RunningCount(ctx context.Context, functionID uuid.UUID) (int64, error)
	StatusCount(ctx context.Context, workflowID uuid.UUID, status string) (int64, error)
}

type ShardOperations interface {
	SingletonOperations
	DebounceOperations
	PeekOperations
	ShadowProcessingOperations
	InsightsOperations

	EnqueueItem(ctx context.Context, i QueueItem, at time.Time, opts EnqueueOpts) (QueueItem, error)

	Lease(ctx context.Context, item QueueItem, leaseDuration time.Duration, now time.Time, options ...LeaseOptionFn) (*ulid.ULID, error)
	ExtendLease(ctx context.Context, i QueueItem, leaseID ulid.ULID, duration time.Duration, opts ...ExtendLeaseOptionFn) (*ulid.ULID, error)

	Requeue(ctx context.Context, i QueueItem, at time.Time, opts ...RequeueOptionFn) error
	RequeueByJobID(ctx context.Context, jobID string, at time.Time) error

	Dequeue(ctx context.Context, i QueueItem, opts ...DequeueOptionFn) error
	DequeueByJobID(ctx context.Context, jobID string) error

	PartitionLease(ctx context.Context, p *QueuePartition, duration time.Duration, opts ...PartitionLeaseOpt) (*ulid.ULID, error)
	PartitionRequeue(ctx context.Context, p *QueuePartition, at time.Time, forceAt bool) error

	Scavenge(ctx context.Context, limit int) (int, error)

	RemoveQueueItem(ctx context.Context, partitionID string, itemID string) error

	SetFunctionMigrate(ctx context.Context, fnID uuid.UUID, migrateLockUntil *time.Time) error
	ResetAttemptsByJobID(ctx context.Context, jobID string) error

	PartitionSize(ctx context.Context, partitionID string, until time.Time) (int64, error)

	ConfigLease(ctx context.Context, key string, duration time.Duration, existingLeaseID ...*ulid.ULID) (*ulid.ULID, error)
	ShardLease(ctx context.Context, key string, duration time.Duration, maxLeases int, existingLeaseID ...*ulid.ULID) (*ulid.ULID, error)
	ReleaseShardLease(ctx context.Context, key string, existingLeaseID ulid.ULID) error

	LoadQueueItem(ctx context.Context, itemID string) (*QueueItem, error)

	IsMigrationLocked(ctx context.Context, fnID uuid.UUID) (*time.Time, error)

	ItemExists(ctx context.Context, jobID string) (bool, error)
	ItemsByRunID(ctx context.Context, runID ulid.ULID) ([]*QueueItem, error)
	PartitionByID(ctx context.Context, partitionID string) (*PartitionInspectionResult, error)

	UnpauseFunction(ctx context.Context, acctID, envID, fnID uuid.UUID) error

	RunJobs(ctx context.Context, workspaceID, workflowID uuid.UUID, runID ulid.ULID, limit, offset int64) ([]JobResponse, error)
}

type BacklogRefillOptions struct {
	CapacityLeases []CapacityLease
}

type BacklogRefillOptionFn func(o *BacklogRefillOptions)

func WithBacklogRefillItemCapacityLeases(itemCapacityLeases []CapacityLease) BacklogRefillOptionFn {
	return func(o *BacklogRefillOptions) {
		o.CapacityLeases = itemCapacityLeases
	}
}
