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

	// Events returns the key used to store the specific batch for the
	// given workflow run.
	Events(ctx context.Context, isSharded bool, fnID uuid.UUID, runID ulid.ULID) string

	// Actions returns the key used to store the action response map used
	// for given workflow run - ie. the results for individual steps.
	Actions(ctx context.Context, isSharded bool, fnID uuid.UUID, runID ulid.ULID) string

	// Stack returns the key used to store the stack for a given run
	Stack(ctx context.Context, isSharded bool, runID ulid.ULID) string

	// ActionInputs returns the key used to store the action inputs for a given
	// run.
	ActionInputs(ctx context.Context, isSharded bool, identifier state.Identifier) string

	// Pending returns the key used to store the pending actions for a given
	// run.
	Pending(ctx context.Context, isSharded bool, identifier state.Identifier) string

	// PauseConsumeKey is an idempotency key used for making sure pause consumptions are idempotent
	PauseConsumeKey(ctx context.Context, isSharded bool, runID ulid.ULID, pauseID uuid.UUID) string
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

func (s runStateKeyGenerator) Events(ctx context.Context, isSharded bool, fnID uuid.UUID, runID ulid.ULID) string {
	return fmt.Sprintf("{%s}:bulk-events:%s:%s", s.Prefix(ctx, s.stateDefaultKey, isSharded, runID), fnID, runID)
}

func (s runStateKeyGenerator) Actions(ctx context.Context, isSharded bool, fnID uuid.UUID, runID ulid.ULID) string {
	return fmt.Sprintf("{%s}:actions:%s:%s", s.Prefix(ctx, s.stateDefaultKey, isSharded, runID), fnID, runID)
}

func (s runStateKeyGenerator) Stack(ctx context.Context, isSharded bool, runID ulid.ULID) string {
	return fmt.Sprintf("{%s}:stack:%s", s.Prefix(ctx, s.stateDefaultKey, isSharded, runID), runID)
}

func (s runStateKeyGenerator) ActionInputs(ctx context.Context, isSharded bool, identifier state.Identifier) string {
	return fmt.Sprintf("{%s}:inputs:%s:%s", s.Prefix(ctx, s.stateDefaultKey, isSharded, identifier.RunID), identifier.WorkflowID, identifier.RunID)
}

func (s runStateKeyGenerator) Pending(ctx context.Context, isSharded bool, identifier state.Identifier) string {
	return fmt.Sprintf("{%s}:pending:%s:%s", s.Prefix(ctx, s.stateDefaultKey, isSharded, identifier.RunID), identifier.WorkflowID, identifier.RunID)
}

func (s runStateKeyGenerator) PauseConsumeKey(ctx context.Context, isSharded bool, runID ulid.ULID, pauseID uuid.UUID) string {
	return fmt.Sprintf("{%s}:pause-key:%s", s.Prefix(ctx, s.stateDefaultKey, isSharded, runID), pauseID.String())
}

type GlobalKeyGenerator interface {
	// Invoke returns the key used to store the correlation key associated with invoke functions
	Invoke(ctx context.Context, wsID uuid.UUID) string
	// Signal returns the key used to store the correlation key associated with
	// signal functions
	Signal(ctx context.Context, wsID uuid.UUID) string
}

type globalKeyGenerator struct {
	stateDefaultKey string
}

func (u globalKeyGenerator) Invoke(ctx context.Context, wsID uuid.UUID) string {
	return fmt.Sprintf("{%s}:invoke:%s", u.stateDefaultKey, wsID)
}

func (u globalKeyGenerator) Signal(ctx context.Context, wsID uuid.UUID) string {
	return fmt.Sprintf("{%s}:signal:%s", u.stateDefaultKey, wsID)
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
	// Backlog keys
	//

	// GlobalShadowPartitionSet returns the key to the global ZSET storing shadow partition pointers.
	GlobalShadowPartitionSet() string
	// BacklogSet returns the key to the ZSET storing pointers (queue item IDs) for a given backlog.
	BacklogSet(backlogID string) string
	// ActiveSet returns the key to the set of active queue items for a given scope and ID.
	ActiveSet(scope string, scopeID string) string
	// ActiveRunsSet returns the key to the set of active runs for a given scope and ID.
	ActiveRunsSet(scope string, scopeID string) string
	// BacklogMeta returns the key to the hash storing serialized QueueBacklog objects by ID.
	BacklogMeta() string
	// BacklogNormalizationLease returns the key to the lease for the backlog for normalization purposes
	BacklogNormalizationLease(backlogID string) string
	// ShadowPartitionSet returns the key to the ZSET storing pointers (backlog IDs) for a given shadow partition.
	ShadowPartitionSet(shadowPartitionID string) string
	// ShadowPartitionMeta returns the key to the hash storing serialized QueueShadowPartition objects by ID.
	ShadowPartitionMeta() string
	// AccountShadowPartitions returns the key to the ZSET storing pointers (shadow partition IDs) for a given account.
	AccountShadowPartitions(accountID uuid.UUID) string
	// GlobalAccountShadowPartitions returns the key to the ZSET storing pointers (account IDs) for accounts with existing shadow partitions.
	GlobalAccountShadowPartitions() string

	// RunActiveSet returns the key to the set of active queue items for a given run ID.
	RunActiveSet(runID ulid.ULID) string

	GlobalAccountNormalizeSet() string
	AccountNormalizeSet(accountID uuid.UUID) string
	PartitionNormalizeSet(partitionID string) string

	BacklogActiveCheckSet() string
	BacklogActiveCheckCooldown(backlogID string) string
	AccountActiveCheckSet() string
	AccountActiveCheckCooldown(accountID string) string

	//
	// Queue metadata keys
	//

	// Sequential returns the key which allows a worker to claim sequential processing
	// of the partitions.
	ConfigLeaseKey(scope string) string
	// ShardLeaseKey returns the key which allows a worker to claim sequential processing
	// of the partitions.
	ShardLeaseKey(scope string) string
	// Sequential returns the key which allows a worker to claim sequential processing
	// of the partitions.
	Sequential() string
	// Scavenger returns the key which allows a worker to claim scavenger processing
	// of the partitions for lost jobs
	Scavenger() string
	// Instrumentation returns the key which allows one worker to run instrumentation against
	// the queue
	Instrumentation() string
	// ActiveChecker returns the key which allows a worker to run spot checks on recently-constrained backlogs
	ActiveChecker() string
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
	// PartitionScavengerIndex returns a key to a set holding in-progress jobs
	// This allows us to scan and scavenge jobs where leases have expired (in the case of failed workers)
	PartitionScavengerIndex(partitionID string) string
	// ThrottleKey returns the throttle key for a given queue item.
	ThrottleKey(t *osqueue.Throttle) string
	// RunIndex returns the index for storing job IDs associated with run IDs.
	RunIndex(runID ulid.ULID) string
	// SingletonKey returns the singleton key for a given queue item.
	SingletonKey(s *osqueue.Singleton) string
	// SingletonRunKey returns the singleton run id key that stores the singleton key for a given run.
	SingletonRunKey(r string) string

	// FnMetadata returns the key for a function's metadata.
	// This is a JSON object; see queue.FnMetadata.
	FnMetadata(fnID uuid.UUID) string
	QueueMigrationLock(fnID uuid.UUID) string
	// Status returns the key used for status queue for the provided function.
	Status(status string, fnID uuid.UUID) string

	// ConcurrencyFnEWMA returns the key storing the amount of times of concurrency hits, used for
	// calculating the EWMA value for the function
	ConcurrencyFnEWMA(fnID uuid.UUID) string

	// QueuePrefix returns the hash prefix used in the queue.
	// This is likely going to be a redis specific requirement.
	QueuePrefix() string

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

func (u queueKeyGenerator) SingletonKey(s *osqueue.Singleton) string {
	if s == nil || s.Key == "" {
		return fmt.Sprintf("{%s}:singleton:-", u.queueDefaultKey)
	}

	return fmt.Sprintf("{%s}:singleton:%s", u.queueDefaultKey, s.Key)
}

func (u queueKeyGenerator) SingletonRunKey(runID string) string {
	return fmt.Sprintf("{%s}:singleton-run:%s", u.queueDefaultKey, runID)
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

func (u queueKeyGenerator) ConfigLeaseKey(scope string) string {
	return fmt.Sprintf("{%s}:queue:%s", u.queueDefaultKey, scope)
}

func (u queueKeyGenerator) ShardLeaseKey(scope string) string {
	return fmt.Sprintf("{%s}:queue:%s", u.queueDefaultKey, scope)
}

func (u queueKeyGenerator) ActiveChecker() string {
	return fmt.Sprintf("{%s}:queue:active-checker", u.queueDefaultKey)
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

func (u queueKeyGenerator) PartitionScavengerIndex(partitionID string) string {
	return fmt.Sprintf("{%s}:scavenger:%s:sorted", u.queueDefaultKey, partitionID)
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

// BacklogSet returns the key to the ZSET storing pointers (queue item IDs) for a given backlog.
func (u queueKeyGenerator) BacklogSet(backlogID string) string {
	if backlogID == "" {
		// this is a placeholder because passing an empty key into Lua will cause multi-slot key errors
		return fmt.Sprintf("{%s}:backlog:sorted:-", u.queueDefaultKey)
	}

	return fmt.Sprintf("{%s}:backlog:sorted:%s", u.queueDefaultKey, backlogID)
}

// ActiveSet returns the key to the number of active queue items for a given backlog.
func (u queueKeyGenerator) ActiveSet(scope string, scopeID string) string {
	if scope == "" || scopeID == "" {
		// this is a placeholder because passing an empty key into Lua will cause multi-slot key errors
		return fmt.Sprintf("{%s}:v2:active:-", u.queueDefaultKey)
	}

	return fmt.Sprintf("{%s}:v2:active:%s:%s", u.queueDefaultKey, scope, scopeID)
}

func isEmptyULID(id ulid.ULID) bool {
	return id == [16]byte{}
}

func (u queueKeyGenerator) RunActiveSet(runID ulid.ULID) string {
	if isEmptyULID(runID) {
		// this is a placeholder because passing an empty key into Lua will cause multi-slot key errors
		return u.ActiveSet("run", "")
	}

	return u.ActiveSet("run", runID.String())
}

// ActiveRunsSet returns the key to the number of active runs for a given backlog.
func (u queueKeyGenerator) ActiveRunsSet(scope string, scopeID string) string {
	if scope == "" || scopeID == "" {
		// this is a placeholder because passing an empty key into Lua will cause multi-slot key errors
		return fmt.Sprintf("{%s}:v2:active-runs:-", u.queueDefaultKey)
	}

	return fmt.Sprintf("{%s}:v2:active-runs:%s:%s", u.queueDefaultKey, scope, scopeID)
}

// BacklogMeta returns the key to the hash storing serialized QueueBacklog objects by ID.
func (u queueKeyGenerator) BacklogMeta() string {
	return fmt.Sprintf("{%s}:backlogs", u.queueDefaultKey)
}

// BacklogNormalizationLease returns the key for the lease of the backlog for normalization purposes
func (u queueKeyGenerator) BacklogNormalizationLease(backlogID string) string {
	return fmt.Sprintf("{%s}:backlog:%s:lease", u.queueDefaultKey, backlogID)
}

// GlobalShadowPartitionSet returns the key to the global ZSET storing shadow partition pointers.
func (u queueKeyGenerator) GlobalShadowPartitionSet() string {
	return fmt.Sprintf("{%s}:shadow:sorted", u.queueDefaultKey)
}

func (u queueKeyGenerator) AccountShadowPartitions(accountID uuid.UUID) string {
	if accountID == uuid.Nil {
		// this is a placeholder because passing an empty key into Lua will cause multi-slot key errors
		return fmt.Sprintf("{%s}:accounts:shadows:sorted:-", u.queueDefaultKey)
	}

	return fmt.Sprintf("{%s}:accounts:%s:shadows:sorted", u.queueDefaultKey, accountID)
}

func (u queueKeyGenerator) GlobalAccountShadowPartitions() string {
	return fmt.Sprintf("{%s}:accounts:shadows:sorted", u.queueDefaultKey)
}

func (u queueKeyGenerator) GlobalAccountNormalizeSet() string {
	return fmt.Sprintf("{%s}:normalize:sorted", u.queueDefaultKey)
}

func (u queueKeyGenerator) AccountNormalizeSet(accountID uuid.UUID) string {
	if accountID == uuid.Nil {
		return fmt.Sprintf("{%s}:normalize:-", u.queueDefaultKey)
	}

	return fmt.Sprintf("{%s}:normalize:accounts:%s:sorted", u.queueDefaultKey, accountID.String())
}

func (u queueKeyGenerator) PartitionNormalizeSet(partitionID string) string {
	if partitionID == "" {
		return fmt.Sprintf("{%s}:normalize:-", u.queueDefaultKey)
	}

	return fmt.Sprintf("{%s}:normalize:partition:%s:sorted", u.queueDefaultKey, partitionID)
}

func (u queueKeyGenerator) BacklogActiveCheckSet() string {
	return fmt.Sprintf("{%s}:active-check:backlog:sorted", u.queueDefaultKey)
}

func (u queueKeyGenerator) BacklogActiveCheckCooldown(backlogID string) string {
	if backlogID == "" {
		return fmt.Sprintf("{%s}:active-check:cooldown:backlog:-", u.queueDefaultKey)
	}
	return fmt.Sprintf("{%s}:active-check:cooldown:backlog:%s", u.queueDefaultKey, backlogID)
}

func (u queueKeyGenerator) AccountActiveCheckSet() string {
	return fmt.Sprintf("{%s}:active-check:account:sorted", u.queueDefaultKey)
}

func (u queueKeyGenerator) AccountActiveCheckCooldown(accountID string) string {
	if accountID == "" {
		return fmt.Sprintf("{%s}:active-check:cooldown:account:-", u.queueDefaultKey)
	}
	return fmt.Sprintf("{%s}:active-check:cooldown:account:%s", u.queueDefaultKey, accountID)
}

func (u queueKeyGenerator) QueuePrefix() string {
	return fmt.Sprintf("{%s}", u.queueDefaultKey)
}

// ShadowPartitionSet returns the key to the ZSET storing pointers (backlog IDs) for a given shadow partition.
func (u queueKeyGenerator) ShadowPartitionSet(shadowPartitionID string) string {
	return fmt.Sprintf("{%s}:shadow:sorted:%s", u.queueDefaultKey, shadowPartitionID)
}

// ShadowPartitionMeta returns the key to the hash storing serialized QueueShadowPartition objects by ID.
func (u queueKeyGenerator) ShadowPartitionMeta() string {
	return fmt.Sprintf("{%s}:shadows", u.queueDefaultKey)
}

func (u queueKeyGenerator) FnMetadata(fnID uuid.UUID) string {
	if fnID == uuid.Nil {
		// None supplied; this means ignore.
		return fmt.Sprintf("{%s}:fnMeta:-", u.queueDefaultKey)
	}
	return fmt.Sprintf("{%s}:fnMeta:%s", u.queueDefaultKey, fnID)
}

func (u queueKeyGenerator) QueueMigrationLock(fnID uuid.UUID) string {
	if fnID == uuid.Nil {
		// None supplied; this means ignore.
		return fmt.Sprintf("{%s}:migrate-lock:-", u.queueDefaultKey)
	}
	return fmt.Sprintf("{%s}:migrate-lock:%s", u.queueDefaultKey, fnID)
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
	// BatchIdempotenceKey returns the key used to store the specific batch of
	// events, that is used to check if a batch event has already been appended for a function
	BatchIdempotenceKey(ctx context.Context, functionId uuid.UUID) string
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

func (u batchKeyGenerator) BatchIdempotenceKey(ctx context.Context, functionId uuid.UUID) string {
	return fmt.Sprintf("{%s}:batch_idempotence", u.PrefixByFunctionId(ctx, u.queueDefaultKey, true, functionId))
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
	// DebounceMigrating returns the key for storing the in-progress debounce migration flag to prevent
	// migrations and timeout execution from racing. This is a hash.
	DebounceMigrating(ctx context.Context) string
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

// DebounceMigrating returns the key for storing the in-progress debounce migration flag to prevent
// migrations and timeout execution from racing. This is a hash.
func (u debounceKeyGenerator) DebounceMigrating(ctx context.Context) string {
	return fmt.Sprintf("{%s}:debounce-migrating", u.queueDefaultKey)
}

type PauseKeyGenerator interface {
	// Pause returns the key used to store an individual pause from its ID.
	Pause(ctx context.Context, pauseID uuid.UUID) string

	// GlobalPauseIndex returns the key used to index all pauses.
	GlobalPauseIndex(ctx context.Context) string

	// RunPauses stores pause IDs for each run as a zset
	RunPauses(ctx context.Context, runID ulid.ULID) string

	// PauseLease stores the key which references a pause's lease.
	//
	// This is stored independently as we may store more than one copy of a pause
	// for easy iteration.
	PauseLease(ctx context.Context, pauseId uuid.UUID) string

	// PauseEvent returns the key used to store data for loading pauses by events.
	PauseEvent(ctx context.Context, workspaceId uuid.UUID, event string) string

	// PauseIndex is a key that's used to index added/expired times for pauses.
	//
	// Added times are necessary to load pauses after a specific point in time,
	// which is used when caching pauses in-memory to only load the subset of pauses
	// added after the cache was last updated.
	PauseIndex(ctx context.Context, kind string, wsID uuid.UUID, event string) string

	// PauseBlockIndex is a key that's used to keep the block ID of the flushed pause
	// so we can still get pauses by ID from blocks.
	PauseBlockIndex(ctx context.Context, pauseID uuid.UUID) string
}

type pauseKeyGenerator struct {
	stateDefaultKey string
}

func (u pauseKeyGenerator) Pause(ctx context.Context, pauseID uuid.UUID) string {
	return fmt.Sprintf("{%s}:pauses:%s", u.stateDefaultKey, pauseID.String())
}

func (u pauseKeyGenerator) GlobalPauseIndex(ctx context.Context) string {
	return fmt.Sprintf("{%s}:pauses-idx", u.stateDefaultKey)
}

func (u pauseKeyGenerator) RunPauses(ctx context.Context, runID ulid.ULID) string {
	return fmt.Sprintf("{%s}:pr:%s", u.stateDefaultKey, runID)
}

func (u pauseKeyGenerator) PauseLease(ctx context.Context, pauseID uuid.UUID) string {
	return fmt.Sprintf("{%s}:pause-lease:%s", u.stateDefaultKey, pauseID.String())
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

func (u pauseKeyGenerator) PauseBlockIndex(ctx context.Context, pauseID uuid.UUID) string {
	return fmt.Sprintf("{%s}:pause-block:%s", u.stateDefaultKey, pauseID.String())
}

type queueItemKeyGenerator struct {
	queueDefaultKey string
}

func (u queueItemKeyGenerator) QueueItem() string {
	return fmt.Sprintf("{%s}:queue:item", u.queueDefaultKey)
}
