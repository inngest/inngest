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

// TODO Set this properly, don't start sharding any time soon before we are sure everything works
var switchover = time.Date(2100, 07, 10, 0, 0, 0, 0, time.UTC)

func IsSharded(runID ulid.ULID) bool {
	return ulid.Time(runID.Time()).After(switchover)
}

type RunStateKeyGenerator interface {
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

	// History returns the key used to store a log entry for run hisotry
	History(ctx context.Context, runID ulid.ULID) string

	// Stack returns the key used to store the stack for a given run
	Stack(ctx context.Context, runID ulid.ULID) string
}

type runStateKeyGenerator struct {
	stateDefaultKey string
}

func (s runStateKeyGenerator) Prefix(defaultPrefix string, runID ulid.ULID) string {
	if IsSharded(runID) {
		return fmt.Sprintf("%s:%s", defaultPrefix, runID)
	}
	return defaultPrefix
}

func (s runStateKeyGenerator) Idempotency(ctx context.Context, id state.Identifier) string {
	return fmt.Sprintf("{%s}:key:%s", s.Prefix(s.stateDefaultKey, id.RunID), id.IdempotencyKey())
}

func (s runStateKeyGenerator) RunMetadata(ctx context.Context, runID ulid.ULID) string {
	return fmt.Sprintf("{%s}:metadata:%s", s.Prefix(s.stateDefaultKey, runID), runID)
}

func (s runStateKeyGenerator) Event(ctx context.Context, identifier state.Identifier) string {
	return fmt.Sprintf("{%s}:events:%s:%s", s.Prefix(s.stateDefaultKey, identifier.RunID), identifier.WorkflowID, identifier.RunID)
}

func (s runStateKeyGenerator) Events(ctx context.Context, identifier state.Identifier) string {
	return fmt.Sprintf("{%s}:bulk-events:%s:%s", s.Prefix(s.stateDefaultKey, identifier.RunID), identifier.WorkflowID, identifier.RunID)
}

func (s runStateKeyGenerator) Actions(ctx context.Context, identifier state.Identifier) string {
	return fmt.Sprintf("{%s}:actions:%s:%s", s.Prefix(s.stateDefaultKey, identifier.RunID), identifier.WorkflowID, identifier.RunID)
}

func (s runStateKeyGenerator) Errors(ctx context.Context, identifier state.Identifier) string {
	return fmt.Sprintf("{%s}:errors:%s:%s", s.Prefix(s.stateDefaultKey, identifier.RunID), identifier.WorkflowID, identifier.RunID)
}

func (s runStateKeyGenerator) History(ctx context.Context, runID ulid.ULID) string {
	return fmt.Sprintf("{%s}:history:%s", s.Prefix(s.stateDefaultKey, runID), runID)
}

func (s runStateKeyGenerator) Stack(ctx context.Context, runID ulid.ULID) string {
	return fmt.Sprintf("{%s}:stack:%s", s.Prefix(s.stateDefaultKey, runID), runID)
}

type GlobalKeyGenerator interface {
	// Workflow returns the key for the current workflow ID and version.
	Workflow(ctx context.Context, workflowID uuid.UUID, version int) string

	// Invoke returns the key used to store the correlation key associated with invoke functions
	Invoke(ctx context.Context, wsID uuid.UUID) string
}

type PauseKeyGenerator interface {
	// Pause returns the key used to store an individual pause from its ID.
	Pause(ctx context.Context, pauseID uuid.UUID) string

	// RunPauses stores pause IDs for each run as a zset
	RunPauses(ctx context.Context, runID ulid.ULID) string

	// PauseLease stores the key which references a pause's lease.
	//
	// This is stored independently as we may store more than one copy of a pause
	// for easy iteration.
	PauseLease(ctx context.Context, pauseId uuid.UUID) string

	// PauseStep returns the prefix of the key used within PauseStep.  This lets us
	// iterate through all pauses for a given identifier
	PauseStepPrefix(context.Context, state.Identifier) string

	// PauseStep returns the key used to store a pause ID by the run ID and step ID.
	PauseStep(context.Context, state.Identifier, string) string

	// PauseEvent returns the key used to store data for loading pauses by events.
	PauseEvent(ctx context.Context, workspaceId uuid.UUID, event string) string

	// PauseIndex is a key that's used to index added/expired times for pauses.
	//
	// Added times are necessary to load pauses after a specific point in time,
	// which is used when caching pauses in-memory to only load the subset of pauses
	// added after the cache was last updated.
	PauseIndex(ctx context.Context, kind string, wsID uuid.UUID, event string) string
}

type globalKeyGenerator struct {
	stateDefaultKey string
}

func (u globalKeyGenerator) Workflow(ctx context.Context, workflowID uuid.UUID, version int) string {
	return fmt.Sprintf("{%s}:workflows:%s-%d", u.stateDefaultKey, workflowID, version)
}

func (u globalKeyGenerator) Invoke(ctx context.Context, wsID uuid.UUID) string {
	return fmt.Sprintf("{%s}:invoke:%s", u.stateDefaultKey, wsID)
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

	// ConcurrencyFnEWMA returns the key storing the amount of times of concurrency hits, used for
	// calculating the EWMA value for the function
	ConcurrencyFnEWMA(fnID uuid.UUID) string
}

type queueKeyGenerator struct {
	queueDefaultKey string
	queueItemKeyGenerator
}

func (u queueKeyGenerator) Shards() string {
	return fmt.Sprintf("{%s}:queue:shards", u.queueDefaultKey)
}

func (u queueKeyGenerator) QueueIndex(id string) string {
	return fmt.Sprintf("{%s}:queue:sorted:%s", u.queueDefaultKey, id)
}

func (u queueKeyGenerator) PartitionItem() string {
	return fmt.Sprintf("{%s}:partition:item", u.queueDefaultKey)
}

// GlobalPartitionIndex returns the sorted index for the partition group, which stores the earliest
// time for each function/queue in the partition.
//
// This is grouped so that we can make N partitions independently.
func (u queueKeyGenerator) GlobalPartitionIndex() string {
	return fmt.Sprintf("{%s}:partition:sorted", u.queueDefaultKey)
}

// GlobalPartitionIndex returns the sorted index for the partition group, which stores the earliest
// time for each function/queue in the partition.
//
// This is grouped so that we can make N partitions independently.
func (u queueKeyGenerator) ShardPartitionIndex(shard string) string {
	if shard == "" {
		return fmt.Sprintf("{%s}:shard:-", u.queueDefaultKey)
	}
	return fmt.Sprintf("{%s}:shard:%s", u.queueDefaultKey, shard)
}

func (u queueKeyGenerator) ThrottleKey(t *osqueue.Throttle) string {
	if t == nil || t.Key == "" {
		return fmt.Sprintf("{%s}:throttle:-", u.queueDefaultKey)
	}
	return fmt.Sprintf("{%s}:throttle:%s", u.queueDefaultKey, t.Key)
}

func (u queueKeyGenerator) PartitionMeta(id string) string {
	return fmt.Sprintf("{%s}:partition:meta:%s", u.queueDefaultKey, id)
}

func (u queueKeyGenerator) Sequential() string {
	return fmt.Sprintf("{%s}:queue:sequential", u.queueDefaultKey)
}

func (u queueKeyGenerator) Scavenger() string {
	return fmt.Sprintf("{%s}:queue:scavenger", u.queueDefaultKey)
}

func (u queueKeyGenerator) Idempotency(key string) string {
	return fmt.Sprintf("{%s}:queue:seen:%s", u.queueDefaultKey, key)
}

func (u queueKeyGenerator) Concurrency(prefix, key string) string {
	if key == "" {
		// None supplied; this means ignore.
		return fmt.Sprintf("{%s}:-", u.queueDefaultKey)
	}
	return fmt.Sprintf("{%s}:concurrency:%s:%s", u.queueDefaultKey, prefix, key)
}

func (u queueKeyGenerator) ConcurrencyIndex() string {
	return fmt.Sprintf("{%s}:concurrency:sorted", u.queueDefaultKey)
}

func (u queueKeyGenerator) RunIndex(runID ulid.ULID) string {
	return fmt.Sprintf("{%s}:idx:run:%s", u.queueDefaultKey, runID)
}

func (u queueKeyGenerator) Status(status string, fnID uuid.UUID) string {
	return fmt.Sprintf("{%s}:queue:status:%s:%s", u.queueDefaultKey, fnID, status)
}

func (u queueKeyGenerator) ConcurrencyFnEWMA(fnID uuid.UUID) string {
	return fmt.Sprintf("{%s}:queue:concurrency-ewma:%s", u.queueDefaultKey, fnID)
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

type batchKeyGenerator struct {
	queueDefaultKey string
	queueItemKeyGenerator
}

func (u batchKeyGenerator) QueuePrefix() string {
	return fmt.Sprintf("{%s}", u.queueDefaultKey)
}

func (u batchKeyGenerator) BatchPointer(ctx context.Context, workflowID uuid.UUID) string {
	return fmt.Sprintf("{%s}:workflows:%s:batch", u.queueDefaultKey, workflowID)
}

func (u batchKeyGenerator) BatchPointerWithKey(ctx context.Context, workflowID uuid.UUID, batchKey string) string {
	return fmt.Sprintf("%s:%s", u.BatchPointer(ctx, workflowID), batchKey)
}

func (u batchKeyGenerator) Batch(ctx context.Context, batchID ulid.ULID) string {
	return fmt.Sprintf("{%s}:batches:%s", u.queueDefaultKey, batchID)
}

func (u batchKeyGenerator) BatchMetadata(ctx context.Context, batchID ulid.ULID) string {
	return fmt.Sprintf("%s:metadata", u.Batch(ctx, batchID))
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

type debounceKeyGenerator struct {
	queueDefaultKey string
	queueItemKeyGenerator
}

// DebouncePointer returns the key which stores the pointer to the current debounce
// for a given function.
func (u debounceKeyGenerator) DebouncePointer(ctx context.Context, fnID uuid.UUID, key string) string {
	return fmt.Sprintf("{%s}:debounce-ptrs:%s:%s", u.queueDefaultKey, fnID, key)
}

// Debounce returns the key for storing debounce-related data given a debounce ID.
// This is a hash of debounce IDs -> debounces.
func (u debounceKeyGenerator) Debounce(ctx context.Context) string {
	return fmt.Sprintf("{%s}:debounce-hash", u.queueDefaultKey)
}

type pauseKeyGenerator struct {
	stateDefaultKey string
}

func (u pauseKeyGenerator) Pause(ctx context.Context, pauseID uuid.UUID) string {
	return fmt.Sprintf("{%s}:pauses:%s", u.stateDefaultKey, pauseID.String())
}

func (u pauseKeyGenerator) RunPauses(ctx context.Context, runID ulid.ULID) string {
	return fmt.Sprintf("{%s}:pr:%s", u.stateDefaultKey, runID)
}

func (u pauseKeyGenerator) PauseLease(ctx context.Context, pauseID uuid.UUID) string {
	return fmt.Sprintf("{%s}:pause-lease:%s", u.stateDefaultKey, pauseID.String())
}

func (u pauseKeyGenerator) PauseStepPrefix(ctx context.Context, identifier state.Identifier) string {
	return fmt.Sprintf("{%s}:pause-steps:%s", u.stateDefaultKey, identifier.RunID)
}

func (u pauseKeyGenerator) PauseStep(ctx context.Context, identifier state.Identifier, stepId string) string {
	prefix := u.PauseStepPrefix(ctx, identifier)
	return fmt.Sprintf("%s-%s", prefix, stepId)
}

func (u pauseKeyGenerator) PauseEvent(ctx context.Context, workspaceID uuid.UUID, s string) string {
	return fmt.Sprintf("{%s}:pause-events:%s:%s", u.stateDefaultKey, workspaceID, s)
}

func (u pauseKeyGenerator) PauseIndex(ctx context.Context, kind string, wsID uuid.UUID, event string) string {
	if event == "" {
		return fmt.Sprintf("{%s}:pause-idx:%s:%s:-", u.stateDefaultKey, kind, wsID)
	}
	return fmt.Sprintf("{%s}:pause-idx:%s:%s:%s", u.stateDefaultKey, kind, wsID, event)
}

type queueItemKeyGenerator struct {
	queueDefaultKey string
}

func (u queueItemKeyGenerator) QueueItem() string {
	return fmt.Sprintf("{%s}:queue:item", u.queueDefaultKey)
}
