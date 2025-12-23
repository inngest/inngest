package queue

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/backoff"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
)

// PartitionPriorityFinder returns the priority for a given queue partition.
type PartitionPriorityFinder func(ctx context.Context, part QueuePartition) uint

// AccountPriorityFinder returns the priority for a given account.
type AccountPriorityFinder func(ctx context.Context, accountId uuid.UUID) uint

type PartitionPausedInfo struct {
	Stale  bool
	Paused bool
}
type PartitionPausedGetter func(ctx context.Context, fnID uuid.UUID) PartitionPausedInfo

type QueueOpt func(q *QueueOptions)

func WithName(name string) func(q *QueueOptions) {
	return func(q *QueueOptions) {
		q.name = name
	}
}

func WithQueueLifecycles(l ...QueueLifecycleListener) QueueOpt {
	return func(q *QueueOptions) {
		q.lifecycles = l
	}
}

func WithPartitionPriorityFinder(ppf PartitionPriorityFinder) QueueOpt {
	return func(q *QueueOptions) {
		q.ppf = ppf
	}
}

func WithPartitionPausedGetter(partitionPausedGetter PartitionPausedGetter) QueueOpt {
	return func(q *queue) {
		q.partitionPausedGetter = partitionPausedGetter
	}
}

func WithAccountPriorityFinder(apf AccountPriorityFinder) QueueOpt {
	return func(q *queue) {
		q.apf = apf
	}
}

func WithIdempotencyTTL(t time.Duration) QueueOpt {
	return func(q *queue) {
		q.idempotencyTTL = t
	}
}

// WithIdempotencyTTLFunc returns custom idempotecy durations given a QueueItem.
// This allows customization of the idempotency TTL based off of specific jobs.
func WithIdempotencyTTLFunc(f func(context.Context, osqueue.QueueItem) time.Duration) QueueOpt {
	return func(q *queue) {
		q.idempotencyTTLFunc = f
	}
}

func WithNumWorkers(n int32) QueueOpt {
	return func(q *queue) {
		q.numWorkers = n
	}
}

func WithShadowNumWorkers(n int32) QueueOpt {
	return func(q *queue) {
		q.numShadowWorkers = n
	}
}

func WithPeekSizeRange(min int64, max int64) QueueOpt {
	return func(q *queue) {
		if max > AbsoluteQueuePeekMax {
			max = AbsoluteQueuePeekMax
		}
		q.peekMin = min
		q.peekMax = max
	}
}

func WithShadowPeekSizeRange(min int64, max int64) QueueOpt {
	return func(q *queue) {
		if max > AbsoluteShadowPartitionPeekMax {
			max = AbsoluteShadowPartitionPeekMax
		}
		q.shadowPeekMin = min
		q.shadowPeekMax = max
	}
}

func WithBacklogRefillLimit(limit int64) QueueOpt {
	return func(q *queue) {
		q.backlogRefillLimit = limit
	}
}

func WithBacklogNormalizationConcurrency(limit int64) QueueOpt {
	return func(q *queue) {
		q.backlogNormalizeConcurrency = limit
	}
}

func WithPeekConcurrencyMultiplier(m int64) QueueOpt {
	return func(q *queue) {
		q.peekCurrMultiplier = m
	}
}

func WithPeekEWMALength(l int) QueueOpt {
	return func(q *queue) {
		q.peekEWMALen = l
	}
}

// WithPollTick specifies the interval at which the queue will poll the backing store
// for available partitions.
func WithPollTick(t time.Duration) QueueOpt {
	return func(q *queue) {
		q.pollTick = t
	}
}

// WithShadowPollTick specifies the interval at which the queue will poll the backing store
// for available shadow partitions.
func WithShadowPollTick(t time.Duration) QueueOpt {
	return func(q *queue) {
		q.shadowPollTick = t
	}
}

// WithBacklogNormalizePollTick specifies the interval at which the queue will poll the backing store
// for available backlogs to normalize.
func WithBacklogNormalizePollTick(t time.Duration) QueueOpt {
	return func(q *queue) {
		q.backlogNormalizePollTick = t
	}
}

// WithActiveCheckPollTick specifies the interval at which the queue will poll the backing store
// for available backlogs to normalize.
func WithActiveCheckPollTick(t time.Duration) QueueOpt {
	return func(q *queue) {
		q.activeCheckTick = t
	}
}

// WithActiveCheckAccountProbability specifies the probability of processing accounts vs. backlogs during an active check run.
func WithActiveCheckAccountProbability(p int) QueueOpt {
	return func(q *queue) {
		q.activeCheckAccountProbability = p
	}
}

// WithActiveCheckAccountConcurrency specifies the number of accounts to be peeked and processed by the active checker in parallel
func WithActiveCheckAccountConcurrency(p int) QueueOpt {
	return func(q *queue) {
		if p > 0 {
			q.activeCheckAccountConcurrency = int64(p)
		}
	}
}

// WithActiveCheckBacklogConcurrency specifies the number of backlogs to be peeked and processed by the active checker in parallel
func WithActiveCheckBacklogConcurrency(p int) QueueOpt {
	return func(q *queue) {
		if p > 0 {
			q.activeCheckBacklogConcurrency = int64(p)
		}
	}
}

// WithActiveCheckScanBatchSize specifies the batch size for iterating over active sets
func WithActiveCheckScanBatchSize(p int) QueueOpt {
	return func(q *queue) {
		if p > 0 {
			q.activeCheckScanBatchSize = int64(p)
		}
	}
}

func WithQueueItemIndexer(i QueueItemIndexer) QueueOpt {
	return func(q *queue) {
		q.itemIndexer = i
	}
}

// WithDenyQueueNames specifies that the worker cannot select jobs from queue partitions
// within the given list of names.  This means that the worker will never work on jobs
// in the specified queues.
//
// NOTE: If this is set and this worker claims the sequential lease, there is no guarantee
// on latency or fairness in the denied queue partitions.
func WithDenyQueueNames(queues ...string) QueueOpt {
	return func(q *queue) {
		q.denyQueues = queues
		q.denyQueueMap = make(map[string]*struct{})
		q.denyQueuePrefixes = make(map[string]*struct{})
		for _, i := range queues {
			q.denyQueueMap[i] = &struct{}{}
			// If WithDenyQueueNames includes "user:*", trim the asterisc and use
			// this as a prefix match.
			if strings.HasSuffix(i, "*") {
				q.denyQueuePrefixes[strings.TrimSuffix(i, "*")] = &struct{}{}
			}
		}
	}
}

// WithAllowQueueNames specifies that the worker can only select jobs from queue partitions
// within the given list of names.  This means that the worker will never work on jobs in
// other queues.
func WithAllowQueueNames(queues ...string) QueueOpt {
	return func(q *queue) {
		q.allowQueues = queues
		q.allowQueueMap = make(map[string]*struct{})
		q.allowQueuePrefixes = make(map[string]*struct{})
		for _, i := range queues {
			q.allowQueueMap[i] = &struct{}{}
			// If WithAllowQueueNames includes "user:*", trim the asterisc and use
			// this as a prefix match.
			if strings.HasSuffix(i, "*") {
				q.allowQueuePrefixes[strings.TrimSuffix(i, "*")] = &struct{}{}
			}
		}
	}
}

// WithKindToQueueMapping maps queue.Item.Kind strings to queue names.  For example,
// when pushing a queue.Item with a kind of PayloadEdge, this job can be mapped to
// a specific queue name here.
//
// The mapping must be provided in terms of item kind to queue name.  If the item
// kind doesn't exist in the mapping the job's queue name will be left nil.  This
// means that the item will be placed in the workflow ID's queue.
func WithKindToQueueMapping(mapping map[string]string) QueueOpt {
	// XXX: Refactor osqueue.Item and this package to resolve these interfaces
	// and clean up this function.
	return func(q *queue) {
		q.queueKindMapping = mapping
	}
}

func WithDisableFifoForFunctions(mapping map[string]struct{}) QueueOpt {
	return func(q *queue) {
		q.disableFifoForFunctions = mapping
	}
}

func WithPeekSizeForFunction(mapping map[string]int64) QueueOpt {
	return func(q *queue) {
		q.peekSizeForFunctions = mapping
	}
}

func WithDisableFifoForAccounts(mapping map[string]struct{}) QueueOpt {
	return func(q *queue) {
		q.disableFifoForAccounts = mapping
	}
}

func WithLogger(l logger.Logger) QueueOpt {
	return func(q *queue) {
		q.log = l
	}
}

func WithBackoffFunc(f backoff.BackoffFunc) func(q *queue) {
	return func(q *queue) {
		q.backoffFunc = f
	}
}

func WithRunMode(m QueueRunMode) func(q *queue) {
	return func(q *queue) {
		q.runMode = m
	}
}

// WithClock allows replacing the queue's default (real) clock by a mock, for testing.
func WithClock(c clockwork.Clock) func(q *queue) {
	return func(q *queue) {
		q.clock = c
	}
}

// WithQueueContinuationLimit sets the continuation limit in the queue, eg. how many
// sequential steps cause hints in the queue to continue executing the same partition.
func WithQueueContinuationLimit(limit uint) QueueOpt {
	return func(q *queue) {
		q.continuationLimit = limit
	}
}

// WithQueueShadowContinuationLimit sets the shadow continuation limit in the queue, eg. how many
// sequential steps cause hints in the queue to continue executing the same shadow partition.
func WithQueueShadowContinuationLimit(limit uint) QueueOpt {
	return func(q *queue) {
		q.shadowContinuationLimit = limit
	}
}

type QueueProcessor struct {
	// name is the identifiable name for this worker, for logging.
	name string

	// quit is a channel that any method can send on to trigger termination
	// of the Run loop.  This typically accepts an error, but a nil error
	// will still quit the runner.
	quit chan error
	// wg stores a waitgroup for all in-progress jobs
	wg *sync.WaitGroup

	// activeCheckerLeaseID stores the lease ID if this queue is the ActiveChecker processor.
	// all runners attempt to claim this lease automatically.
	activeCheckerLeaseID *ulid.ULID
	// activeCheckerLeaseLock ensures that there are no data races writing to
	// or reading from activeCheckerLeaseID in parallel.
	activeCheckerLeaseLock *sync.RWMutex

	// workers is a buffered channel which allows scanners to send queue items
	// to workers to be processed
	workers chan processItem
	// sem stores a semaphore controlling the number of jobs currently
	// being processed.  This lets us check whether there's capacity in the queue
	// prior to leasing items.
	sem *trackingSemaphore

	// seqLeaseID stores the lease ID if this queue is the sequential processor.
	// all runners attempt to claim this lease automatically.
	seqLeaseID *ulid.ULID
	// seqLeaseLock ensures that there are no data races writing to
	// or reading from seqLeaseID in parallel.
	seqLeaseLock *sync.RWMutex

	// instrumentationLeaseID stores the lease ID if executor is running queue
	// instrumentations
	instrumentationLeaseID *ulid.ULID
	// instrumentationLeaseLock ensures that there are no data races writing to or
	// reading from instrumentationLeaseID
	instrumentationLeaseLock *sync.RWMutex

	// continues stores a map of all partition IDs to continues for a partition.
	// this lets us optimize running consecutive steps for a function, as a continuation, to a specific limit.
	continues        map[string]continuation
	continueCooldown map[string]time.Time

	// continuesLock protects the continues map.
	continuesLock *sync.Mutex

	// scavengerLeaseID stores the lease ID if this queue is the scavenger processor.
	// all runners attempt to claim this lease automatically.
	scavengerLeaseID *ulid.ULID
	// scavengerLeaseLock ensures that there are no data races writing to
	// or reading from scavengerLeaseID in parallel.
	scavengerLeaseLock *sync.RWMutex
}

type QueueOptions struct {
	// primaryQueueShard stores the queue shard to use.
	primaryQueueShard QueueShard

	// queueShardClients contains all non-default queue shard clients.
	queueShardClients map[string]QueueShard
	shardSelector     ShardSelector

	ppf                   PartitionPriorityFinder
	apf                   AccountPriorityFinder
	partitionPausedGetter PartitionPausedGetter

	lifecycles QueueLifecycleListeners

	allowKeyQueues                  AllowKeyQueues
	partitionConstraintConfigGetter PartitionConstraintConfigGetter

	activeCheckTick               time.Duration
	activeCheckAccountConcurrency int64
	activeCheckBacklogConcurrency int64
	activeCheckScanBatchSize      int64

	activeCheckAccountProbability int
	activeSpotCheckProbability    ActiveSpotChecksProbability
	readOnlySpotChecks            ReadOnlySpotChecks

	shadowPartitionProcessCount QueueShadowPartitionProcessCount

	tenantInstrumentor TenantInstrumentor

	// idempotencyTTL is the default or static idempotency duration apply to jobs,
	// if idempotencyTTLFunc is not defined.
	idempotencyTTL time.Duration
	// idempotencyTTLFunc returns an time.Duration representing how long job IDs
	// remain idempotent.
	idempotencyTTLFunc func(context.Context, osqueue.QueueItem) time.Duration
	// pollTick is the interval between each scan for jobs.
	pollTick                 time.Duration
	shadowPollTick           time.Duration
	backlogNormalizePollTick time.Duration
	// numWorkers stores the number of workers available to concurrently process jobs.
	numWorkers int32
	// numShadowWorkers stores the number of workers available to concurrently scan partitions
	numShadowWorkers int32
	// numBacklogNormalizationWorkers stores the maximum number of workers available to concurrenctly scan normalization partitions
	numBacklogNormalizationWorkers int32
	// peek min & max sets the range for partitions to peek for items
	peekMin int64
	peekMax int64
	// usePeekEWMA specifies whether we should use EWMA for peeking.
	usePeekEWMA bool
	// peekCurrMultiplier is a multiplier used for calculating the dynamic peek size
	// based on the EWMA values
	peekCurrMultiplier int64
	// peekEWMALen is the size of the list to hold the most recent values
	peekEWMALen int
	// queueKindMapping stores a map of job kind => queue names
	queueKindMapping        map[string]string
	disableFifoForFunctions map[string]struct{}
	disableFifoForAccounts  map[string]struct{}
	peekSizeForFunctions    map[string]int64
	log                     logger.Logger

	// itemIndexer returns indexes for a given queue item.
	itemIndexer QueueItemIndexer

	// denyQueues provides a denylist ensuring that the queue will never claim
	// this partition, meaning that no jobs from this queue will run on this worker.
	denyQueues        []string
	denyQueueMap      map[string]*struct{}
	denyQueuePrefixes map[string]*struct{}

	// allowQueues provides an allowlist, ensuring that the queue only peeks the specified
	// partitions.  jobs from other partitions will never be scanned or processed.
	allowQueues   []string
	allowQueueMap map[string]*struct{}
	// allowQueuePrefixes are memoized prefixes that can be allowed.
	allowQueuePrefixes map[string]*struct{}

	// instrumentInterval represents the frequency and instrumentation will attempt to run
	instrumentInterval time.Duration

	// backoffFunc is the backoff function to use when retrying operations.
	backoffFunc backoff.BackoffFunc

	clock clockwork.Clock

	// runMode defines the processing scopes or capabilities of the queue instances
	runMode QueueRunMode

	continuationLimit uint

	shadowContinues             map[string]shadowContinuation
	shadowContinueCooldown      map[string]time.Time
	shadowContinuesLock         *sync.Mutex
	shadowContinuationLimit     uint
	shadowPeekMin               int64
	shadowPeekMax               int64
	backlogRefillLimit          int64
	backlogNormalizeConcurrency int64

	normalizeRefreshItemCustomConcurrencyKeys NormalizeRefreshItemCustomConcurrencyKeysFn
	refreshItemThrottle                       RefreshItemThrottleFn

	enableJobPromotion bool

	capacityManager             constraintapi.RolloutManager
	useConstraintAPI            constraintapi.UseConstraintAPIFn
	capacityLeaseExtendInterval time.Duration

	enableThrottleInstrumentation EnableThrottleInstrumentationFn
}

// ShardSelector returns a shard reference for the given queue item.
// This allows applying a policy to enqueue items to different queue shards.
type ShardSelector func(ctx context.Context, accountId uuid.UUID, queueName *string) (QueueShard, error)

func WithShardSelector(s ShardSelector) func(q *queue) {
	return func(q *queue) {
		q.shardSelector = s
	}
}

func WithPeekEWMA(on bool) func(q *queue) {
	return func(q *queue) {
		q.usePeekEWMA = on
	}
}

// PartitionConstraintConfigGetter returns the constraint configuration for a given partition
type PartitionConstraintConfigGetter func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig

// WithPartitionConstraintConfigGetter assigns a function that returns queue constraints for a given partition.
func WithPartitionConstraintConfigGetter(f PartitionConstraintConfigGetter) func(q *queue) {
	return func(q *queue) {
		q.partitionConstraintConfigGetter = f
	}
}

// AllowKeyQueues determines if key queues should be enabled for the account
type AllowKeyQueues func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool

func WithAllowKeyQueues(kq AllowKeyQueues) QueueOpt {
	return func(q *queue) {
		q.allowKeyQueues = kq
	}
}

// QueueShadowPartitionProcessCount determines how many times the shadow scanner
// continue to process a shadow partition's backlog.
// This helps with reducing churn on leases for the shadow partition and allow handling
// larger amount of backlogs if there are a ton of backlog due to keys
type QueueShadowPartitionProcessCount func(ctx context.Context, acctID uuid.UUID) int

func WithQueueShadowPartitionProcessCount(spc QueueShadowPartitionProcessCount) QueueOpt {
	return func(q *queue) {
		q.shadowPartitionProcessCount = spc
	}
}

type (
	NormalizeRefreshItemCustomConcurrencyKeysFn func(ctx context.Context, item *osqueue.QueueItem, existingKeys []state.CustomConcurrency, shadowPartition *QueueShadowPartition) ([]state.CustomConcurrency, error)
	RefreshItemThrottleFn                       func(ctx context.Context, item *osqueue.QueueItem) (*osqueue.Throttle, error)
)

func WithNormalizeRefreshItemCustomConcurrencyKeys(fn NormalizeRefreshItemCustomConcurrencyKeysFn) QueueOpt {
	return func(q *queue) {
		q.normalizeRefreshItemCustomConcurrencyKeys = fn
	}
}

func WithRefreshItemThrottle(fn RefreshItemThrottleFn) QueueOpt {
	return func(q *queue) {
		q.refreshItemThrottle = fn
	}
}

type (
	ActiveSpotChecksProbability func(ctx context.Context, acctID uuid.UUID) (backlogRefillCheckProbability int, accountSpotCheckProbability int)
	ReadOnlySpotChecks          func(ctx context.Context, acctID uuid.UUID) bool
)

func WithActiveSpotCheckProbability(fn ActiveSpotChecksProbability) QueueOpt {
	return func(q *queue) {
		q.activeSpotCheckProbability = fn
	}
}

func WithReadOnlySpotChecks(fn ReadOnlySpotChecks) QueueOpt {
	return func(q *queue) {
		q.readOnlySpotChecks = fn
	}
}

type TenantInstrumentor func(ctx context.Context, partitionID string) error

func WithTenantInstrumentor(fn TenantInstrumentor) QueueOpt {
	return func(q *queue) {
		q.tenantInstrumentor = fn
	}
}

func WithInstrumentInterval(t time.Duration) QueueOpt {
	return func(q *queue) {
		if t > 0 {
			q.instrumentInterval = t
		}
	}
}

func WithQueueShardClients(queueShards map[string]QueueShard) QueueOpt {
	return func(q *queue) {
		q.queueShardClients = queueShards
	}
}

func WithEnableJobPromotion(enable bool) QueueOpt {
	return func(q *queue) {
		q.enableJobPromotion = enable
	}
}

func WithCapacityManager(capacityManager constraintapi.RolloutManager) QueueOpt {
	return func(q *queue) {
		q.capacityManager = capacityManager
	}
}

func WithUseConstraintAPI(uca constraintapi.UseConstraintAPIFn) QueueOpt {
	return func(q *queue) {
		q.useConstraintAPI = uca
	}
}

func WithCapacityLeaseExtendInterval(interval time.Duration) QueueOpt {
	return func(q *queue) {
		q.capacityLeaseExtendInterval = interval
	}
}

type EnableThrottleInstrumentationFn func(ctx context.Context, accountID, fnID uuid.UUID) bool

func WithEnableThrottleInstrumentation(fn EnableThrottleInstrumentationFn) QueueOpt {
	return func(q *queue) {
		q.enableThrottleInstrumentation = fn
	}
}
