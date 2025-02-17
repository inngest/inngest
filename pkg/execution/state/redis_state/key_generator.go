package redis_state

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
)

type RunStateKeyGenerator interface {
	// Idempotency stores the idempotency key for atomic lookup.
	Idempotency(ctx context.Context, isSharded bool, identifier state.Identifier) string

	// RunMetadata stores state regarding the current run identifier, such
	// as the workflow version, the time the run started, etc.
	RunMetadata(ctx context.Context, isSharded bool, runID ulid.ULID) string

	// Event returns the key used to store the specific event for the
	// given workflow run.
	Event(ctx context.Context, isSharded bool, identifier state.Identifier) string

	// Events returns the key used to store the specific batch for the
	// given workflow run.
	Events(ctx context.Context, isSharded bool, identifier state.Identifier) string

	// Actions returns the key used to store the action response map used
	// for given workflow run - ie. the results for individual steps.
	Actions(ctx context.Context, isSharded bool, identifier state.Identifier) string

	// Errors returns the key used to store the error hash map used
	// for given workflow run.
	Errors(ctx context.Context, isSharded bool, identifier state.Identifier) string

	// History returns the key used to store a log entry for run hisotry
	History(ctx context.Context, isSharded bool, runID ulid.ULID) string

	// Stack returns the key used to store the stack for a given run
	Stack(ctx context.Context, isSharded bool, runID ulid.ULID) string

	// ActionInputs returns the key used to store the action inputs for a given
	// run.
	ActionInputs(ctx context.Context, isSharded bool, identifier state.Identifier) string
}

type runStateKeyGenerator struct {
	stateDefaultKey string
}

func (s runStateKeyGenerator) Prefix(ctx context.Context, defaultPrefix string, isSharded bool, runId ulid.ULID) string {
	if isSharded {
		return fmt.Sprintf("%s:%s", defaultPrefix, runId)
	}
	return defaultPrefix
}

func (s runStateKeyGenerator) PrefixByAccountId(ctx context.Context, defaultPrefix string, isSharded bool, accountId uuid.UUID) string {
	if isSharded {
		return fmt.Sprintf("%s:%s", defaultPrefix, accountId.String())
	}
	return defaultPrefix
}

func (s runStateKeyGenerator) Idempotency(ctx context.Context, isSharded bool, identifier state.Identifier) string {
	return fmt.Sprintf("{%s}:key:%s", s.PrefixByAccountId(ctx, s.stateDefaultKey, isSharded, identifier.AccountID), identifier.IdempotencyKey())
}

func (s runStateKeyGenerator) RunMetadata(ctx context.Context, isSharded bool, runID ulid.ULID) string {
	return fmt.Sprintf("{%s}:metadata:%s", s.Prefix(ctx, s.stateDefaultKey, isSharded, runID), runID)
}

func (s runStateKeyGenerator) Event(ctx context.Context, isSharded bool, identifier state.Identifier) string {
	return fmt.Sprintf("{%s}:events:%s:%s", s.Prefix(ctx, s.stateDefaultKey, isSharded, identifier.RunID), identifier.WorkflowID, identifier.RunID)
}

func (s runStateKeyGenerator) Events(ctx context.Context, isSharded bool, identifier state.Identifier) string {
	return fmt.Sprintf("{%s}:bulk-events:%s:%s", s.Prefix(ctx, s.stateDefaultKey, isSharded, identifier.RunID), identifier.WorkflowID, identifier.RunID)
}

func (s runStateKeyGenerator) Actions(ctx context.Context, isSharded bool, identifier state.Identifier) string {
	return fmt.Sprintf("{%s}:actions:%s:%s", s.Prefix(ctx, s.stateDefaultKey, isSharded, identifier.RunID), identifier.WorkflowID, identifier.RunID)
}

func (s runStateKeyGenerator) Errors(ctx context.Context, isSharded bool, identifier state.Identifier) string {
	return fmt.Sprintf("{%s}:errors:%s:%s", s.Prefix(ctx, s.stateDefaultKey, isSharded, identifier.RunID), identifier.WorkflowID, identifier.RunID)
}

func (s runStateKeyGenerator) History(ctx context.Context, isSharded bool, runID ulid.ULID) string {
	return fmt.Sprintf("{%s}:history:%s", s.Prefix(ctx, s.stateDefaultKey, isSharded, runID), runID)
}

func (s runStateKeyGenerator) Stack(ctx context.Context, isSharded bool, runID ulid.ULID) string {
	return fmt.Sprintf("{%s}:stack:%s", s.Prefix(ctx, s.stateDefaultKey, isSharded, runID), runID)
}

func (s runStateKeyGenerator) ActionInputs(ctx context.Context, isSharded bool, identifier state.Identifier) string {
	return fmt.Sprintf("{%s}:inputs:%s:%s", s.Prefix(ctx, s.stateDefaultKey, isSharded, identifier.RunID), identifier.WorkflowID, identifier.RunID)
}

type GlobalKeyGenerator interface {
	// Invoke returns the key used to store the correlation key associated with invoke functions
	Invoke(ctx context.Context, wsID uuid.UUID) string
}

type globalKeyGenerator struct {
	stateDefaultKey string
}

func (u globalKeyGenerator) Invoke(ctx context.Context, wsID uuid.UUID) string {
	return fmt.Sprintf("{%s}:invoke:%s", u.stateDefaultKey, wsID)
}

type QueueKeyGenerator interface {
	// QueueItem returns the key for the hash containing all items within a
	// queue for a function.
	QueueItem() string

	//
	// Partition keys
	//

	// PartitionItem returns the key for the hash containing all partition items.
	// This key points to a map of key → (QueuePartition{} structs stored as JSON)
	// For default partitions, the keys are the function IDs (UUIDv4 represented as strings).
	// For other partitions, the keys are exactly as returned by PartitionQueueSet(...).
	PartitionItem() string

	// GlobalPartitionIndex returns the sorted set for the partition queue;  the
	// earliest time that each function is available.  This is a global queue of
	// all functions across every partition, used for minimum latency.
	// Returns: string key, pointing to ZSET.
	// Members of this set are:
	// - for default partitions, the function ID (UUIDv4 represented as strings).
	// - for other partitions, exactly as returned by PartitionQueueSet(...).
	GlobalPartitionIndex() string

	// AccountPartitionIndex is like GlobalPartitionIndex but only includes partitions
	// for a specific account
	// Returns: string key, pointing to ZSET
	AccountPartitionIndex(accountId uuid.UUID) string

	// GlobalAccountIndex returns the sorted set for the account queue;
	// the earliest time that each account has work available. This is a global queue
	// of all accounts, used for fairness.
	// Returns: string key, pointing to ZSET
	// Members of this set are
	// - account IDs
	GlobalAccountIndex() string

	// PartitionQueueSet returns the key containing the sorted ZSET for a function's custom
	// concurrency, throttling, or (future) other custom key-based queues.
	//
	// The xxhash should be the evaluated hash of the key.
	//
	// Returns: string key, pointing to a ZSET. This is a partition; the partition data is
	// stored in the partition item (see PartitionItem()).
	PartitionQueueSet(pType enums.PartitionType, scopeID, xxhash string) string

	//
	// Queue metadata keys
	//

	// Sequential returns the key which allows a worker to claim sequential processing
	// of the partitions.
	Sequential() string
	// Scavenger returns the key which allows a worker to claim scavenger processing
	// of the partitions for lost jobs
	Scavenger() string
	// Instrumentation returns the key which allows one worker to run instrumentation against
	// the queue
	Instrumentation() string
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
	// ThrottleKey returns the throttle key for a given queue item.
	ThrottleKey(t *osqueue.Throttle) string
	// RunIndex returns the index for storing job IDs associated with run IDs.
	RunIndex(runID ulid.ULID) string

	// FnMetadata returns the key for a function's metadata.
	// This is a JSON object; see queue.FnMetadata.
	FnMetadata(fnID uuid.UUID) string
	// Status returns the key used for status queue for the provided function.
	Status(status string, fnID uuid.UUID) string

	// ConcurrencyFnEWMA returns the key storing the amount of times of concurrency hits, used for
	// calculating the EWMA value for the function
	ConcurrencyFnEWMA(fnID uuid.UUID) string

	// GuaranteedCapacityMap is a key to a hashmap of guaranteed capacities available.  The values of this
	// key are JSON-encoded GuaranteedCapacity items.
	GuaranteedCapacityMap() string

	//
	// ***************** Deprecated *****************
	//

	// FnQueueSet returns the key containing the sorted zset for a function's default queue.
	// Returns: string key, pointing to ZSET. This is a partition; the partition data is stored in
	// the partition item (see PartitionItem()).
	FnQueueSet(id string) string // deprecated
	// PartitionMeta returns the key to store metadata for partitions, eg.
	// the number of items enqueued, number in progress, etc.
	PartitionMeta(id string) string // deprecated
}

type queueKeyGenerator struct {
	queueDefaultKey string
	queueItemKeyGenerator
}

func (u queueKeyGenerator) GuaranteedCapacityMap() string {
	return fmt.Sprintf("{%s}:queue:guaranteed-capacity", u.queueDefaultKey)
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

func (u queueKeyGenerator) Instrumentation() string {
	return fmt.Sprintf("{%s}:queue:instrument", u.queueDefaultKey)
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

func (u queueKeyGenerator) AccountPartitionIndex(accountId uuid.UUID) string {
	return fmt.Sprintf("{%s}:accounts:%s:partition:sorted", u.queueDefaultKey, accountId)
}

func (u queueKeyGenerator) GlobalAccountIndex() string {
	return fmt.Sprintf("{%s}:accounts:sorted", u.queueDefaultKey)

}

func (u queueKeyGenerator) FnQueueSet(id string) string {
	return u.PartitionQueueSet(enums.PartitionTypeDefault, id, "")
}

func (u queueKeyGenerator) PartitionQueueSet(pType enums.PartitionType, scopeID, xxhash string) string {
	switch pType {
	case enums.PartitionTypeConcurrencyKey:
		return fmt.Sprintf("{%s}:sorted:c:%s<%s>", u.queueDefaultKey, scopeID, xxhash)
	case enums.PartitionTypeThrottle:
		return fmt.Sprintf("{%s}:sorted:t:%s<%s>", u.queueDefaultKey, scopeID, xxhash)
	default:
		// Default - used prior to concurrency and throttle key queues.
		return fmt.Sprintf("{%s}:queue:sorted:%s", u.queueDefaultKey, scopeID)
	}
}

func (u queueKeyGenerator) FnMetadata(fnID uuid.UUID) string {
	if fnID == uuid.Nil {
		// None supplied; this means ignore.
		return fmt.Sprintf("{%s}:fnMeta:-", u.queueDefaultKey)
	}
	return fmt.Sprintf("{%s}:fnMeta:%s", u.queueDefaultKey, fnID)
}

type BatchKeyGenerator interface {
	// QueuePrefix returns the hash prefix used in the queue.
	// This is likely going to be a redis specific requirement.
	QueuePrefix(ctx context.Context, functionId uuid.UUID) string
	// QueueItem returns the key for the hash containing all items within a
	// queue for a function.  This is used to check leases on debounce jobs.
	QueueItem() string
	// BatchPointer returns the key used as the pointer reference to the
	// actual batch
	BatchPointer(ctx context.Context, functionId uuid.UUID) string
	// BatchPointerWithKey returns the key used as the pointer reference to the
	// actual batch for a given batchKey
	BatchPointerWithKey(ctx context.Context, functionId uuid.UUID, key string) string
	// Batch returns the key used to store the specific batch of
	// events, that is used to trigger a function run
	Batch(ctx context.Context, functionId uuid.UUID, batchId ulid.ULID) string
	// BatchMetadata returns the key used to store the metadata related
	// to a batch
	BatchMetadata(ctx context.Context, functionId uuid.UUID, batchId ulid.ULID) string
}

type batchKeyGenerator struct {
	queueDefaultKey string
	queueItemKeyGenerator
}

func (u batchKeyGenerator) PrefixByFunctionId(ctx context.Context, defaultPrefix string, isSharded bool, functionId uuid.UUID) string {
	if isSharded {
		return fmt.Sprintf("%s:%s", defaultPrefix, functionId.String())
	}
	return defaultPrefix
}

func (u batchKeyGenerator) QueuePrefix(ctx context.Context, functionId uuid.UUID) string {
	return fmt.Sprintf("{%s}", u.PrefixByFunctionId(ctx, u.queueDefaultKey, true, functionId))
}

func (u batchKeyGenerator) BatchPointer(ctx context.Context, functionId uuid.UUID) string {
	return fmt.Sprintf("{%s}:workflows:%s:batch", u.PrefixByFunctionId(ctx, u.queueDefaultKey, true, functionId), functionId)
}

func (u batchKeyGenerator) BatchPointerWithKey(ctx context.Context, functionId uuid.UUID, batchKey string) string {
	return fmt.Sprintf("%s:%s", u.BatchPointer(ctx, functionId), batchKey)
}

func (u batchKeyGenerator) Batch(ctx context.Context, functionId uuid.UUID, batchID ulid.ULID) string {
	return fmt.Sprintf("{%s}:batches:%s", u.PrefixByFunctionId(ctx, u.queueDefaultKey, true, functionId), batchID)
}

func (u batchKeyGenerator) BatchMetadata(ctx context.Context, functionId uuid.UUID, batchID ulid.ULID) string {
	return fmt.Sprintf("%s:metadata", u.Batch(ctx, functionId, batchID))
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
