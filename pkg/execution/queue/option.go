package queue

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/backoff"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/trace/noop"
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

type QueueProcessorOpt func(q *queueProcessor)

func WithName(name string) QueueProcessorOpt {
	return func(q *queueProcessor) {
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
		q.PartitionPriorityFinder = ppf
	}
}

func WithPartitionPausedGetter(partitionPausedGetter PartitionPausedGetter) QueueOpt {
	return func(q *QueueOptions) {
		q.PartitionPausedGetter = partitionPausedGetter
	}
}

func WithAccountPriorityFinder(apf AccountPriorityFinder) QueueOpt {
	return func(q *QueueOptions) {
		q.AccountPriorityFinder = apf
	}
}

func WithIdempotencyTTL(t time.Duration) QueueOpt {
	return func(q *QueueOptions) {
		q.IdempotencyTTL = t
	}
}

// WithIdempotencyTTLFunc returns custom idempotecy durations given a QueueItem.
// This allows customization of the idempotency TTL based off of specific jobs.
func WithIdempotencyTTLFunc(f func(context.Context, QueueItem) time.Duration) QueueOpt {
	return func(q *QueueOptions) {
		q.IdempotencyTTLFunc = f
	}
}

func WithNumWorkers(n int32) QueueOpt {
	return func(q *QueueOptions) {
		q.numWorkers = n
	}
}

func WithShadowNumWorkers(n int32) QueueOpt {
	return func(q *QueueOptions) {
		q.numShadowWorkers = n
	}
}

func WithPeekSizeRange(min int64, max int64) QueueOpt {
	return func(q *QueueOptions) {
		if max > AbsoluteQueuePeekMax {
			max = AbsoluteQueuePeekMax
		}
		q.PeekMin = min
		q.PeekMax = max
	}
}

func WithShadowPeekSizeRange(min int64, max int64) QueueOpt {
	return func(q *QueueOptions) {
		if max > AbsoluteShadowPartitionPeekMax {
			max = AbsoluteShadowPartitionPeekMax
		}
		q.shadowPeekMin = min
		q.shadowPeekMax = max
	}
}

func WithBacklogRefillLimit(limit int64) QueueOpt {
	return func(q *QueueOptions) {
		q.backlogRefillLimit = limit
	}
}

func WithBacklogNormalizationConcurrency(limit int64) QueueOpt {
	return func(q *QueueOptions) {
		q.backlogNormalizeConcurrency = limit
	}
}

func WithPeekConcurrencyMultiplier(m int64) QueueOpt {
	return func(q *QueueOptions) {
		q.peekCurrMultiplier = m
	}
}

func WithPeekEWMALength(l int) QueueOpt {
	return func(q *QueueOptions) {
		q.PeekEWMALen = l
	}
}

// WithPollTick specifies the interval at which the queue will poll the backing store
// for available partitions.
func WithPollTick(t time.Duration) QueueOpt {
	return func(q *QueueOptions) {
		q.pollTick = t
	}
}

// WithShadowPollTick specifies the interval at which the queue will poll the backing store
// for available shadow partitions.
func WithShadowPollTick(t time.Duration) QueueOpt {
	return func(q *QueueOptions) {
		q.shadowPollTick = t
	}
}

// WithBacklogNormalizePollTick specifies the interval at which the queue will poll the backing store
// for available backlogs to normalize.
func WithBacklogNormalizePollTick(t time.Duration) QueueOpt {
	return func(q *QueueOptions) {
		q.backlogNormalizePollTick = t
	}
}

// WithActiveCheckPollTick specifies the interval at which the queue will poll the backing store
// for available backlogs to normalize.
func WithActiveCheckPollTick(t time.Duration) QueueOpt {
	return func(q *QueueOptions) {
		q.ActiveCheckTick = t
	}
}

// WithActiveCheckAccountProbability specifies the probability of processing accounts vs. backlogs during an active check run.
func WithActiveCheckAccountProbability(p int) QueueOpt {
	return func(q *QueueOptions) {
		q.ActiveCheckAccountProbability = p
	}
}

// WithActiveCheckAccountConcurrency specifies the number of accounts to be peeked and processed by the active checker in parallel
func WithActiveCheckAccountConcurrency(p int) QueueOpt {
	return func(q *QueueOptions) {
		if p > 0 {
			q.ActiveCheckAccountConcurrency = int64(p)
		}
	}
}

// WithActiveCheckBacklogConcurrency specifies the number of backlogs to be peeked and processed by the active checker in parallel
func WithActiveCheckBacklogConcurrency(p int) QueueOpt {
	return func(q *QueueOptions) {
		if p > 0 {
			q.ActiveCheckBacklogConcurrency = int64(p)
		}
	}
}

// WithActiveCheckScanBatchSize specifies the batch size for iterating over active sets
func WithActiveCheckScanBatchSize(p int) QueueOpt {
	return func(q *QueueOptions) {
		if p > 0 {
			q.ActiveCheckScanBatchSize = int64(p)
		}
	}
}

// WithDenyQueueNames specifies that the worker cannot select jobs from queue partitions
// within the given list of names.  This means that the worker will never work on jobs
// in the specified queues.
//
// NOTE: If this is set and this worker claims the sequential lease, there is no guarantee
// on latency or fairness in the denied queue partitions.
func WithDenyQueueNames(queues ...string) QueueOpt {
	return func(q *QueueOptions) {
		q.DenyQueues = queues
		q.DenyQueueMap = make(map[string]*struct{})
		q.DenyQueuePrefixes = make(map[string]*struct{})
		for _, i := range queues {
			q.DenyQueueMap[i] = &struct{}{}
			// If WithDenyQueueNames includes "user:*", trim the asterisc and use
			// this as a prefix match.
			if strings.HasSuffix(i, "*") {
				q.DenyQueuePrefixes[strings.TrimSuffix(i, "*")] = &struct{}{}
			}
		}
	}
}

// WithAllowQueueNames specifies that the worker can only select jobs from queue partitions
// within the given list of names.  This means that the worker will never work on jobs in
// other queues.
func WithAllowQueueNames(queues ...string) QueueOpt {
	return func(q *QueueOptions) {
		q.AllowQueues = queues
		q.AllowQueueMap = make(map[string]*struct{})
		q.AllowQueuePrefixes = make(map[string]*struct{})
		for _, i := range queues {
			q.AllowQueueMap[i] = &struct{}{}
			// If WithAllowQueueNames includes "user:*", trim the asterisc and use
			// this as a prefix match.
			if strings.HasSuffix(i, "*") {
				q.AllowQueuePrefixes[strings.TrimSuffix(i, "*")] = &struct{}{}
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
	return func(q *QueueOptions) {
		q.queueKindMapping = mapping
	}
}

func WithDisableFifoForFunctions(mapping map[string]struct{}) QueueOpt {
	return func(q *QueueOptions) {
		q.disableFifoForFunctions = mapping
	}
}

func WithPeekSizeForFunction(mapping map[string]int64) QueueOpt {
	return func(q *QueueOptions) {
		q.peekSizeForFunctions = mapping
	}
}

func WithDisableFifoForAccounts(mapping map[string]struct{}) QueueOpt {
	return func(q *QueueOptions) {
		q.disableFifoForAccounts = mapping
	}
}

func WithLogger(l logger.Logger) QueueOpt {
	return func(q *QueueOptions) {
		q.log = l
	}
}

func WithBackoffFunc(f backoff.BackoffFunc) QueueOpt {
	return func(q *QueueOptions) {
		q.backoffFunc = f
	}
}

func WithRunMode(m QueueRunMode) QueueOpt {
	return func(q *QueueOptions) {
		q.runMode = m
	}
}

// WithClock allows replacing the queue's default (real) clock by a mock, for testing.
func WithClock(c clockwork.Clock) QueueOpt {
	return func(q *QueueOptions) {
		q.Clock = c
	}
}

// WithQueueContinuationLimit sets the continuation limit in the queue, eg. how many
// sequential steps cause hints in the queue to continue executing the same partition.
func WithQueueContinuationLimit(limit uint) QueueOpt {
	return func(q *QueueOptions) {
		q.continuationLimit = limit
	}
}

// WithContinuationSkipProbability sets the probability (0.0–1.0) that
// scanContinuations skips processing on any given scan tick. The default is
// consts.QueueContinuationSkipProbability (0.2), which spreads load across
// production replicas but adds unnecessary latency in single-instance dev servers.
func WithContinuationSkipProbability(p float64) QueueOpt {
	return func(q *QueueOptions) {
		q.continuationSkipProbability = p
	}
}

// WithQueueShadowContinuationLimit sets the shadow continuation limit in the queue, eg. how many
// sequential steps cause hints in the queue to continue executing the same shadow partition.
func WithQueueShadowContinuationLimit(limit uint) QueueOpt {
	return func(q *QueueOptions) {
		q.shadowContinuationLimit = limit
	}
}

type QueueRunMode struct {
	// Sequential determines whether Run() instance acquires sequential lease and processes items sequentially if lease is granted
	Sequential bool

	// Scavenger determines whether scavenger lease is acquired and scavenger is processed if lease is granted
	Scavenger bool

	// Partition determines whether partitions are processed
	Partition bool

	// Account determines whether accounts are processed
	Account bool

	// AccountWeight is the weight of processing accounts over partitions between 0 - 100 where 100 means only process accounts
	AccountWeight int

	// Continuations enables continuations
	Continuations bool

	// Shadow enables shadow partition processing
	ShadowPartition bool

	// AccountShadowPartition enables scanning of accounts for fair shadow partition processing
	AccountShadowPartition bool

	// AccountShadowPartitionWeight is the weight of processing accounts over global shadow partitions between 0 - 100 where 100 means only process accounts
	AccountShadowPartitionWeight int

	// ShadowContinuations enables shadow continuations
	ShadowContinuations bool

	// ShadowContinuationSkipProbability represents the probability to skip continuations (defaults to 0.2)
	ShadowContinuationSkipProbability float64

	// NormalizePartition enables the processing of partitions for normalization
	NormalizePartition bool

	// ActiveChecker enables background checking of active sets.
	ActiveChecker bool

	// ExclusiveAccounts defines a list of account IDs to peek exclusively.
	// This can be used to configure executors processing only a static subset of accounts.
	ExclusiveAccounts []uuid.UUID
}

type QueueOptions struct {
	PartitionPriorityFinder PartitionPriorityFinder
	AccountPriorityFinder   AccountPriorityFinder
	PartitionPausedGetter   PartitionPausedGetter

	lifecycles QueueLifecycleListeners

	AllowKeyQueues                  AllowKeyQueues
	PartitionConstraintConfigGetter PartitionConstraintConfigGetter

	ActiveCheckTick               time.Duration
	ActiveCheckAccountConcurrency int64
	ActiveCheckBacklogConcurrency int64
	ActiveCheckScanBatchSize      int64

	ActiveCheckAccountProbability int
	ActiveSpotCheckProbability    ActiveSpotChecksProbability
	ReadOnlySpotChecks            ReadOnlySpotChecks

	shadowPartitionProcessCount QueueShadowPartitionProcessCount

	TenantInstrumentor TenantInstrumentor

	// IdempotencyTTL is the default or static idempotency duration apply to jobs,
	// if idempotencyTTLFunc is not defined.
	IdempotencyTTL time.Duration
	// IdempotencyTTLFunc returns an time.Duration representing how long job IDs
	// remain idempotent.
	IdempotencyTTLFunc func(context.Context, QueueItem) time.Duration
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
	PeekMin int64
	PeekMax int64
	// usePeekEWMA specifies whether we should use EWMA for peeking.
	usePeekEWMA bool
	// peekCurrMultiplier is a multiplier used for calculating the dynamic peek size
	// based on the EWMA values
	peekCurrMultiplier int64
	// PeekEWMALen is the size of the list to hold the most recent values
	PeekEWMALen int
	// queueKindMapping stores a map of job kind => queue names
	queueKindMapping        map[string]string
	disableFifoForFunctions map[string]struct{}
	disableFifoForAccounts  map[string]struct{}
	peekSizeForFunctions    map[string]int64
	log                     logger.Logger

	// DenyQueues provides a denylist ensuring that the queue will never claim
	// this partition, meaning that no jobs from this queue will run on this worker.
	DenyQueues        []string
	DenyQueueMap      map[string]*struct{}
	DenyQueuePrefixes map[string]*struct{}

	// AllowQueues provides an allowlist, ensuring that the queue only peeks the specified
	// partitions.  jobs from other partitions will never be scanned or processed.
	AllowQueues   []string
	AllowQueueMap map[string]*struct{}
	// AllowQueuePrefixes are memoized prefixes that can be allowed.
	AllowQueuePrefixes map[string]*struct{}

	// instrumentInterval represents the frequency and instrumentation will attempt to run
	instrumentInterval time.Duration

	// backoffFunc is the backoff function to use when retrying operations.
	backoffFunc backoff.BackoffFunc

	Clock clockwork.Clock

	// runMode defines the processing scopes or capabilities of the queue instances
	runMode QueueRunMode

	continuationLimit           uint
	continuationSkipProbability float64

	shadowContinuationLimit uint

	shadowPeekMin               int64
	shadowPeekMax               int64
	backlogRefillLimit          int64
	backlogNormalizeConcurrency int64

	NormalizeRefreshItemCustomConcurrencyKeys NormalizeRefreshItemCustomConcurrencyKeysFn
	RefreshItemThrottle                       RefreshItemThrottleFn

	enableJobPromotion bool

	CapacityManager                    constraintapi.RolloutManager
	UseConstraintAPI                   constraintapi.UseConstraintAPIFn
	EnableCapacityLeaseInstrumentation constraintapi.EnableHighCardinalityInstrumentation
	CapacityLeaseExtendInterval        time.Duration

	EnableThrottleInstrumentation EnableThrottleInstrumentationFn

	ConditionalTracer trace.ConditionalTracer
}

// ShardSelector returns a shard reference for the given queue item.
// This allows applying a policy to enqueue items to different queue shards.
type ShardSelector func(ctx context.Context, accountId uuid.UUID, queueName *string) (QueueShard, error)

func WithPeekEWMA(on bool) QueueOpt {
	return func(q *QueueOptions) {
		q.usePeekEWMA = on
	}
}

// PartitionConstraintConfigGetter returns the constraint configuration for a given partition
type PartitionConstraintConfigGetter func(ctx context.Context, p PartitionIdentifier) PartitionConstraintConfig

// WithPartitionConstraintConfigGetter assigns a function that returns queue constraints for a given partition.
func WithPartitionConstraintConfigGetter(f PartitionConstraintConfigGetter) QueueOpt {
	return func(q *QueueOptions) {
		q.PartitionConstraintConfigGetter = f
	}
}

// AllowKeyQueues determines if key queues should be enabled for the account
type AllowKeyQueues func(ctx context.Context, acctID uuid.UUID, fnID uuid.UUID) bool

func WithAllowKeyQueues(kq AllowKeyQueues) QueueOpt {
	return func(q *QueueOptions) {
		q.AllowKeyQueues = kq
	}
}

// QueueShadowPartitionProcessCount determines how many times the shadow scanner
// continue to process a shadow partition's backlog.
// This helps with reducing churn on leases for the shadow partition and allow handling
// larger amount of backlogs if there are a ton of backlog due to keys
type QueueShadowPartitionProcessCount func(ctx context.Context, acctID uuid.UUID) int

func WithQueueShadowPartitionProcessCount(spc QueueShadowPartitionProcessCount) QueueOpt {
	return func(q *QueueOptions) {
		q.shadowPartitionProcessCount = spc
	}
}

type (
	NormalizeRefreshItemCustomConcurrencyKeysFn func(ctx context.Context, item *QueueItem, existingKeys []state.CustomConcurrency, shadowPartition *QueueShadowPartition) ([]state.CustomConcurrency, error)
	RefreshItemThrottleFn                       func(ctx context.Context, item *QueueItem) (*Throttle, error)
)

func WithNormalizeRefreshItemCustomConcurrencyKeys(fn NormalizeRefreshItemCustomConcurrencyKeysFn) QueueOpt {
	return func(q *QueueOptions) {
		q.NormalizeRefreshItemCustomConcurrencyKeys = fn
	}
}

func WithRefreshItemThrottle(fn RefreshItemThrottleFn) QueueOpt {
	return func(q *QueueOptions) {
		q.RefreshItemThrottle = fn
	}
}

type (
	ActiveSpotChecksProbability func(ctx context.Context, acctID uuid.UUID) (backlogRefillCheckProbability int, accountSpotCheckProbability int)
	ReadOnlySpotChecks          func(ctx context.Context, acctID uuid.UUID) bool
)

func WithActiveSpotCheckProbability(fn ActiveSpotChecksProbability) QueueOpt {
	return func(q *QueueOptions) {
		q.ActiveSpotCheckProbability = fn
	}
}

func WithReadOnlySpotChecks(fn ReadOnlySpotChecks) QueueOpt {
	return func(q *QueueOptions) {
		q.ReadOnlySpotChecks = fn
	}
}

type TenantInstrumentor func(ctx context.Context, partitionID string) error

func WithTenantInstrumentor(fn TenantInstrumentor) QueueOpt {
	return func(q *QueueOptions) {
		q.TenantInstrumentor = fn
	}
}

func WithInstrumentInterval(t time.Duration) QueueOpt {
	return func(q *QueueOptions) {
		if t > 0 {
			q.instrumentInterval = t
		}
	}
}

func WithEnableJobPromotion(enable bool) QueueOpt {
	return func(q *QueueOptions) {
		q.enableJobPromotion = enable
	}
}

func WithCapacityManager(capacityManager constraintapi.RolloutManager) QueueOpt {
	return func(q *QueueOptions) {
		q.CapacityManager = capacityManager
	}
}

func WithUseConstraintAPI(uca constraintapi.UseConstraintAPIFn) QueueOpt {
	return func(q *QueueOptions) {
		q.UseConstraintAPI = uca
	}
}

func WithCapacityLeaseExtendInterval(interval time.Duration) QueueOpt {
	return func(q *QueueOptions) {
		q.CapacityLeaseExtendInterval = interval
	}
}

func WithCapacityLeaseInstrumentation(enable constraintapi.EnableHighCardinalityInstrumentation) QueueOpt {
	return func(q *QueueOptions) {
		q.EnableCapacityLeaseInstrumentation = enable
	}
}

type EnableThrottleInstrumentationFn func(ctx context.Context, accountID, fnID uuid.UUID) bool

func WithEnableThrottleInstrumentation(fn EnableThrottleInstrumentationFn) QueueOpt {
	return func(q *QueueOptions) {
		q.EnableThrottleInstrumentation = fn
	}
}

func WithConditionalTracer(tracer trace.ConditionalTracer) QueueOpt {
	return func(q *QueueOptions) {
		q.ConditionalTracer = tracer
	}
}

// continuation represents a partition continuation, forcung the queue to continue working
// on a partition once a job from a partition has been processed.
type continuation struct {
	partition *QueuePartition
	// count is stored and incremented each time the partition is enqueued.
	count uint
}

// ShadowContinuation is the equivalent of continuation for shadow partitions
type ShadowContinuation struct {
	ShadowPart *QueueShadowPartition
	Count      uint
}

// ProcessItem references the queue partition and queue item to be processed by a worker.
// both items need to be passed to a worker as both items are needed to generate concurrency
// keys to extend leases and dequeue.
type ProcessItem struct {
	P QueuePartition
	I QueueItem

	// PCtr represents the number of times the partition has been continued.
	PCtr uint

	CapacityLease *CapacityLease

	// DisableConstraintUpdates determines whether ExtendLease, Requeue,
	// and Dequeue should update constraint state.
	//
	// Disable constraint updates in case
	// - we are processing an item for a system queue
	// - we are holding an active capacity lease
	//
	// For system queues, we skip constraint checks + updates entirely,
	// for regular functions we manage constraint checks + updates in the Constraint API,
	// if enabled for the current account.
	//
	// If the Constraint API is disabled or the lease expired, we will manage constraint state internally.
	//
	// NOTE: This value is set in itemLeaseConstraintCheck.
	DisableConstraintUpdates bool
}

type capacityLease struct {
	currentCapacityLeaseID *ulid.ULID
	capacityLeaseLock      sync.Mutex
}

func newCapacityLease(initialLease *CapacityLease) *capacityLease {
	cl := &capacityLease{
		capacityLeaseLock: sync.Mutex{},
	}
	if initialLease != nil {
		cl.currentCapacityLeaseID = &initialLease.LeaseID
	}

	return cl
}

func (p *capacityLease) set(leaseID *ulid.ULID) {
	p.capacityLeaseLock.Lock()
	defer p.capacityLeaseLock.Unlock()
	p.currentCapacityLeaseID = leaseID
}

func (p *capacityLease) get() *ulid.ULID {
	p.capacityLeaseLock.Lock()
	defer p.capacityLeaseLock.Unlock()
	return p.currentCapacityLeaseID
}

func (p *capacityLease) has() bool {
	p.capacityLeaseLock.Lock()
	defer p.capacityLeaseLock.Unlock()
	return p.currentCapacityLeaseID != nil
}

// FnMetadata is stored within the queue for retrieving
type FnMetadata struct {
	// NOTE: This is not encoded via JSON as we should always have the function
	// ID prior to doing a lookup, or should be able to retrieve the function ID
	// via the key.
	FnID uuid.UUID `json:"fnID"`

	// Paused represents whether the fn is paused.  This allows us to prevent leases
	// to a given partition if the partition belongs to a fn.
	Paused bool `json:"off"`

	// Migrate indicates if this queue is to be migrated or not
	Migrate bool `json:"migrate"`
}

type PartitionIdentifier struct {
	SystemQueueName *string
	FunctionID      uuid.UUID
	AccountID       uuid.UUID
	EnvID           uuid.UUID
}

func NewQueueOptions(
	options ...QueueOpt,
) *QueueOptions {
	o := &QueueOptions{
		PartitionPriorityFinder: func(_ context.Context, _ QueuePartition) uint {
			return PriorityDefault
		},
		AccountPriorityFinder: func(_ context.Context, _ uuid.UUID) uint {
			return PriorityDefault
		},
		PartitionPausedGetter: func(ctx context.Context, fnID uuid.UUID) PartitionPausedInfo {
			return PartitionPausedInfo{}
		},
		PeekMin:                     DefaultQueuePeekMin,
		PeekMax:                     DefaultQueuePeekMax,
		shadowPeekMin:               ShadowPartitionPeekMinBacklogs,
		shadowPeekMax:               ShadowPartitionPeekMaxBacklogs,
		backlogRefillLimit:          BacklogRefillHardLimit,
		backlogNormalizeConcurrency: defaultBacklogNormalizeConcurrency,
		runMode: QueueRunMode{
			Sequential:                        true,
			Scavenger:                         true,
			Partition:                         true,
			Account:                           true,
			AccountWeight:                     85,
			ShadowPartition:                   true,
			AccountShadowPartition:            true,
			AccountShadowPartitionWeight:      85,
			NormalizePartition:                true,
			ShadowContinuationSkipProbability: consts.QueueContinuationSkipProbability,
		},
		numWorkers:                     defaultNumWorkers,
		numShadowWorkers:               defaultNumShadowWorkers,
		numBacklogNormalizationWorkers: defaultBacklogNormalizationWorkers,
		pollTick:                       defaultPollTick,
		shadowPollTick:                 defaultShadowPollTick,
		backlogNormalizePollTick:       defaultBacklogNormalizePollTick,
		ActiveCheckTick:                defaultActiveCheckTick,
		IdempotencyTTL:                 defaultIdempotencyTTL,
		queueKindMapping:               make(map[string]string),
		peekSizeForFunctions:           make(map[string]int64),
		instrumentInterval:             DefaultInstrumentInterval,
		PartitionConstraintConfigGetter: func(ctx context.Context, pi PartitionIdentifier) PartitionConstraintConfig {
			def := DefaultConcurrency

			return PartitionConstraintConfig{
				Concurrency: PartitionConcurrency{
					AccountConcurrency:  def,
					FunctionConcurrency: def,
				},
			}
		},
		AllowKeyQueues: func(ctx context.Context, acctID, fnID uuid.UUID) bool {
			return false
		},
		shadowPartitionProcessCount: func(ctx context.Context, acctID uuid.UUID) int {
			return 5
		},
		TenantInstrumentor: func(ctx context.Context, partitionID string) error {
			return nil
		},
		backoffFunc:       backoff.DefaultBackoff,
		Clock:             clockwork.NewRealClock(),
		continuationLimit:           consts.DefaultQueueContinueLimit,
		continuationSkipProbability: consts.QueueContinuationSkipProbability,
		NormalizeRefreshItemCustomConcurrencyKeys: func(ctx context.Context, item *QueueItem, existingKeys []state.CustomConcurrency, shadowPartition *QueueShadowPartition) ([]state.CustomConcurrency, error) {
			return existingKeys, nil
		},
		RefreshItemThrottle: func(ctx context.Context, item *QueueItem) (*Throttle, error) {
			return nil, nil
		},
		ReadOnlySpotChecks: func(ctx context.Context, acctID uuid.UUID) bool {
			return true
		},
		ActiveSpotCheckProbability: func(ctx context.Context, acctID uuid.UUID) (backlogRefillCheckProbability int, accountSpotCheckProbability int) {
			return 100, 100
		},
		ActiveCheckAccountProbability: 10,
		ActiveCheckAccountConcurrency: ActiveCheckAccountConcurrency,
		ActiveCheckBacklogConcurrency: ActiveCheckBacklogConcurrency,
		ActiveCheckScanBatchSize:      ActiveCheckScanBatchSize,
		CapacityLeaseExtendInterval:   QueueLeaseDuration / 2,
		ConditionalTracer: trace.NewConditionalTracer(noop.Tracer{}, func(ctx context.Context, accountID, envID uuid.UUID) bool {
			return false
		}),
		EnableCapacityLeaseInstrumentation: func(ctx context.Context, accountID, envID, functionID uuid.UUID) (enable bool) {
			return false
		},
	}

	for _, qopt := range options {
		qopt(o)
	}
	return o
}
