package redis_state

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
)

// TODO Set this properly
var switchover = time.Date(2024, 07, 10, 0, 0, 0, 0, time.UTC)

type ShardedKeyGenerator interface {

	// Idempotency stores the idempotency key for atomic lookup.
	Idempotency(context.Context, state.Identifier) string

	// RunMetadata stores state regarding the current run identifier, such
	// as the workflow version, the time the run started, etc.
	RunMetadata(ctx context.Context, runID ulid.ULID) string

	// Event returns the key used to store the specific event for the
	// given workflow run.
	Event(context.Context, state.Identifier) string

	// Events returns the key used to store the specific batch for the
	// given workflow run.
	Events(context.Context, state.Identifier) string

	// Actions returns the key used to store the action response map used
	// for given workflow run - ie. the results for individual steps.
	Actions(context.Context, state.Identifier) string

	// Errors returns the key used to store the error hash map used
	// for given workflow run.
	Errors(context.Context, state.Identifier) string

	// PauseStep returns the prefix of the key used within PauseStep.  This lets us
	// iterate through all pauses for a given identifier
	PauseStepPrefix(context.Context, state.Identifier) string

	// PauseStep returns the key used to store a pause ID by the run ID and step ID.
	PauseStep(context.Context, state.Identifier, string) string

	// RunPauses stores pause IDs for each run as a zset
	RunPauses(ctx context.Context, runID ulid.ULID) string

	// History returns the key used to store a log entry for run hisotry
	History(ctx context.Context, runID ulid.ULID) string

	// Stack returns the key used to store the stack for a given run
	Stack(ctx context.Context, runID ulid.ULID) string
}

type shardedKeyGenerator struct {
}

func (s shardedKeyGenerator) Prefix(defaultPrefix string, runID ulid.ULID) string {
	if ulid.Time(runID.Time()).After(switchover) {
		return fmt.Sprintf("{%s}", runID)
	}
	return defaultPrefix
}

const stateDefaultKey = "{state}"
const queueDefaultKey = "{queue}"

func (s shardedKeyGenerator) Idempotency(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("%s:key:%s", s.Prefix(stateDefaultKey, id.RunID), id.IdempotencyKey())
}

func (s shardedKeyGenerator) RunMetadata(ctx context.Context, runID ulid.ULID) string {
	return fmt.Sprintf("%s:metadata:%s", s.Prefix(stateDefaultKey, runID), runID)
}

func (s shardedKeyGenerator) Event(ctx context.Context, identifier state.Identifier) string {
	return fmt.Sprintf("%s:events:%s:%s", s.Prefix(stateDefaultKey, identifier.RunID), identifier.WorkflowID, identifier.RunID)
}

func (s shardedKeyGenerator) Events(ctx context.Context, identifier state.Identifier) string {
	return fmt.Sprintf("%s:bulk-events:%s:%s", s.Prefix(stateDefaultKey, identifier.RunID), identifier.WorkflowID, identifier.RunID)
}

func (s shardedKeyGenerator) Actions(ctx context.Context, identifier state.Identifier) string {
	return fmt.Sprintf("%s:actions:%s:%s", s.Prefix(stateDefaultKey, identifier.RunID), identifier.WorkflowID, identifier.RunID)
}

func (s shardedKeyGenerator) Errors(ctx context.Context, identifier state.Identifier) string {
	return fmt.Sprintf("%s:errors:%s:%s", s.Prefix(stateDefaultKey, identifier.RunID), identifier.WorkflowID, identifier.RunID)
}

func (s shardedKeyGenerator) PauseStepPrefix(ctx context.Context, identifier state.Identifier) string {
	return fmt.Sprintf("%s:pause-steps:%s", s.Prefix(stateDefaultKey, identifier.RunID), identifier.RunID)
}

func (s shardedKeyGenerator) PauseStep(ctx context.Context, identifier state.Identifier, s2 string) string {
	prefix := s.PauseStepPrefix(ctx, identifier)
	return fmt.Sprintf("%s-%s", prefix, s2)
}

func (s shardedKeyGenerator) RunPauses(ctx context.Context, runID ulid.ULID) string {
	return fmt.Sprintf("%s:pr:%s", s.Prefix(stateDefaultKey, runID), runID)
}

func (s shardedKeyGenerator) History(ctx context.Context, runID ulid.ULID) string {
	return fmt.Sprintf("%s:history:%s", s.Prefix(stateDefaultKey, runID), runID)
}

func (s shardedKeyGenerator) Stack(ctx context.Context, runID ulid.ULID) string {
	return fmt.Sprintf("%s:stack:%s", s.Prefix(stateDefaultKey, runID), runID)
}

func newShardedKeyGenerator() ShardedKeyGenerator {
	return &shardedKeyGenerator{}
}

type UnshardedKeyGenerator interface {
	QueueKeyGenerator
	DebounceKeyGenerator
	BatchKeyGenerator

	// Workflow returns the key for the current workflow ID and version.
	Workflow(ctx context.Context, workflowID uuid.UUID, version int) string

	// PauseLease stores the key which references a pause's lease.
	//
	// This is stored independently as we may store more than one copy of a pause
	// for easy iteration.
	PauseLease(context.Context, uuid.UUID) string

	// PauseID returns the key used to store an individual pause from its ID.
	PauseID(context.Context, uuid.UUID) string

	// PauseEvent returns the key used to store data for loading pauses by events.
	PauseEvent(context.Context, uuid.UUID, string) string

	// PauseIndex is a key that's used to index added/expired times for pauses.
	//
	// Added times are necessary to load pauses after a specific point in time,
	// which is used when caching pauses in-memory to only load the subset of pauses
	// added after the cache was last updated.
	PauseIndex(ctx context.Context, kind string, wsID uuid.UUID, event string) string

	// Invoke returns the key used to store the correlation key associated with invoke functions
	Invoke(ctx context.Context, wsID uuid.UUID) string
}

type unshardedKeyGenerator struct {
}

func (u unshardedKeyGenerator) Workflow(ctx context.Context, workflowID uuid.UUID, version int) string {
	return fmt.Sprintf("{%s}:workflows:%s-%d", stateDefaultKey, workflowID, version)
}

func (u unshardedKeyGenerator) PauseLease(ctx context.Context, id uuid.UUID) string {
	return fmt.Sprintf("{%s}:pause-lease:%s", stateDefaultKey, id.String())
}

func (u unshardedKeyGenerator) PauseID(ctx context.Context, id uuid.UUID) string {
	return fmt.Sprintf("{%s}:pauses:%s", stateDefaultKey, id.String())
}

func (u unshardedKeyGenerator) PauseEvent(ctx context.Context, id uuid.UUID, s string) string {
	return fmt.Sprintf("{%s}:pause-events:%s:%s", stateDefaultKey, id, s)
}

func (u unshardedKeyGenerator) PauseIndex(ctx context.Context, kind string, wsID uuid.UUID, event string) string {
	if event == "" {
		return fmt.Sprintf("{%s}:pause-idx:%s:%s:-", stateDefaultKey, kind, wsID)
	}
	return fmt.Sprintf("{%s}:pause-idx:%s:%s:%s", stateDefaultKey, kind, wsID, event)
}

func (u unshardedKeyGenerator) Invoke(ctx context.Context, wsID uuid.UUID) string {
	return fmt.Sprintf("{%s}:invoke:%s", stateDefaultKey, wsID)
}

func newUnshardedKeyGenerator() UnshardedKeyGenerator {
	return &unshardedKeyGenerator{}
}

type QueueKeyGenerator interface {
	// QueueItem returns the key for the hash containing all items within a
	// queue for a function.
	QueueItem() string
	// QueueIndex returns the key containing the sorted zset for a function
	// queue.
	QueueIndex(id string) string

	//
	// Partition keys
	//

	// Shards is a key to a hashmap of shards available.  The values of this
	// key are JSON-encoded shards.
	Shards() string
	// PartitionItem returns the key for the hash containing all partition items.
	PartitionItem() string
	// PartitionMeta returns the key to store metadata for partitions, eg.
	// the number of items enqueued, number in progress, etc.
	PartitionMeta(id string) string
	// GlobalPartitionIndex returns the sorted set for the partition queue;  the
	// earliest time that each function is available.  This is a global queue of
	// all functions across every partition, used for minimum latency.
	GlobalPartitionIndex() string
	// ShardPartitionIndex returns the sorted set for the shard's partition queue.
	ShardPartitionIndex(shard string) string
	// ThrottleKey returns the throttle key for a given queue item.
	ThrottleKey(t *osqueue.Throttle) string

	//
	// Queue metadata keys
	//

	// Sequential returns the key which allows a worker to claim sequential processing
	// of the partitions.
	Sequential() string
	// Scavenger returns the key which allows a worker to claim scavenger processing
	// of the partitions for lost jobs
	Scavenger() string
	// Idempotency stores the map for storing idempotency keys in redis
	Idempotency(key string) string
	// Concurrency returns a key for a given concurrency string.  This stores an ordered
	// zset of items that are in progress for the given concurrency key, giving us a total count
	// of in-progress leased items.
	Concurrency(prefix, key string) string
	// ConcurrencyIndex returns a key for storing pointers to partition concurrency queues that
	// have in-progress work.  This allows us to scan and scavenge jobs in concurrency queues where
	// leases have expired (in the case of failed workers)
	ConcurrencyIndex() string

	// RunIndex returns the index for storing job IDs associated with run IDs.
	RunIndex(runID ulid.ULID) string

	// Status returns the key used for status queue for the provided function.
	Status(status string, fnID uuid.UUID) string

	// ***************** Deprecated ************************
	BatchPointer(context.Context, uuid.UUID) string
	Batch(context.Context, ulid.ULID) string
	BatchMetadata(context.Context, ulid.ULID) string
}

type DebounceKeyGenerator interface {
	// QueueItem returns the key for the hash containing all items within a
	// queue for a function.  This is used to check leases on debounce jobs.
	QueueItem() string
	// DebouncePointer returns the key which stores the pointer to the current debounce
	// for a given function.
	DebouncePointer(ctx context.Context, fnID uuid.UUID, key string) string
	// Debounce returns the key for storing debounce-related data given a debounce ID.
	Debounce(ctx context.Context) string
}

type BatchKeyGenerator interface {
	// QueuePrefix returns the hash prefix used in the queue.
	// This is likely going to be a redis specific requirement.
	QueuePrefix() string
	// QueueItem returns the key for the hash containing all items within a
	// queue for a function.  This is used to check leases on debounce jobs.
	QueueItem() string
	// BatchPointer returns the key used as the pointer reference to the
	// actual batch
	BatchPointer(context.Context, uuid.UUID) string
	// BatchPointerWithKey returns the key used as the pointer reference to the
	// actual batch for a given batchKey
	BatchPointerWithKey(context.Context, uuid.UUID, string) string
	// Batch returns the key used to store the specific batch of
	// events, that is used to trigger a function run
	Batch(context.Context, ulid.ULID) string
	// BatchMetadata returns the key used to store the metadata related
	// to a batch
	BatchMetadata(context.Context, ulid.ULID) string
}

func (u *unshardedKeyGenerator) Shards() string {
	return fmt.Sprintf("%s:queue:shards", queueDefaultKey)
}

func (u *unshardedKeyGenerator) QueueItem() string {
	return fmt.Sprintf("%s:queue:item", queueDefaultKey)
}

func (u *unshardedKeyGenerator) QueueIndex(id string) string {
	return fmt.Sprintf("%s:queue:sorted:%s", queueDefaultKey, id)
}

func (u *unshardedKeyGenerator) PartitionItem() string {
	return fmt.Sprintf("%s:partition:item", queueDefaultKey)
}

// GlobalPartitionIndex returns the sorted index for the partition group, which stores the earliest
// time for each function/queue in the partition.
//
// This is grouped so that we can make N partitions independently.
func (u *unshardedKeyGenerator) GlobalPartitionIndex() string {
	return fmt.Sprintf("%s:partition:sorted", queueDefaultKey)
}

// GlobalPartitionIndex returns the sorted index for the partition group, which stores the earliest
// time for each function/queue in the partition.
//
// This is grouped so that we can make N partitions independently.
func (u *unshardedKeyGenerator) ShardPartitionIndex(shard string) string {
	if shard == "" {
		return fmt.Sprintf("%s:shard:-", queueDefaultKey)
	}
	return fmt.Sprintf("%s:shard:%s", queueDefaultKey, shard)
}

func (u *unshardedKeyGenerator) ThrottleKey(t *osqueue.Throttle) string {
	if t == nil || t.Key == "" {
		return fmt.Sprintf("%s:throttle:-", queueDefaultKey)
	}
	return fmt.Sprintf("%s:throttle:%s", queueDefaultKey, t.Key)
}

func (u *unshardedKeyGenerator) PartitionMeta(id string) string {
	return fmt.Sprintf("%s:partition:meta:%s", queueDefaultKey, id)
}

func (u *unshardedKeyGenerator) Sequential() string {
	return fmt.Sprintf("%s:queue:sequential", queueDefaultKey)
}

func (u *unshardedKeyGenerator) Scavenger() string {
	return fmt.Sprintf("%s:queue:scavenger", queueDefaultKey)
}

func (u *unshardedKeyGenerator) Idempotency(key string) string {
	return fmt.Sprintf("%s:queue:seen:%s", queueDefaultKey, key)
}

func (u *unshardedKeyGenerator) Concurrency(prefix, key string) string {
	if key == "" {
		// None supplied; this means ignore.
		return fmt.Sprintf("%s:-", queueDefaultKey)
	}
	return fmt.Sprintf("%s:concurrency:%s:%s", queueDefaultKey, prefix, key)
}

func (u *unshardedKeyGenerator) ConcurrencyIndex() string {
	return fmt.Sprintf("%s:concurrency:sorted", queueDefaultKey)
}

func (u *unshardedKeyGenerator) QueuePrefix() string {
	return queueDefaultKey
}

func (u *unshardedKeyGenerator) BatchPointer(ctx context.Context, workflowID uuid.UUID) string {
	return fmt.Sprintf("%s:workflows:%s:batch", queueDefaultKey, workflowID)
}

func (u *unshardedKeyGenerator) BatchPointerWithKey(ctx context.Context, workflowID uuid.UUID, batchKey string) string {
	return fmt.Sprintf("%s:%s", u.BatchPointer(ctx, workflowID), batchKey)
}

func (u *unshardedKeyGenerator) Batch(ctx context.Context, batchID ulid.ULID) string {
	return fmt.Sprintf("%s:batches:%s", queueDefaultKey, batchID)
}

func (u *unshardedKeyGenerator) BatchMetadata(ctx context.Context, batchID ulid.ULID) string {
	return fmt.Sprintf("%s:metadata", u.Batch(ctx, batchID))
}

// DebouncePointer returns the key which stores the pointer to the current debounce
// for a given function.
func (u *unshardedKeyGenerator) DebouncePointer(ctx context.Context, fnID uuid.UUID, key string) string {
	return fmt.Sprintf("%s:debounce-ptrs:%s:%s", queueDefaultKey, fnID, key)
}

// Debounce returns the key for storing debounce-related data given a debounce ID.
// This is a hash of debounce IDs -> debounces.
func (u *unshardedKeyGenerator) Debounce(ctx context.Context) string {
	return fmt.Sprintf("%s:debounce-hash", queueDefaultKey)
}

func (u *unshardedKeyGenerator) RunIndex(runID ulid.ULID) string {
	return fmt.Sprintf("%s:idx:run:%s", queueDefaultKey, runID)
}

func (u *unshardedKeyGenerator) Status(status string, fnID uuid.UUID) string {
	return fmt.Sprintf("%s:queue:status:%s:%s", queueDefaultKey, fnID, status)
}
