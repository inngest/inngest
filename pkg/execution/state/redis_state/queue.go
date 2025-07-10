package redis_state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"math"
	mrand "math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/VividCortex/ewma"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/pkg/backoff"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"gonum.org/v1/gonum/stat/sampleuv"
)

const (
	PartitionSelectionMax = int64(100)
	PartitionPeekMax      = PartitionSelectionMax * 3
	AccountPeekMax        = int64(30)
)

const (
	pkgName = "redis_state.state.execution.inngest"

	// PartitionLeaseDuration dictates how long a worker holds the lease for
	// a partition.  This gives the worker a right to scan all queue items
	// for that partition to schedule the execution of jobs.
	//
	// Right now, this must be short enough to reduce contention but long enough
	// to account for the latency of peeking QueuePeekMax jobs from Redis.
	PartitionLeaseDuration = 4 * time.Second
	// PartitionRequeueExtension is the length of time that we extend a partition's
	// vesting time when requeueing by default.
	PartitionRequeueExtension = 30 * time.Second

	// PartitionConcurrencyLimitRequeueExtension is the length of time that a partition
	// is requeued if there is no global or partition(function) capacity because of
	// concurrency limits.
	//
	// This is short, as there are still functions that are due to run (ie vesting < now),
	// but long enough to reduce thrash.
	//
	// This means that jobs not started because of concurrency limits incur up to this amount
	// of additional latency.
	//
	// NOTE: This must be greater than PartitionLookahead
	// NOTE: This is the maximum latency introduced into concurrnecy limited partitions in the
	//       worst case.
	PartitionConcurrencyLimitRequeueExtension = 5 * time.Second
	PartitionThrottleLimitRequeueExtension    = 1 * time.Second
	PartitionPausedRequeueExtension           = 24 * time.Hour
	PartitionLookahead                        = time.Second

	ShadowPartitionLeaseDuration  = 4 * time.Second // same as PartitionLeaseDuration
	BacklogNormalizeLeaseDuration = 4 * time.Second // same as PartitionLeaseDuration

	ShadowPartitionRefillCapacityReachedRequeueExtension = 1 * time.Second
	ShadowPartitionRefillPausedRequeueExtension          = 24 * time.Hour
	BacklogDefaultRequeueExtension                       = 2 * time.Second

	// default values
	DefaultQueuePeekMin  int64 = 300
	DefaultQueuePeekMax  int64 = 750
	AbsoluteQueuePeekMax int64 = 5000

	QueuePeekCurrMultiplier int64 = 4 // threshold 25%
	QueuePeekEWMALen        int   = 10
	QueueLeaseDuration            = 30 * time.Second
	ConfigLeaseDuration           = 10 * time.Second
	ConfigLeaseMax                = 20 * time.Second

	ScavengePeekSize                 = 100
	ScavengeConcurrencyQueuePeekSize = 100

	PriorityMax     uint = 0
	PriorityDefault uint = 5
	PriorityMin     uint = 9

	// FunctionStartScoreBufferTime is the grace period used to compare function start
	// times to edge enqueue times.
	FunctionStartScoreBufferTime = 10 * time.Second

	defaultNumWorkers                  = 100
	defaultNumShadowWorkers            = 100
	defaultBacklogNormalizationWorkers = 10
	defaultBacklogNormalizeLimit       = int64(500)

	defaultPollTick                 = 10 * time.Millisecond
	defaultShadowPollTick           = 100 * time.Millisecond
	defaultBacklogNormalizePollTick = 250 * time.Millisecond
	defaultActiveCheckTick          = 10 * time.Second

	defaultIdempotencyTTL = 12 * time.Hour
	defaultConcurrency    = 1000 // TODO: add function to override.

	NoConcurrencyLimit = -1
)

var (
	ErrQueueItemExists               = fmt.Errorf("queue item already exists")
	ErrQueueItemNotFound             = fmt.Errorf("queue item not found")
	ErrQueueItemAlreadyLeased        = fmt.Errorf("queue item already leased")
	ErrQueueItemLeaseMismatch        = fmt.Errorf("item lease does not match")
	ErrQueueItemNotLeased            = fmt.Errorf("queue item is not leased")
	ErrQueuePeekMaxExceedsLimits     = fmt.Errorf("peek exceeded the maximum limit of %d", AbsoluteQueuePeekMax)
	ErrQueueItemSingletonExists      = fmt.Errorf("singleton item already exists")
	ErrPriorityTooLow                = fmt.Errorf("priority is too low")
	ErrPriorityTooHigh               = fmt.Errorf("priority is too high")
	ErrPartitionNotFound             = fmt.Errorf("partition not found")
	ErrPartitionAlreadyLeased        = fmt.Errorf("partition already leased")
	ErrPartitionPeekMaxExceedsLimits = fmt.Errorf("peek exceeded the maximum limit of %d", PartitionPeekMax)
	ErrAccountPeekMaxExceedsLimits   = fmt.Errorf("account peek exceeded the maximum limit of %d", AccountPeekMax)
	ErrPartitionGarbageCollected     = fmt.Errorf("partition garbage collected")
	ErrPartitionPaused               = fmt.Errorf("partition is paused")
	ErrConfigAlreadyLeased           = fmt.Errorf("config scanner already leased")
	ErrConfigLeaseExceedsLimits      = fmt.Errorf("config lease duration exceeds the maximum of %d seconds", int(ConfigLeaseMax.Seconds()))

	ErrPartitionConcurrencyLimit = fmt.Errorf("at partition concurrency limit")
	ErrAccountConcurrencyLimit   = fmt.Errorf("at account concurrency limit")

	// ErrSystemConcurrencyLimit represents a concurrency limit for system partitions
	ErrSystemConcurrencyLimit = fmt.Errorf("at system concurrency limit")

	// ErrConcurrencyLimitCustomKey represents a concurrency limit being hit for *some*, but *not all*
	// jobs in a queue, via custom concurrency keys which are evaluated to a specific string.
	ErrConcurrencyLimitCustomKey = fmt.Errorf("at concurrency limit")
)

var rnd *util.FrandRNG

func init() {
	// For weighted shuffles generate a new rand.
	rnd = util.NewFrandRNG()
}

type QueueManager interface {
	osqueue.JobQueueReader
	osqueue.Queue
	osqueue.QueueDirectAccess

	Dequeue(ctx context.Context, queueShard QueueShard, i osqueue.QueueItem) error
	Requeue(ctx context.Context, queueShard QueueShard, i osqueue.QueueItem, at time.Time) error
	RequeueByJobID(ctx context.Context, queueShard QueueShard, jobID string, at time.Time) error

	// ItemsByPartition returns a queue item iterator for a function within a specific time range
	ItemsByPartition(ctx context.Context, queueShard QueueShard, partitionID uuid.UUID, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*osqueue.QueueItem], error)

	// ItemsByBacklog returns a queue item iterator for a backlog within a specific time range
	ItemsByBacklog(ctx context.Context, queueShard QueueShard, backlogID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*osqueue.QueueItem], error)

	PartitionByID(ctx context.Context, queueShard QueueShard, partitionID string) (*PartitionInspectionResult, error)
}

// PartitionPriorityFinder returns the priority for a given queue partition.
type PartitionPriorityFinder func(ctx context.Context, part QueuePartition) uint

// AccountPriorityFinder returns the priority for a given account.
type AccountPriorityFinder func(ctx context.Context, accountId uuid.UUID) uint

type QueueOpt func(q *queue)

func WithName(name string) func(q *queue) {
	return func(q *queue) {
		q.name = name
	}
}

func WithQueueLifecycles(l ...QueueLifecycleListener) QueueOpt {
	return func(q *queue) {
		q.lifecycles = l
	}
}

func WithPartitionPriorityFinder(ppf PartitionPriorityFinder) QueueOpt {
	return func(q *queue) {
		q.ppf = ppf
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

func WithBacklogNormalizationLimit(limit int64) QueueOpt {
	return func(q *queue) {
		q.backlogNormalizeLimit = limit
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

// WithCustomConcurrencyKeyLimitRefresher assigns a function that returns concurrency keys with
// current limits for a given queue item, eg. a step in a function.
func WithCustomConcurrencyKeyLimitRefresher(f QueueItemConcurrencyKeyLimitRefresher) func(q *queue) {
	return func(q *queue) {
		q.customConcurrencyLimitRefresher = f
	}
}

// WithConcurrencyLimitGetter assigns a function that returns concurrency limits
// for a given partition.
func WithConcurrencyLimitGetter(f ConcurrencyLimitGetter) func(q *queue) {
	return func(q *queue) {
		q.concurrencyLimitGetter = func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
			limits := f(ctx, p)
			// Always clip limits for accounts to impose _some_ limit.
			if limits.AccountLimit <= 0 {
				limits.AccountLimit = consts.DefaultConcurrencyLimit
			}
			return limits
		}
	}
}

// WithConcurrencyLimitGetter assigns a function that returns concurrency limits
// for a given partition.
func WithSystemConcurrencyLimitGetter(f SystemConcurrencyLimitGetter) func(q *queue) {
	return func(q *queue) {
		q.systemConcurrencyLimitGetter = f
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

type QueueShard struct {
	Name string
	Kind string

	RedisClient *QueueClient
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

// QueueItemConcurrencyKeyLimitRefresher returns concurrency keys with current limits given a queue item.
//
// Each queue item can have its own concurrency keys.  For example, you can define
// concurrency limits for steps within a function.  This ensures that there will never be
// more than N concurrent items running at once.
type QueueItemConcurrencyKeyLimitRefresher func(ctx context.Context, i osqueue.QueueItem) []state.CustomConcurrency

type PartitionConcurrencyLimits struct {
	// AccountLimit returns the current account concurrency limit, which is always applied. Defaults to maximum concurrency.
	AccountLimit int

	// FunctionLimit returns the function-scoped concurrency limit, if configured. Defaults to maximum concurrency.
	FunctionLimit int

	// CustomKeyLimit returns the custom concurrency limit for a concurrency key partition. Defaults to maximum concurrency.
	CustomKeyLimit int
}

type SystemPartitionConcurrencyLimits struct {
	// GlobalLimit returns the account-level equivalent concurrency limit for system partitions, which is always applied. Defaults to maximum concurrency.
	GlobalLimit int

	// PartitionLimit returns the partition-scoped concurrency limit, if configured. Defaults to maximum concurrency.
	PartitionLimit int
}

// ConcurrencyLimitGetter returns the fn, account, and custom limits for a given partition.
type ConcurrencyLimitGetter func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits

// SystemConcurrencyLimitGetter returns the concurrency limits for a given system partition.
type SystemConcurrencyLimitGetter func(ctx context.Context, p QueuePartition) SystemPartitionConcurrencyLimits

// PartitionConstraintConfigGetter returns the constraint configuration for a given partition
type PartitionConstraintConfigGetter func(ctx context.Context, p QueueShadowPartition) (*PartitionConstraintConfig, error)

// WithPartitionConstraintConfigGetter assigns a function that returns queue constraints for a given partition.
func WithPartitionConstraintConfigGetter(f PartitionConstraintConfigGetter) func(q *queue) {
	return func(q *queue) {
		q.partitionConstraintConfigGetter = f
	}
}

// AllowSystemKeyQueues determines if key queues should be enabled for system queues
type AllowSystemKeyQueues func(ctx context.Context) bool

// AllowKeyQueues determines if key queues should be enabled for the account
type AllowKeyQueues func(ctx context.Context, acctID uuid.UUID) bool

func WithAllowKeyQueues(kq AllowKeyQueues) QueueOpt {
	return func(q *queue) {
		q.allowKeyQueues = kq
	}
}

func WithEnqueueSystemPartitionsToBacklog(enqueueToBacklog bool) QueueOpt {
	return func(q *queue) {
		q.enqueueSystemQueuesToBacklog = enqueueToBacklog
	}
}

func WithDisableLeaseChecksForSystemQueues(disableChecks bool) QueueOpt {
	return func(q *queue) {
		q.disableLeaseChecksForSystemQueues = disableChecks
	}
}

// DisableLeaseChecks determines if existing lease checks on partition leasing and queue item
// leasing should be disabled or not
type DisableLeaseChecks func(ctx context.Context, acctID uuid.UUID) bool

type DisableSystemQueueLeaseChecks func(ctx context.Context) bool

func WithDisableLeaseChecks(lc DisableLeaseChecks) QueueOpt {
	return func(q *queue) {
		q.disableLeaseChecks = lc
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

type NormalizeRefreshItemCustomConcurrencyKeysFn func(ctx context.Context, item *osqueue.QueueItem, existingKeys []state.CustomConcurrency, shadowPartition *QueueShadowPartition) ([]state.CustomConcurrency, error)
type RefreshItemThrottleFn func(ctx context.Context, item *osqueue.QueueItem) (*osqueue.Throttle, error)

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

type ActiveSpotChecksProbability func(ctx context.Context, acctID uuid.UUID) (backlogRefillCheckProbability int, accountSpotCheckProbability int)
type ReadOnlySpotChecks func(ctx context.Context, acctID uuid.UUID) bool

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

func NewQueue(primaryQueueShard QueueShard, opts ...QueueOpt) *queue {
	ctx := context.Background()

	q := &queue{
		primaryQueueShard: primaryQueueShard,
		queueShardClients: map[string]QueueShard{primaryQueueShard.Name: primaryQueueShard},
		ppf: func(_ context.Context, _ QueuePartition) uint {
			return PriorityDefault
		},
		apf: func(_ context.Context, _ uuid.UUID) uint {
			return PriorityDefault
		},
		peekMin:               DefaultQueuePeekMin,
		peekMax:               DefaultQueuePeekMax,
		shadowPeekMin:         ShadowPartitionPeekMinBacklogs,
		shadowPeekMax:         ShadowPartitionPeekMaxBacklogs,
		backlogRefillLimit:    BacklogRefillHardLimit,
		backlogNormalizeLimit: defaultBacklogNormalizeLimit,
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
		wg:                             &sync.WaitGroup{},
		seqLeaseLock:                   &sync.RWMutex{},
		scavengerLeaseLock:             &sync.RWMutex{},
		activeCheckerLeaseLock:         &sync.RWMutex{},
		instrumentationLeaseLock:       &sync.RWMutex{},
		pollTick:                       defaultPollTick,
		shadowPollTick:                 defaultShadowPollTick,
		backlogNormalizePollTick:       defaultBacklogNormalizePollTick,
		activeCheckTick:                defaultActiveCheckTick,
		idempotencyTTL:                 defaultIdempotencyTTL,
		queueKindMapping:               make(map[string]string),
		peekSizeForFunctions:           make(map[string]int64),
		log:                            logger.StdlibLogger(ctx),
		partitionConstraintConfigGetter: func(ctx context.Context, p QueueShadowPartition) (*PartitionConstraintConfig, error) {
			def := defaultConcurrency

			return &PartitionConstraintConfig{
				Concurrency: ShadowPartitionConcurrency{
					AccountConcurrency:  def,
					FunctionConcurrency: def,
				},
			}, nil
		},
		concurrencyLimitGetter: func(ctx context.Context, p QueuePartition) PartitionConcurrencyLimits {
			def := defaultConcurrency
			if p.ConcurrencyLimit > 0 {
				def = p.ConcurrencyLimit
			}
			// Use the defaults, and add no concurrency limits to custom keys.
			limits := PartitionConcurrencyLimits{
				AccountLimit:   def,
				FunctionLimit:  def,
				CustomKeyLimit: -1,
			}
			if p.EvaluatedConcurrencyKey == "" {
				limits.CustomKeyLimit = NoConcurrencyLimit
			}
			return limits
		},
		systemConcurrencyLimitGetter: func(ctx context.Context, p QueuePartition) SystemPartitionConcurrencyLimits {
			def := defaultConcurrency
			if p.ConcurrencyLimit > 0 {
				def = p.ConcurrencyLimit
			}
			return SystemPartitionConcurrencyLimits{
				GlobalLimit:    def,
				PartitionLimit: def,
			}
		},
		customConcurrencyLimitRefresher: func(ctx context.Context, item osqueue.QueueItem) []state.CustomConcurrency {
			// No-op: Use whatever's in the queue item by default
			return item.Data.GetConcurrencyKeys()
		},
		allowKeyQueues: func(ctx context.Context, acctID uuid.UUID) bool {
			return false
		},
		enqueueSystemQueuesToBacklog: false,
		disableLeaseChecks: func(ctx context.Context, acctID uuid.UUID) bool {
			return false
		},
		disableLeaseChecksForSystemQueues: true,
		shadowPartitionProcessCount: func(ctx context.Context, acctID uuid.UUID) int {
			return 5
		},
		itemIndexer:             QueueItemIndexerFunc,
		backoffFunc:             backoff.DefaultBackoff,
		clock:                   clockwork.NewRealClock(),
		continuesLock:           &sync.Mutex{},
		continues:               map[string]continuation{},
		continueCooldown:        map[string]time.Time{},
		continuationLimit:       consts.DefaultQueueContinueLimit,
		shadowContinuesLock:     &sync.Mutex{},
		shadowContinuationLimit: consts.DefaultQueueContinueLimit,
		shadowContinues:         map[string]shadowContinuation{},
		shadowContinueCooldown:  map[string]time.Time{},
		normalizeRefreshItemCustomConcurrencyKeys: func(ctx context.Context, item *osqueue.QueueItem, existingKeys []state.CustomConcurrency, shadowPartition *QueueShadowPartition) ([]state.CustomConcurrency, error) {
			return existingKeys, nil
		},
		refreshItemThrottle: func(ctx context.Context, item *osqueue.QueueItem) (*osqueue.Throttle, error) {
			return nil, nil
		},
		readOnlySpotChecks: func(ctx context.Context, acctID uuid.UUID) bool {
			return true
		},
		activeSpotCheckProbability: func(ctx context.Context, acctID uuid.UUID) (backlogRefillCheckProbability int, accountSpotCheckProbability int) {
			return 100, 100
		},
		activeCheckAccountProbability: 10,
		activeCheckAccountConcurrency: ActiveCheckAccountConcurrency,
		activeCheckBacklogConcurrency: ActiveCheckBacklogConcurrency,
		activeCheckScanBatchSize:      ActiveCheckScanBatchSize,
	}

	// default to using primary queue client for shard selection
	q.shardSelector = func(_ context.Context, _ uuid.UUID, _ *string) (QueueShard, error) {
		return q.primaryQueueShard, nil
	}

	for _, opt := range opts {
		opt(q)
	}

	q.sem = &trackingSemaphore{Weighted: semaphore.NewWeighted(int64(q.numWorkers))}
	q.workers = make(chan processItem, q.numWorkers)

	return q
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

type queue struct {
	// name is the identifiable name for this worker, for logging.
	name string

	// primaryQueueShard stores the queue shard to use.
	primaryQueueShard QueueShard

	// queueShardClients contains all non-default queue shard clients.
	queueShardClients map[string]QueueShard
	shardSelector     ShardSelector

	ppf PartitionPriorityFinder
	apf AccountPriorityFinder

	lifecycles QueueLifecycleListeners

	concurrencyLimitGetter          ConcurrencyLimitGetter
	systemConcurrencyLimitGetter    SystemConcurrencyLimitGetter
	customConcurrencyLimitRefresher QueueItemConcurrencyKeyLimitRefresher

	allowKeyQueues                  AllowKeyQueues
	enqueueSystemQueuesToBacklog    bool
	partitionConstraintConfigGetter PartitionConstraintConfigGetter

	activeCheckTick               time.Duration
	activeCheckAccountConcurrency int64
	activeCheckBacklogConcurrency int64
	activeCheckScanBatchSize      int64

	activeCheckAccountProbability int
	activeSpotCheckProbability    ActiveSpotChecksProbability
	readOnlySpotChecks            ReadOnlySpotChecks
	// activeCheckerLeaseID stores the lease ID if this queue is the ActiveChecker processor.
	// all runners attempt to claim this lease automatically.
	activeCheckerLeaseID *ulid.ULID
	// activeCheckerLeaseLock ensures that there are no data races writing to
	// or reading from activeCheckerLeaseID in parallel.
	activeCheckerLeaseLock *sync.RWMutex

	disableLeaseChecks                DisableLeaseChecks
	disableLeaseChecksForSystemQueues bool

	shadowPartitionProcessCount QueueShadowPartitionProcessCount

	// idempotencyTTL is the default or static idempotency duration apply to jobs,
	// if idempotencyTTLFunc is not defined.
	idempotencyTTL time.Duration
	// idempotencyTTLFunc returns an time.Duration representing how long job IDs
	// remain idempotent.
	idempotencyTTLFunc func(context.Context, osqueue.QueueItem) time.Duration
	// pollTick is the interval between each scan for jobs.
	pollTick time.Duration
	// quit is a channel that any method can send on to trigger termination
	// of the Run loop.  This typically accepts an error, but a nil error
	// will still quit the runner.
	quit chan error
	// wg stores a waitgroup for all in-progress jobs
	wg *sync.WaitGroup
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
	// workers is a buffered channel which allows scanners to send queue items
	// to workers to be processed
	workers chan processItem
	// sem stores a semaphore controlling the number of jobs currently
	// being processed.  This lets us check whether there's capacity in the queue
	// prior to leasing items.
	sem *trackingSemaphore
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

	// scavengerLeaseID stores the lease ID if this queue is the scavenger processor.
	// all runners attempt to claim this lease automatically.
	scavengerLeaseID *ulid.ULID
	// scavengerLeaseLock ensures that there are no data races writing to
	// or reading from scavengerLeaseID in parallel.
	scavengerLeaseLock *sync.RWMutex

	// backoffFunc is the backoff function to use when retrying operations.
	backoffFunc backoff.BackoffFunc

	clock clockwork.Clock

	// runMode defines the processing scopes or capabilities of the queue instances
	runMode QueueRunMode

	// continues stores a map of all partition IDs to continues for a partition.
	// this lets us optimize running consecutive steps for a function, as a continuation, to a specific limit.
	continues        map[string]continuation
	continueCooldown map[string]time.Time

	// continuesLock protects the continues map.
	continuesLock     *sync.Mutex
	continuationLimit uint

	shadowPollTick           time.Duration
	backlogNormalizePollTick time.Duration

	shadowContinues         map[string]shadowContinuation
	shadowContinueCooldown  map[string]time.Time
	shadowContinuesLock     *sync.Mutex
	shadowContinuationLimit uint
	shadowPeekMin           int64
	shadowPeekMax           int64
	backlogRefillLimit      int64
	backlogNormalizeLimit   int64

	normalizeRefreshItemCustomConcurrencyKeys NormalizeRefreshItemCustomConcurrencyKeysFn
	refreshItemThrottle                       RefreshItemThrottleFn

	enableJobPromotion bool
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
}

// continuation represents a partition continuation, forcung the queue to continue working
// on a partition once a job from a partition has been processed.
type continuation struct {
	partition *QueuePartition
	// count is stored and incremented each time the partition is enqueued.
	count uint
}

// shadowContinuation is the equivalent of continuation for shadow partitions
type shadowContinuation struct {
	shadowPart *QueueShadowPartition
	count      uint
}

// processItem references the queue partition and queue item to be processed by a worker.
// both items need to be passed to a worker as both items are needed to generate concurrency
// keys to extend leases and dequeue.
type processItem struct {
	P QueuePartition
	I osqueue.QueueItem

	// PCtr represents the number of times the partition has been continued.
	PCtr uint
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

// QueuePartition represents an individual queue for a workflow.  It stores the
// time of the earliest job within the workflow.
type QueuePartition struct {
	// ID represents the key used within the global Partition hash and global pointer set
	// which represents this QueuePartition.  This is the function ID for enums.PartitionTypeDefault,
	// or the entire key returned from the key generator for other types.
	ID string `json:"id,omitempty"`
	// PartitionType is the int-value of the enums.PartitionType for this
	// partition.  By default, partitions are function-scoped without any
	// custom keys.
	PartitionType int `json:"pt,omitempty"`
	// QueueName is used for manually overriding queue items to be enqueued for
	// system jobs like pause events and timeouts, batch timeouts, and replays.
	//
	// NOTE: This field is required for backwards compatibility, as old system partitions
	// simply set the queue name.
	//
	// This should almost always be nil.
	QueueName *string `json:"queue,omitempty"`
	// ConcurrencyScope is the int-value representation of the enums.ConcurrencyScope,
	// if this is a concurrency-scoped partition.
	ConcurrencyScope int `json:"cs,omitempty"`
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

	//
	// Concurrency
	//

	// ConcurrencyLimit represents the max concurrency for the queue partition.  This allows
	// us to optimize the queue by checking for the max when leasing partitions
	// directly.
	//
	// This ALWAYS exists, even for function level partitions.
	ConcurrencyLimit int `json:"l,omitempty"`
	// EvaluatedConcurrencyKey represents the evaluated and hashed custom key for the queue partition, if this is
	// for a custom key.
	EvaluatedConcurrencyKey string `json:"ck,omitempty"`
	// UnevaluatedConcurrencyHash is the hashed but unevaluated custom key for the queue partition, if this is
	// for a custom key.
	//
	// This must be set so that we can fetch the latest concurrency limits dynamically when
	// leasing a partition, if desired, via the ConcurrencyLimitGetter.
	UnevaluatedConcurrencyHash string `json:"ch,omitempty"`
}

func (qp QueuePartition) IsSystem() bool {
	return qp.QueueName != nil && *qp.QueueName != ""
}

// zsetKey represents the key used to store the zset for this partition's items.
// For default partitions, this is different to the ID (for backwards compatibility, it's just
// the fn ID without prefixes)
func (qp QueuePartition) zsetKey(kg QueueKeyGenerator) string {
	// For system partitions, return zset using custom queueName
	if qp.IsSystem() {
		return kg.PartitionQueueSet(enums.PartitionTypeDefault, qp.Queue(), "")
	}

	// Backwards compatibility with old fn queues
	if qp.PartitionType == int(enums.PartitionTypeDefault) && qp.FunctionID != nil {
		// return the top-level function queue.
		return kg.PartitionQueueSet(enums.PartitionTypeDefault, qp.FunctionID.String(), "")
	}

	if qp.ID == "" {
		// return a blank queue key.  This is used for nil queue partitions.
		return kg.PartitionQueueSet(enums.PartitionTypeDefault, "-", "")
	}

	// qp.ID is already a properly defined key (concurrency key queues).
	return qp.ID
}

// concurrencyKey returns the single concurrency key for the given partition, depending
// on the partition type.  This is used to check the partition's in-progress items whilst
// requeueing partitions.
func (qp QueuePartition) concurrencyKey(kg QueueKeyGenerator) string {
	switch enums.PartitionType(qp.PartitionType) {
	case enums.PartitionTypeDefault:
		return qp.fnConcurrencyKey(kg)
	case enums.PartitionTypeConcurrencyKey:
		// Hierarchically, custom keys take precedence.
		return qp.customConcurrencyKey(kg)
	default:
		panic(fmt.Sprintf("unexpected partition type encountered in concurrencyKey %q", qp.PartitionType))
	}
}

// fnConcurrencyKey returns the concurrency key for a function scope limit, on the
// entire function (not custom keys)
func (qp QueuePartition) fnConcurrencyKey(kg QueueKeyGenerator) string {
	// Enable system partitions to use the queueName override instead of the fnId
	if qp.IsSystem() {
		return kg.Concurrency("p", qp.Queue())
	}

	if qp.FunctionID == nil {
		return kg.Concurrency("p", "-")
	}
	return kg.Concurrency("p", qp.FunctionID.String())
}

// acctConcurrencyKey returns the concurrency key for the account limit, on the
// entire account (not custom keys)
func (qp QueuePartition) acctConcurrencyKey(kg QueueKeyGenerator) string {
	// Enable system partitions to use the queueName override instead of the accountId
	if qp.IsSystem() {
		return kg.Concurrency("account", qp.Queue())
	}
	if qp.AccountID == uuid.Nil {
		return kg.Concurrency("account", "-")
	}
	return kg.Concurrency("account", qp.AccountID.String())
}

// customConcurrencyKey returns the concurrency key if this partition represents
// a custom concurrnecy limit.
func (qp QueuePartition) customConcurrencyKey(kg QueueKeyGenerator) string {
	// This should never happen, but we attempt to handle it gracefully
	if qp.IsSystem() {
		// this is consistent with the concrete WithCustomConcurrencyKeyGenerator in cloud previously
		return kg.Concurrency("custom", qp.Queue())
	}

	if qp.EvaluatedConcurrencyKey == "" {
		return kg.Concurrency("custom", "-")
	}
	return kg.Concurrency("custom", qp.EvaluatedConcurrencyKey)
}

func (qp QueuePartition) Queue() string {
	// This is redundant but acts as a safeguard, so that
	// we always return the ID (queueName) for system partitions
	if qp.IsSystem() {
		return *qp.QueueName
	}

	if qp.ID == "" && qp.FunctionID != nil {
		return qp.FunctionID.String()
	}

	return qp.ID
}

func (qp QueuePartition) MarshalBinary() ([]byte, error) {
	return json.Marshal(qp)
}

// ItemPartitions returns the partition for a given item.
func (q *queue) ItemPartition(ctx context.Context, shard QueueShard, i osqueue.QueueItem) (QueuePartition, int) {
	queueName := i.QueueName

	// sanity check: both QueueNames should be set, but sometimes aren't
	if queueName == nil && i.QueueName != nil {
		queueName = i.QueueName
		q.log.Warn("encountered queue item with inconsistent custom queue name, should have both i.QueueName and i.Data.QueueName set",
			"item", i,
		)
	}

	// sanity check: queueName values must match
	if i.Data.QueueName != nil && i.QueueName != nil && *i.Data.QueueName != *i.QueueName {
		q.log.Warn("encountered queue item with inconsistent custom queue names, should have matching values for i.QueueName and i.Data.QueueName",
			"item", i,
		)
	}

	// The only case when we manually set a queueName is for system partitions
	if queueName != nil {
		systemPartition := QueuePartition{
			// NOTE: Never remove this. The ID is required to enqueue items to the
			// partition, as it is used for conditional checks in Lua
			ID:        *queueName,
			QueueName: queueName,
		}
		// Fetch most recent system concurrency limit
		systemLimits := q.systemConcurrencyLimitGetter(ctx, systemPartition)
		systemPartition.ConcurrencyLimit = systemLimits.PartitionLimit

		return systemPartition, systemLimits.GlobalLimit
	}

	if i.FunctionID == uuid.Nil {
		q.log.Error("unexpected missing functionID in ItemPartitions()", "item", i)
	}

	fnPartition := QueuePartition{
		ID:            i.FunctionID.String(),
		PartitionType: int(enums.PartitionTypeDefault), // Function partition
		FunctionID:    &i.FunctionID,
		AccountID:     i.Data.Identifier.AccountID,
	}

	limits, _ := duration(ctx, q.primaryQueueShard.Name, "partition_fn_concurrency_getter", q.clock.Now(), func(ctx context.Context) (PartitionConcurrencyLimits, error) {
		// Get the function limit from the `concurrencyLimitGetter`.
		return q.concurrencyLimitGetter(ctx, fnPartition), nil
	})

	// The concurrency limit for fns MUST be added for leasing.
	fnPartition.ConcurrencyLimit = limits.FunctionLimit
	if fnPartition.ConcurrencyLimit <= 0 {
		// Use account-level limits, as there are no function level limits
		fnPartition.ConcurrencyLimit = limits.AccountLimit
	}

	return fnPartition, limits.AccountLimit
}

// ItemPartitions returns up 3 item partitions for a given queue item, as well as the account concurrency limit.
// Note: Currently, we only ever return 2 partitions (2x custom concurrency keys or function + custom concurrency key)
// Note: For backwards compatibility, we may return a third partition for the function itself, in case two custom concurrency keys are used.
// This will change with the implementation of throttling key queues.
func (q *queue) ItemPartitions(ctx context.Context, shard QueueShard, i osqueue.QueueItem) (fnPartition, customConcurrencyKey1, customConcurrencyKey2 QueuePartition, accountConcurrencyLimit int) {
	fnPartition, accountConcurrencyLimit = q.ItemPartition(ctx, shard, i)

	ckeys := i.Data.GetConcurrencyKeys()
	if len(ckeys) == 0 {
		return
	}

	// Check if we have custom concurrency keys for the given function.  If so,
	// we're going to create new partitions for each of the custom keys.  This allows
	// us to create queues of queues for each concurrency key.
	//
	// See the 'key queues' spec for more information (internally).
	//
	// NOTE: This is an optimization that ensures we return *updated* concurrency keys
	// for any recently published function configuration.  The embeddeed ckeys from the
	// queue items above may be outdated.
	if q.customConcurrencyLimitRefresher != nil {
		// As an optimization, allow fetching updated concurrency limits if desired.
		updated, _ := duration(ctx, q.primaryQueueShard.Name, "partition_custom_concurrency_getter", q.clock.Now(), func(ctx context.Context) ([]state.CustomConcurrency, error) {
			return q.customConcurrencyLimitRefresher(ctx, i), nil
		})
		for _, update := range updated {
			// This is quadratic, but concurrency keys are limited to 2 so it's
			// okay.
			for n, existing := range ckeys {
				if existing.Key == update.Key {
					ckeys[n].Limit = update.Limit
				}
			}
		}
	}

	// Up to 2 concurrency keys.
	for j, key := range ckeys {
		scope, id, checksum, _ := key.ParseKey()

		// TODO: Is this supposed to stay? Then the comment below should change
		// (if not, do we validate against this case from happening in cloud?)
		if checksum == "" && key.Key != "" {
			// For testing, use the key here.
			checksum = key.Key
		}

		partition := QueuePartition{
			ID:               shard.RedisClient.kg.PartitionQueueSet(enums.PartitionTypeConcurrencyKey, id.String(), checksum),
			PartitionType:    int(enums.PartitionTypeConcurrencyKey),
			FunctionID:       &i.FunctionID,
			AccountID:        i.Data.Identifier.AccountID,
			ConcurrencyScope: int(scope),

			EvaluatedConcurrencyKey:    key.Key,
			UnevaluatedConcurrencyHash: key.Hash,

			// Note: This uses the latest limit for the key queue,
			// retrieved from customConcurrencyLimitRefresher
			ConcurrencyLimit: key.Limit,
		}

		switch scope {
		case enums.ConcurrencyScopeFn:
			partition.FunctionID = &i.FunctionID
		case enums.ConcurrencyScopeEnv:
			partition.EnvID = &i.WorkspaceID
		case enums.ConcurrencyScopeAccount:
			// AccountID comes from the concurrency key in this case
			partition.AccountID = id
		}

		switch j {
		case 0:
			customConcurrencyKey1 = partition
		case 1:
			customConcurrencyKey2 = partition
		}
	}

	return
}

func (q *queue) EnqueueItem(ctx context.Context, shard QueueShard, i osqueue.QueueItem, at time.Time, opts osqueue.EnqueueOpts) (osqueue.QueueItem, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "EnqueueItem"), redis_telemetry.ScopeQueue)

	if shard.Kind != string(enums.QueueShardKindRedis) {
		return osqueue.QueueItem{}, fmt.Errorf("unsupported queue shard kind for EnqueueItem: %s", shard.Kind)
	}

	kg := shard.RedisClient.kg

	if len(i.ID) == 0 {
		i.SetID(ctx, ulid.MustNew(ulid.Now(), rnd).String())
	} else {
		if !opts.PassthroughJobId {
			i.SetID(ctx, i.ID)
		}
	}

	now := q.clock.Now()

	// XXX: If the length of ID >= max, error.
	if i.WallTimeMS == 0 {
		i.WallTimeMS = at.UnixMilli()
	}

	if at.Before(now) {
		// Normalize to now to minimize latency.
		i.WallTimeMS = now.UnixMilli()
	}

	// Add the At timestamp, if not included.
	if i.AtMS == 0 {
		i.AtMS = at.UnixMilli()
	}

	if i.Data.JobID == nil {
		i.Data.JobID = &i.ID
	}

	partitionTime := at
	if at.Before(now) {
		// We don't want to enqueue partitions (pointers to fns) before now.
		// Doing so allows users to stay at the front of the queue for
		// leases.
		partitionTime = q.clock.Now()
	}

	i.EnqueuedAt = now.UnixMilli()

	defaultPartition, _ := q.ItemPartition(ctx, shard, i)

	isSystemPartition := defaultPartition.IsSystem()

	if defaultPartition.AccountID == uuid.Nil && !isSystemPartition {
		q.log.Warn("attempting to enqueue item to non-system partition without account ID", "item", i)
	}

	enqueueToBacklogs := isSystemPartition && q.enqueueSystemQueuesToBacklog
	if !isSystemPartition && defaultPartition.AccountID != uuid.Nil && q.allowKeyQueues != nil {
		enqueueToBacklogs = q.allowKeyQueues(ctx, defaultPartition.AccountID)
	}

	var backlog QueueBacklog
	var shadowPartition QueueShadowPartition
	if enqueueToBacklogs {
		backlog = q.ItemBacklog(ctx, i)
		shadowPartition = q.ItemShadowPartition(ctx, i)
	}

	keys := []string{
		kg.QueueItem(),            // Queue item
		kg.PartitionItem(),        // Partition item, map
		kg.GlobalPartitionIndex(), // Global partition queue
		kg.GlobalAccountIndex(),
		kg.AccountPartitionIndex(i.Data.Identifier.AccountID), // new queue items always contain the account ID
		kg.Idempotency(i.ID),
		kg.FnMetadata(i.FunctionID),

		// Add all 3 partition sets
		defaultPartition.zsetKey(kg),

		// Key queues v2
		kg.BacklogSet(backlog.BacklogID),
		kg.BacklogMeta(),
		kg.GlobalShadowPartitionSet(),
		kg.ShadowPartitionSet(shadowPartition.PartitionID),
		kg.ShadowPartitionMeta(),
		kg.GlobalAccountShadowPartitions(),
		kg.AccountShadowPartitions(i.Data.Identifier.AccountID), // will be empty for system queues

		// Key queue Normalization
		kg.BacklogSet(opts.NormalizeFromBacklogID),
		kg.PartitionNormalizeSet(shadowPartition.PartitionID),
		kg.AccountNormalizeSet(i.Data.Identifier.AccountID),
		kg.GlobalAccountNormalizeSet(),

		// Singletons
		kg.SingletonRunKey(i.Data.Identifier.RunID.String()),
		kg.SingletonKey(i.Data.Singleton),
	}
	// Append indexes
	for _, idx := range q.itemIndexer(ctx, i, shard.RedisClient.kg) {
		if idx != "" {
			keys = append(keys, idx)
		}
	}

	enqueueToBacklogsVal := "0"
	if enqueueToBacklogs {
		enqueueToBacklogsVal = "1"
	}

	args, err := StrSlice([]any{
		i,
		i.ID,
		at.UnixMilli(),
		partitionTime.Unix(),
		now.UnixMilli(),
		FnMetadata{
			// enqueue.lua only writes function metadata if it doesn't already exist.
			// if it doesn't exist, and we're enqueuing something, this implies the fn is not currently paused.
			FnID:   i.FunctionID,
			Paused: false,
		},
		defaultPartition,
		defaultPartition.ID,
		i.Data.Identifier.AccountID.String(),
		i.Data.Identifier.RunID.String(),

		enqueueToBacklogsVal,
		shadowPartition,
		backlog,
		backlog.BacklogID,

		opts.NormalizeFromBacklogID,
	})
	if err != nil {
		return i, err
	}

	q.log.Trace("enqueue item",
		"id", i.ID,
		"kind", i.Data.Kind,
		"time", at.Format(time.StampMilli),
		"partition_time", partitionTime.Format(time.StampMilli),
		"partition", shadowPartition.PartitionID,
		"backlog", enqueueToBacklogs,
	)

	status, err := scripts["queue/enqueue"].Exec(
		redis_telemetry.WithScriptName(ctx, "enqueue"),
		shard.RedisClient.Client(),
		keys,
		args,
	).AsInt64()
	if err != nil {
		return i, fmt.Errorf("error enqueueing item: %w", err)
	}
	switch status {
	case 0:
		// Hint to executor that we should refill if the item has no delay
		refillSoon := i.ExpectedDelay() < ShadowPartitionLookahead
		if enqueueToBacklogs && refillSoon {
			q.addShadowContinue(ctx, &shadowPartition, 0)
		}

		return i, nil
	case 1:
		return i, ErrQueueItemExists
	case 2:
		return i, ErrQueueItemSingletonExists
	default:
		return i, fmt.Errorf("unknown response enqueueing item: %v (%T)", status, status)
	}
}

// SetFunctionPaused sets the "Paused" flag (represented in JSON as "off") for the given
// function ID's queue partition.
// If a function is unpaused, we requeue the partition with a score of "now" to ensure that it is processed.
func (q *queue) SetFunctionPaused(ctx context.Context, accountId uuid.UUID, fnID uuid.UUID, paused bool) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "SetFunctionPaused"), redis_telemetry.ScopeQueue)

	iterate := func(shard QueueShard) error {
		// TODO Support other storage backends
		if shard.Kind != string(enums.QueueShardKindRedis) {
			return nil
		}

		pausedArg := "0"
		if paused {
			pausedArg = "1"
		}

		// This is written to the store if fn metadata doesn't exist.
		defaultFnMetadata := FnMetadata{
			FnID:   fnID,
			Paused: paused,
		}

		keys := []string{shard.RedisClient.kg.FnMetadata(fnID)}
		args, err := StrSlice([]any{
			pausedArg,
			defaultFnMetadata,
		})
		if err != nil {
			return err
		}

		status, err := scripts["queue/fnSetPaused"].Exec(
			redis_telemetry.WithScriptName(ctx, "fnSetPaused"),
			shard.RedisClient.unshardedRc,
			keys,
			args,
		).AsInt64()
		if err != nil {
			return fmt.Errorf("error updating paused state: %w", err)
		}
		switch status {
		case 0:
			// If a function was paused, there's no need to process it. We can push back paused partitions for a long time.
			// Instead of doing this here, we push back paused partitions in partitionPeek to prevent racing a currently processing partition.
			if !paused {
				fnPart := QueuePartition{
					ID:         fnID.String(),
					FunctionID: &fnID,
					AccountID:  accountId,
				}

				// When it does get unpaused, we should immediately start processing it again
				err := q.PartitionRequeue(ctx, shard, &fnPart, time.Now(), false)
				if err != nil && !errors.Is(err, ErrPartitionNotFound) && !errors.Is(err, ErrPartitionGarbageCollected) {
					return fmt.Errorf("could not requeue partition after modifying paused state to %t: %w", paused, err)
				}
			}

			return nil
		default:
			return fmt.Errorf("unknown response updating paused state: %d", status)
		}
	}

	if q.queueShardClients != nil {
		eg := errgroup.Group{}
		for _, shard := range q.queueShardClients {
			shard := shard
			eg.Go(func() error {
				err := iterate(shard)
				if err != nil {
					return fmt.Errorf("could not update paused state for shard %s: %w", shard.Name, err)
				}

				return nil
			})
		}

		if err := eg.Wait(); err != nil {
			return err
		}
	}

	return nil
}

// dropPartitionPointerIfEmpty atomically drops a pointer queue member if the associated
// ZSET is empty. This is used to ensure that we don't have pointers to empty ZSETs, in case
// the cleanup process fails.
func (q *queue) dropPartitionPointerIfEmpty(ctx context.Context, shard QueueShard, keyIndex, keyPartition, indexMember string) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "SetFunctionPaused"), redis_telemetry.ScopeQueue)

	if shard.Kind != string(enums.QueueShardKindRedis) {
		return nil
	}

	keys := []string{keyIndex, keyPartition}
	args, err := StrSlice([]any{
		indexMember,
	})
	if err != nil {
		return err
	}

	status, err := scripts["queue/dropPartitionPointerIfEmpty"].Exec(
		redis_telemetry.WithScriptName(ctx, "dropPartitionPointerIfEmpty"),
		shard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error dropping pointer %q from %q if %q was empty: %w", indexMember, keyIndex, keyPartition, err)
	}
	switch status {
	case 0, 1:
		return nil
	default:
		return fmt.Errorf("unknown response dropping pointer if empty: %d", status)
	}
}

func (q *queue) SetFunctionMigrate(ctx context.Context, sourceShard string, fnID uuid.UUID, migrate bool) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "SetFunctionMigrate"), redis_telemetry.ScopeQueue)

	defaultMeta := FnMetadata{
		FnID:    fnID,
		Migrate: migrate,
	}

	if q.queueShardClients == nil {
		return fmt.Errorf("no queue shard clients are available")
	}

	shard, ok := q.queueShardClients[sourceShard]
	if !ok {
		return fmt.Errorf("no queue shard available for '%s'", sourceShard)
	}

	flag := 0
	if migrate {
		flag = 1
	}

	keys := []string{shard.RedisClient.kg.FnMetadata(fnID)}
	args, err := StrSlice([]any{
		flag,
		defaultMeta,
	})
	if err != nil {
		return err
	}

	status, err := scripts["queue/fnSetMigrate"].Exec(
		ctx,
		shard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error updating queue migrate state: %w", err)
	}

	switch status {
	case 0:
		return nil

	default:
		return fmt.Errorf("unknown response updating queue migration state: %d", err)
	}
}

func (q *queue) Migrate(ctx context.Context, sourceShardName string, fnID uuid.UUID, limit int64, concurrency int, handler osqueue.QueueMigrationHandler) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "MigrationPeek"), redis_telemetry.ScopeQueue)

	shard, ok := q.queueShardClients[sourceShardName]
	if !ok {
		return -1, fmt.Errorf("no queue shard available for '%s'", sourceShardName)
	}

	from := time.Time{}
	// setting it to 5 years ahead should be enough to cover all queue items in the partition
	until := q.clock.Now().Add(24 * time.Hour * 365 * 5)
	items, err := q.ItemsByPartition(ctx, shard, fnID, from, until,
		WithQueueItemIterBatchSize(limit),
	)
	if err != nil {
		// the partition doesn't exist, meaning there are no workloads
		if errors.Is(err, rueidis.Nil) {
			return 0, nil
		}

		return -1, fmt.Errorf("error preparing partition iteration: %w", err)
	}

	// Should process in order because we don't want out of order execution when moved over
	var processed int64

	process := func(qi *osqueue.QueueItem) error {
		if err := handler(ctx, qi); err != nil {
			return err
		}

		if err := q.Dequeue(ctx, shard, *qi); err != nil {
			q.log.Error("error dequeueing queue item after migration", "error", err)
		}

		atomic.AddInt64(&processed, 1)
		return nil
	}

	if concurrency > 0 {
		eg := errgroup.Group{}
		eg.SetLimit(concurrency)

		for qi := range items {
			i := qi
			eg.Go(func() error {
				return process(i)
			})
		}

		err := eg.Wait()
		if err != nil {
			return atomic.LoadInt64(&processed), err
		}

		return atomic.LoadInt64(&processed), nil
	}

	for qi := range items {
		if err := process(qi); err != nil {
			return processed, err
		}
	}

	return atomic.LoadInt64(&processed), nil
}

func (q *queue) RemoveQueueItem(ctx context.Context, shardName string, partitionKey string, itemID string) error {
	queueShard, ok := q.queueShardClients[shardName]
	if !ok {
		return fmt.Errorf("queue shard not found %q", shardName)
	}
	return q.removeQueueItem(ctx, queueShard, partitionKey, itemID)
}

// removeQueueItem attempts to remove a specific item in the target queue shard
// and also remove it from the queue item hash as well
func (q *queue) removeQueueItem(ctx context.Context, shard QueueShard, partitionKey string, itemID string) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "removeQueueItem"), redis_telemetry.ScopeQueue)

	keys := []string{
		partitionKey,
		shard.RedisClient.kg.QueueItem(),
	}
	args := []string{itemID}

	code, err := scripts["queue/removeItem"].Exec(
		ctx,
		shard.RedisClient.Client(),
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error deleting queue item: %w", err)
	}

	switch code {
	case 0:
		q.log.Debug("removed queue item", "item_id", itemID)

		return nil
	default:
		return fmt.Errorf("unknown status when attempting to remove item: %d", code)
	}
}

func (q *queue) LoadQueueItem(ctx context.Context, shardName string, itemID string) (*osqueue.QueueItem, error) {
	queueShard, ok := q.queueShardClients[shardName]
	if !ok {
		return nil, fmt.Errorf("queue shard not found %q", shardName)
	}

	kg := queueShard.RedisClient.KeyGenerator()
	client := queueShard.RedisClient.Client()

	queueItemStr, err := client.Do(ctx, client.B().Hget().Key(kg.QueueItem()).Field(itemID).Build()).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, ErrQueueItemNotFound
		}

		return nil, fmt.Errorf("could not load queue item: %w", err)
	}

	qi := &osqueue.QueueItem{}
	if err := json.Unmarshal([]byte(queueItemStr), qi); err != nil {
		return nil, fmt.Errorf("error unmarshalling loaded queue item: %w", err)
	}

	return qi, nil
}

// Peek takes n items from a queue, up until QueuePeekMax.  For peeking workflow/
// function jobs the queue name must be the ID of the workflow;  each workflow has
// its own queue of jobs using its ID as the queue name.
//
// If limit is -1, this will return the first unleased item - representing the next available item in the
// queue.
func (q *queue) Peek(ctx context.Context, partition *QueuePartition, until time.Time, limit int64) ([]*osqueue.QueueItem, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Peek"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for Peek: %s", q.primaryQueueShard.Kind)
	}

	if partition == nil {
		return nil, fmt.Errorf("expected partition to be set")
	}

	// Check whether limit is -1, peeking next available time
	isPeekNext := limit == -1

	if limit > AbsoluteQueuePeekMax {
		// Lua's max unpack() length is 8000; don't allow users to peek more than
		// 1k at a time regardless.
		limit = AbsoluteQueuePeekMax
	}
	if limit > q.peekMax {
		limit = q.peekMax
	}
	if limit <= 0 {
		limit = q.peekMin
	}
	if isPeekNext {
		limit = 1
	}

	partitionKey := partition.zsetKey(q.primaryQueueShard.RedisClient.kg)
	return q.peek(
		ctx,
		q.primaryQueueShard,
		peekOpts{
			Limit:        limit,
			Until:        until,
			PartitionKey: partitionKey,
			PartitionID:  partition.ID,
		},
	)
}

func (q *queue) PeekRandom(ctx context.Context, partition *QueuePartition, until time.Time, limit int64) ([]*osqueue.QueueItem, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Peek"), redis_telemetry.ScopeQueue)
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for Peek: %s", q.primaryQueueShard.Kind)
	}
	if partition == nil {
		return nil, fmt.Errorf("expected partition to be set")
	}
	if limit > AbsoluteQueuePeekMax {
		// Lua's max unpack() length is 8000; don't allow users to peek more than
		// 1k at a time regardless.
		limit = AbsoluteQueuePeekMax
	}
	if limit > q.peekMax {
		limit = q.peekMax
	}
	if limit <= 0 {
		limit = q.peekMin
	}
	partitionKey := partition.zsetKey(q.primaryQueueShard.RedisClient.kg)
	return q.peek(
		ctx,
		q.primaryQueueShard,
		peekOpts{
			Limit:        limit,
			Until:        until,
			PartitionKey: partitionKey,
			PartitionID:  partition.ID,
			Random:       true,
		},
	)
}

type peekOpts struct {
	PartitionID  string
	PartitionKey string
	Random       bool
	From         *time.Time
	Until        time.Time
	Limit        int64
}

func (q *queue) peek(ctx context.Context, shard QueueShard, opts peekOpts) ([]*osqueue.QueueItem, error) {
	from := "-inf"
	if opts.From != nil && !opts.From.IsZero() {
		from = strconv.Itoa(int(opts.From.UnixMilli()))
	}

	until := "+inf"
	if opts.Until.UnixMilli() > 0 {
		until = strconv.Itoa(int(opts.Until.UnixMilli()))
	}

	randomOffset := "0"
	if opts.Random {
		randomOffset = "1"
	}

	keys := []string{
		opts.PartitionKey,
		shard.RedisClient.kg.QueueItem(),
	}
	args, err := StrSlice([]any{
		from,
		until,
		opts.Limit,
		randomOffset,
	})
	if err != nil {
		return nil, err
	}

	peekRet, err := scripts["queue/peek"].Exec(
		redis_telemetry.WithScriptName(ctx, "peek"),
		shard.RedisClient.unshardedRc,
		keys,
		args,
	).ToAny()
	if err != nil {
		return nil, fmt.Errorf("error peeking queue items: %w", err)
	}

	returnedSet, ok := peekRet.([]any)
	if !ok {
		return nil, fmt.Errorf("unknown return type from peek: %T", peekRet)
	}

	var potentiallyMissingItems, allQueueItemIds []any
	if len(returnedSet) == 2 {
		potentiallyMissingItems, ok = returnedSet[0].([]any)
		if !ok {
			return nil, fmt.Errorf("unexpected first item in set returned from peek: %T", peekRet)
		}

		allQueueItemIds, ok = returnedSet[1].([]any)
		if !ok {
			return nil, fmt.Errorf("unexpected first item in set returned from peek: %T", peekRet)
		}
	} else if len(returnedSet) != 0 {
		return nil, fmt.Errorf("expected zero or two items in set returned by peek: %v", returnedSet)
	}

	items := make([]any, 0, len(allQueueItemIds))
	missingQueueItems := make([]string, 0, len(allQueueItemIds))
	for idx, itemId := range allQueueItemIds {
		item := potentiallyMissingItems[idx]
		if item == nil {
			if itemId == nil {
				return nil, fmt.Errorf("encountered nil queue item key in partition queue %q", opts.PartitionKey)
			}

			str, ok := itemId.(string)
			if !ok {
				return nil, fmt.Errorf("encountered non-string queue item key in partition queue %q", opts.PartitionKey)
			}

			missingQueueItems = append(missingQueueItems, str)
		} else {
			items = append(items, item)
		}
	}

	if len(missingQueueItems) > 0 {
		q.log.Warn("encountered missing queue items in partition queue",
			"key", opts.PartitionKey,
			"items", missingQueueItems,
		)

		eg := errgroup.Group{}
		for _, missingItemId := range missingQueueItems {
			id := missingItemId
			eg.Go(func() error {
				return q.removeQueueItem(ctx, shard, opts.PartitionKey, id)
			})
		}

		if err := eg.Wait(); err != nil {
			return nil, fmt.Errorf("error cleaning up nil partitions in account pointer queue: %w", err)
		}
	}

	return util.ParallelDecode(items, func(val any) (*osqueue.QueueItem, bool, error) {
		if val == nil {
			q.log.Error("nil item value in peek response", "partition", opts.PartitionKey)
			return nil, true, nil
		}

		str, ok := val.(string)
		if !ok {
			return nil, false, fmt.Errorf("non-string value in peek response: %T", val)
		}

		if str == "" {
			return nil, false, fmt.Errorf("received empty string in decode queue item from peek")
		}

		qi := &osqueue.QueueItem{}
		if err := json.Unmarshal(unsafe.Slice(unsafe.StringData(str), len(str)), qi); err != nil {
			return nil, false, fmt.Errorf("error unmarshalling peeked queue item: %w", err)
		}

		now := q.clock.Now()
		if qi.IsLeased(now) {
			metrics.IncrQueuePeekLeaseContentionCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					// "partition_id": opts.PartitionID,
					"queue_shard": shard.Name,
				},
			})

			// Leased item, don't return.
			return nil, true, nil
		}

		// The nested osqueue.Item never has an ID set;  always re-set it
		qi.Data.JobID = &qi.ID
		return qi, false, nil
	})
}

// RequeueByJobID requeues a job for a specific time given a partition name and job ID.
//
// If the queue item referenced by the job ID is not outstanding (ie. it has a lease, is in
// progress, or doesn't exist) this returns an error.
//
// Note: This only works with items that directly go into ready queues (system queues).
func (q *queue) RequeueByJobID(ctx context.Context, queueShard QueueShard, jobID string, at time.Time) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "RequeueByJobID"), redis_telemetry.ScopeQueue)

	if queueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for RequeueByJobID: %s", queueShard.Kind)
	}

	jobID = osqueue.HashID(ctx, jobID)

	// Find the queue item so that we can fetch the shard info.
	i := osqueue.QueueItem{}
	if err := queueShard.RedisClient.unshardedRc.Do(ctx, queueShard.RedisClient.unshardedRc.B().Hget().Key(queueShard.RedisClient.kg.QueueItem()).Field(jobID).Build()).DecodeJSON(&i); err != nil {
		return err
	}

	// Don't requeue before now.
	now := q.clock.Now()
	if at.Before(now) {
		at = now
	}

	// Remove all items from all partitions.  For this, we need all partitions for
	// the queue item instead of just the partition passed via args.
	//
	// This is because a single queue item may be present in more than one queue.
	fnPartition, _ := q.ItemPartition(ctx, queueShard, i)

	keys := []string{
		queueShard.RedisClient.kg.QueueItem(),
		queueShard.RedisClient.kg.PartitionItem(), // Partition item, map
		queueShard.RedisClient.kg.GlobalPartitionIndex(),
		queueShard.RedisClient.kg.GlobalAccountIndex(),
		queueShard.RedisClient.kg.AccountPartitionIndex(i.Data.Identifier.AccountID),

		fnPartition.zsetKey(queueShard.RedisClient.kg),
	}

	args, err := StrSlice([]any{
		jobID,
		strconv.Itoa(int(at.UnixMilli())),
		strconv.Itoa(int(now.UnixMilli())),
		fnPartition,
		fnPartition.ID,
		i.Data.Identifier.AccountID.String(),
	})
	if err != nil {
		return err
	}
	status, err := scripts["queue/requeueByID"].Exec(
		redis_telemetry.WithScriptName(ctx, "requeueByID"),
		queueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		q.log.Error("error requeueing queue item by JobID",
			"error", err,
			"item", i,
			"fnPartition", fnPartition,
		)
		return fmt.Errorf("error requeueing item: %w", err)
	}
	switch status {
	case 0:
		return nil
	case -1:
		return ErrQueueItemNotFound
	case -2:
		return ErrQueueItemAlreadyLeased
	default:
		return fmt.Errorf("unknown requeue by id response: %d", status)
	}
}

func (q *queue) itemEnableKeyQueues(ctx context.Context, item osqueue.QueueItem) bool {
	isSystem := item.QueueName != nil || item.Data.QueueName != nil
	if isSystem {
		return q.enqueueSystemQueuesToBacklog
	}

	if item.Data.Identifier.AccountID != uuid.Nil && q.allowKeyQueues != nil {
		return q.allowKeyQueues(ctx, item.Data.Identifier.AccountID)
	}

	return false
}

func (q *queue) itemDisableLeaseChecks(ctx context.Context, item osqueue.QueueItem) bool {
	isSystem := item.QueueName != nil || item.Data.QueueName != nil
	if isSystem {
		return q.disableLeaseChecksForSystemQueues
	}

	if item.Data.Identifier.AccountID != uuid.Nil && q.disableLeaseChecks != nil {
		return q.disableLeaseChecks(ctx, item.Data.Identifier.AccountID)
	}

	return false
}

// Lease temporarily dequeues an item from the queue by obtaining a lease, preventing
// other workers from working on this queue item at the same time.
//
// Obtaining a lease updates the vesting time for the queue item until now() +
// lease duration. This returns the newly acquired lease ID on success.
func (q *queue) Lease(ctx context.Context, item osqueue.QueueItem, leaseDuration time.Duration, now time.Time, denies *leaseDenies) (*ulid.ULID, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Lease"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for Lease: %s", q.primaryQueueShard.Kind)
	}

	kg := q.primaryQueueShard.RedisClient.kg

	enableKeyQueues := q.itemEnableKeyQueues(ctx, item)

	refilledFromBacklog := enableKeyQueues && item.RefilledFrom != ""

	checkConstraints := !refilledFromBacklog || !q.itemDisableLeaseChecks(ctx, item)

	if checkConstraints {
		if item.Data.Throttle != nil && denies != nil && denies.denyThrottle(item.Data.Throttle.Key) {
			return nil, ErrQueueItemThrottled
		}

		// Check to see if this key has already been denied in the lease iteration.
		// If partition concurrency limits were encountered previously, fail early.
		if denies != nil && denies.denyConcurrency(item.FunctionID.String()) {
			// Note that we do not need to wrap the key as the key is already present.
			return nil, ErrPartitionConcurrencyLimit
		}

		// Same for account concurrency limits
		if denies != nil && denies.denyConcurrency(item.Data.Identifier.AccountID.String()) {
			return nil, ErrAccountConcurrencyLimit
		}
	}

	backlog := q.ItemBacklog(ctx, item)
	partition := q.ItemShadowPartition(ctx, item)

	if checkConstraints {
		// Check to see if this key has already been denied in the lease iteration.
		// If so, fail early.
		if denies != nil && len(backlog.ConcurrencyKeys) > 0 && denies.denyConcurrency(backlog.customConcurrencyKeyID(1)) {
			return nil, ErrConcurrencyLimitCustomKey
		}

		// Check to see if this key has already been denied in the lease iteration.
		// If so, fail early.
		if denies != nil && len(backlog.ConcurrencyKeys) > 1 && denies.denyConcurrency(backlog.customConcurrencyKeyID(2)) {
			return nil, ErrConcurrencyLimitCustomKey
		}
	}

	leaseID, err := ulid.New(ulid.Timestamp(now.Add(leaseDuration).UTC()), rnd)
	if err != nil {
		return nil, fmt.Errorf("error generating id: %w", err)
	}

	refilledFromBacklogVal := "0"
	if refilledFromBacklog {
		refilledFromBacklogVal = "1"
	}

	checkConstraintsVal := "0"
	if checkConstraints {
		checkConstraintsVal = "1"
	}

	keys := []string{
		kg.QueueItem(),
		kg.ConcurrencyIndex(),

		partition.readyQueueKey(kg),

		// In progress (concurrency) ZSETs
		partition.accountInProgressKey(kg),
		partition.inProgressKey(kg),
		backlog.customKeyInProgress(kg, 1),
		backlog.customKeyInProgress(kg, 2),

		// Active set keys (ready + in progress)
		partition.accountActiveKey(kg),
		partition.activeKey(kg),
		backlog.customKeyActive(kg, 1),
		backlog.customKeyActive(kg, 2),
		backlog.activeKey(kg),

		// Active run sets
		kg.RunActiveSet(item.Data.Identifier.RunID), // Set for active items in run
		partition.accountActiveRunKey(kg),           // Set for active runs in account
		partition.activeRunKey(kg),                  // Set for active runs in partition
		backlog.customKeyActiveRuns(kg, 1),          // Set for active runs with custom concurrency key 1
		backlog.customKeyActiveRuns(kg, 2),          // Set for active runs with custom concurrency key 2

		kg.ThrottleKey(item.Data.Throttle),
	}

	partConcurrency := partition.Concurrency.FunctionConcurrency
	if partition.SystemQueueName != nil {
		partConcurrency = partition.Concurrency.SystemConcurrency
	}

	args, err := StrSlice([]any{
		item.ID,
		partition.PartitionID,
		item.Data.Identifier.AccountID,
		item.Data.Identifier.RunID.String(),

		leaseID.String(),
		now.UnixMilli(),

		// Concurrency limits
		partition.Concurrency.AccountConcurrency,
		partConcurrency,
		partition.CustomConcurrencyLimit(1),
		partition.CustomConcurrencyLimit(2),

		// Key queues v2
		checkConstraintsVal,
		refilledFromBacklogVal,
	})
	if err != nil {
		return nil, err
	}

	status, err := scripts["queue/lease"].Exec(
		redis_telemetry.WithScriptName(ctx, "lease"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).ToInt64()
	if err != nil {
		return nil, fmt.Errorf("error leasing queue item: %w", err)
	}

	itemDelay := item.ExpectedDelay()
	metrics.HistogramQueueOperationDelay(ctx, itemDelay, metrics.HistogramOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard": q.primaryQueueShard.Name,
			"op":          "item",
		}},
	)

	l := q.log.With("item_delay", itemDelay.String())

	refillDelay := item.RefillDelay()
	metrics.HistogramQueueOperationDelay(ctx, refillDelay, metrics.HistogramOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard": q.primaryQueueShard.Name,
			"op":          "refill",
		}},
	)
	l = l.With("refill_delay", refillDelay.String())

	// leaseDelay is the time between refilling and leasing
	leaseDelay := item.LeaseDelay(now)
	metrics.HistogramQueueOperationDelay(ctx, leaseDelay, metrics.HistogramOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard": q.primaryQueueShard.Name,
			"op":          "lease",
		}},
	)
	l = l.With("lease_delay", leaseDelay.String())

	l.Trace("leasing item",
		"id", item.ID,
		"kind", item.Data.Kind,
		"lease_id", leaseID.String(),
		"partition_id", partition.PartitionID,
		"item_delay", itemDelay.String(),
		"refilled", refilledFromBacklog,
		"check", checkConstraints,
		"status", status,
	)

	switch status {
	case 0:
		return &leaseID, nil
	case -1:
		return nil, ErrQueueItemNotFound
	case -2:
		return nil, ErrQueueItemAlreadyLeased
	case -3:
		// This partition is reused for function partitions without keys, system partions,
		// and potentially concurrency key partitions. Errors should be returned based on
		// the partition type

		if partition.SystemQueueName != nil {
			return nil, newKeyError(ErrSystemConcurrencyLimit, partition.PartitionID)
		}

		return nil, newKeyError(ErrPartitionConcurrencyLimit, item.FunctionID.String())
	case -4:
		return nil, newKeyError(ErrConcurrencyLimitCustomKey, backlog.customConcurrencyKeyID(1))
	case -5:
		return nil, newKeyError(ErrConcurrencyLimitCustomKey, backlog.customConcurrencyKeyID(2))
	case -6:
		return nil, newKeyError(ErrAccountConcurrencyLimit, item.Data.Identifier.AccountID.String())
	case -7:
		if item.Data.Throttle == nil {
			// This should never happen, as the throttle key is nil.
			return nil, fmt.Errorf("lease attempted throttle with nil throttle config: %#v", item)
		}
		return nil, newKeyError(ErrQueueItemThrottled, item.Data.Throttle.Key)
	default:
		return nil, fmt.Errorf("unknown response leasing item: %d", status)
	}
}

// ExtendLease extens the lease for a given queue item, given the queue item is currently
// leased with the given ID.  This returns a new lease ID if the lease is successfully ended.
//
// The existing lease ID must be passed in so that we can guarantee that the worker
// renewing the lease still owns the lease.
//
// Renewing a lease updates the vesting time for the queue item until now() +
// lease duration. This returns the newly acquired lease ID on success.
func (q *queue) ExtendLease(ctx context.Context, i osqueue.QueueItem, leaseID ulid.ULID, duration time.Duration) (*ulid.ULID, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ExtendLease"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for ExtendLease: %s", q.primaryQueueShard.Kind)
	}

	kg := q.primaryQueueShard.RedisClient.kg

	newLeaseID, err := ulid.New(ulid.Timestamp(q.clock.Now().Add(duration).UTC()), rnd)
	if err != nil {
		return nil, fmt.Errorf("error generating id: %w", err)
	}

	backlog := q.ItemBacklog(ctx, i)
	partition := q.ItemShadowPartition(ctx, i)

	keys := []string{
		q.primaryQueueShard.RedisClient.kg.QueueItem(),
		// And pass in the key queue's concurrency keys.
		partition.inProgressKey(kg),
		backlog.customKeyInProgress(kg, 1),
		backlog.customKeyInProgress(kg, 2),
		partition.accountInProgressKey(kg),
		q.primaryQueueShard.RedisClient.kg.ConcurrencyIndex(),
	}

	args, err := StrSlice([]any{
		i.ID,
		leaseID.String(),
		newLeaseID.String(),
		partition.PartitionID,
	})
	if err != nil {
		return nil, err
	}

	status, err := scripts["queue/extendLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "extendLease"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error extending lease: %w", err)
	}
	switch status {
	case 0:
		return &newLeaseID, nil
	case 1:
		return nil, ErrQueueItemNotFound
	case 2:
		return nil, ErrQueueItemNotLeased
	case 3:
		return nil, ErrQueueItemLeaseMismatch
	default:
		return nil, fmt.Errorf("unknown response extending lease: %d", status)
	}
}

// Dequeue removes an item from the queue entirely.
func (q *queue) Dequeue(ctx context.Context, queueShard QueueShard, i osqueue.QueueItem) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Dequeue"), redis_telemetry.ScopeQueue)

	if queueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for Dequeue: %s", queueShard.Kind)
	}

	kg := queueShard.RedisClient.kg

	partition := q.ItemShadowPartition(ctx, i)
	backlog := q.ItemBacklog(ctx, i)

	keys := []string{
		kg.QueueItem(),
		kg.PartitionItem(),

		kg.ConcurrencyIndex(),

		partition.readyQueueKey(kg),
		kg.GlobalPartitionIndex(),
		kg.GlobalAccountIndex(),
		kg.AccountPartitionIndex(i.Data.Identifier.AccountID),

		kg.ShadowPartitionMeta(),
		kg.BacklogMeta(),

		kg.BacklogSet(backlog.BacklogID),
		kg.ShadowPartitionSet(partition.PartitionID),
		kg.GlobalShadowPartitionSet(),
		kg.GlobalAccountShadowPartitions(),
		kg.AccountShadowPartitions(i.Data.Identifier.AccountID),
		kg.PartitionNormalizeSet(partition.PartitionID),

		// In progress keys
		partition.accountInProgressKey(kg),
		partition.inProgressKey(kg),
		backlog.customKeyInProgress(kg, 1),
		backlog.customKeyInProgress(kg, 2),

		// Active set keys
		partition.accountActiveKey(kg),
		partition.activeKey(kg),
		backlog.customKeyActive(kg, 1),
		backlog.customKeyActive(kg, 2),
		backlog.activeKey(kg),

		// Active run sets
		kg.RunActiveSet(i.Data.Identifier.RunID), // Set for active items in run
		partition.accountActiveRunKey(kg),        // Set for active runs in account
		partition.activeRunKey(kg),               // Set for active runs in partition
		backlog.customKeyActiveRuns(kg, 1),       // Set for active runs with custom concurrency key 1
		backlog.customKeyActiveRuns(kg, 2),       // Set for active runs with custom concurrency key 2

		kg.Idempotency(i.ID),

		// Singleton
		kg.SingletonRunKey(i.Data.Identifier.RunID.String()),
	}

	// Append indexes
	for _, idx := range q.itemIndexer(ctx, i, queueShard.RedisClient.kg) {
		if idx != "" {
			keys = append(keys, idx)
		}
	}

	idempotency := q.idempotencyTTL
	if q.idempotencyTTLFunc != nil {
		idempotency = q.idempotencyTTLFunc(ctx, i)
	}

	args, err := StrSlice([]any{
		i.ID,
		partition.PartitionID,
		backlog.BacklogID,
		i.Data.Identifier.AccountID.String(),
		i.Data.Identifier.RunID.String(),

		int(idempotency.Seconds()),
	})
	if err != nil {
		return err
	}

	status, err := scripts["queue/dequeue"].Exec(
		redis_telemetry.WithScriptName(ctx, "dequeue"),
		queueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error dequeueing item: %w", err)
	}
	switch status {
	case 0:
		q.log.Debug("dequeued item", "job_id", i.ID, "item", i)

		return nil
	case 1:
		return ErrQueueItemNotFound
	default:
		return fmt.Errorf("unknown response dequeueing item: %d", status)
	}
}

// Requeue requeues an item in the future.
func (q *queue) Requeue(ctx context.Context, queueShard QueueShard, i osqueue.QueueItem, at time.Time) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Requeue"), redis_telemetry.ScopeQueue)

	l := q.log.With("item", i)

	if queueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for Requeue: %s", queueShard.Kind)
	}

	kg := queueShard.RedisClient.kg

	now := q.clock.Now()
	if at.Before(now) {
		at = now
	}

	// Unset any lease ID as this is requeued.
	i.LeaseID = nil
	// Update the At timestamp.
	// NOTE: This does no priority factorization or FIFO for function ordering,
	// eg. adjusting AtMS based off of function run time.
	i.AtMS = at.UnixMilli()
	// Update the wall time that this should run at.
	i.WallTimeMS = at.UnixMilli()

	// Reset refill details
	i.RefilledFrom = ""
	i.RefilledAt = 0

	// Reset enqueuedAt (used for latency calculation)
	i.EnqueuedAt = now.UnixMilli()

	fnPartition, _ := q.ItemPartition(ctx, queueShard, i)
	shadowPartition := q.ItemShadowPartition(ctx, i)

	requeueToBacklog := q.itemEnableKeyQueues(ctx, i)

	requeueToBacklogsVal := "0"
	if requeueToBacklog {
		requeueToBacklogsVal = "1"

		// To avoid requeueing item into a stale backlog, retrieve latest throttle
		if i.Data.Throttle != nil && i.Data.Throttle.KeyExpressionHash == "" {
			refreshedThrottle, err := q.refreshItemThrottle(ctx, &i)
			if err != nil {
				// If we cannot find the event for the queue item, dequeue it. The state
				// must exist for the entire duration of a function run.
				if errors.Is(err, state.ErrEventNotFound) {
					l.Warn("could not find event for refreshing throttle before requeue")

					err := q.Dequeue(ctx, queueShard, i)
					if err != nil && !errors.Is(err, ErrQueueItemNotFound) {
						return fmt.Errorf("could not dequeue item with missing throttle state: %w", err)
					}

					return nil
				}

				return fmt.Errorf("could not refresh item throttle: %w", err)
			}

			// Update throttle to latest evaluated value + expression hash
			i.Data.Throttle = refreshedThrottle
		}
	}

	backlog := q.ItemBacklog(ctx, i)

	keys := []string{
		kg.QueueItem(),
		kg.PartitionItem(), // Partition item, map
		kg.ConcurrencyIndex(),

		kg.GlobalPartitionIndex(),
		kg.GlobalAccountIndex(),
		kg.AccountPartitionIndex(i.Data.Identifier.AccountID),

		shadowPartition.readyQueueKey(kg),

		// In progress (concurrency) keys
		shadowPartition.accountInProgressKey(kg),
		shadowPartition.inProgressKey(kg),
		backlog.customKeyInProgress(kg, 1),
		backlog.customKeyInProgress(kg, 2),

		// Active set keys
		shadowPartition.accountActiveKey(kg),
		shadowPartition.activeKey(kg),
		backlog.customKeyActive(kg, 1),
		backlog.customKeyActive(kg, 2),
		backlog.activeKey(kg),

		// Active run sets
		kg.RunActiveSet(i.Data.Identifier.RunID), // Set for active items in run
		shadowPartition.accountActiveRunKey(kg),  // Set for active runs in account
		shadowPartition.activeRunKey(kg),         // Set for active runs in partition
		backlog.customKeyActiveRuns(kg, 1),       // Set for active runs with custom concurrency key 1
		backlog.customKeyActiveRuns(kg, 2),       // Set for active runs with custom concurrency key 2

		// key queues v2
		kg.BacklogSet(backlog.BacklogID),
		kg.BacklogMeta(),
		kg.GlobalShadowPartitionSet(),
		kg.ShadowPartitionSet(shadowPartition.PartitionID),
		kg.ShadowPartitionMeta(),
		kg.GlobalAccountShadowPartitions(),
		kg.AccountShadowPartitions(i.Data.Identifier.AccountID), // empty for system partitions
	}
	// Append indexes
	for _, idx := range q.itemIndexer(ctx, i, queueShard.RedisClient.kg) {
		if idx != "" {
			keys = append(keys, idx)
		}
	}

	args, err := StrSlice([]any{
		i.ID,
		i,
		at.UnixMilli(),

		i.Data.Identifier.AccountID.String(),
		i.Data.Identifier.RunID.String(),
		fnPartition.ID,
		fnPartition,

		now.UnixMilli(),

		requeueToBacklogsVal,
		shadowPartition,
		backlog.BacklogID,
		backlog,
	})
	if err != nil {
		return err
	}

	q.log.Trace("requeueing queue item",
		"id", i.ID,
		"kind", i.Data.Kind,
		"time", at.Format(time.StampMilli),
		"partition_id", shadowPartition.PartitionID,
		"backlog", requeueToBacklogsVal,
	)

	status, err := scripts["queue/requeue"].Exec(
		redis_telemetry.WithScriptName(ctx, "requeue"),
		queueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		q.log.Error("error requeueing queue item",
			"error", err,
			"item", i,
			"partition", fnPartition,
			"shadow", shadowPartition,
		)
		return fmt.Errorf("error requeueing item: %w", err)
	}
	switch status {
	case 0:
		switch requeueToBacklogsVal {
		case "1":
			metrics.IncrBacklogRequeuedCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": q.primaryQueueShard.Name,
					// "partition_id": i.FunctionID.String(),
				},
			})
		}

		return nil
	case 1:
		// This should only ever happen if a run is cancelled and all queue items
		// are deleted before requeueing.
		return ErrQueueItemNotFound
	default:
		return fmt.Errorf("unknown response requeueing item: %v (%T)", status, status)
	}
}

func (q *queue) peekGlobalNormalizeAccounts(ctx context.Context, until time.Time, limit int64) ([]uuid.UUID, error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for peekGlobalNormalizeAccounts: %s", q.primaryQueueShard.Kind)
	}

	rc := q.primaryQueueShard.RedisClient

	p := peeker[QueueBacklog]{
		q:                      q,
		opName:                 "peekGlobalNormalizeAccounts",
		max:                    NormalizeAccountPeekMax,
		isMillisecondPrecision: true,
	}

	return p.peekUUIDPointer(ctx, rc.kg.GlobalAccountNormalizeSet(), true, until, limit)
}

// PartitionLease leases a partition for a given workflow ID.  It returns the new lease ID.
//
// NOTE: This does not check the queue/partition name against allow or denylists;  it assumes
// that the worker always wants to lease the given queue.  Filtering must be done when peeking
// when running a worker.
func (q *queue) PartitionLease(ctx context.Context, p *QueuePartition, duration time.Duration) (*ulid.ULID, int, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PartitionLease"), redis_telemetry.ScopeQueue)

	shard := q.primaryQueueShard

	if shard.Kind != string(enums.QueueShardKindRedis) {
		return nil, 0, fmt.Errorf("unsupported queue shard kind for PartitionLease: %s", shard.Kind)
	}

	kg := shard.RedisClient.kg

	var accountLimit, functionLimit, customKeyLimit int
	if p.IsSystem() {
		limits := q.systemConcurrencyLimitGetter(ctx, *p)
		accountLimit = limits.GlobalLimit
		functionLimit = limits.PartitionLimit
	} else {
		limits := q.concurrencyLimitGetter(ctx, *p)
		accountLimit = limits.AccountLimit
		functionLimit = limits.FunctionLimit
		customKeyLimit = limits.CustomKeyLimit
	}

	// XXX: Check for function throttling prior to leasing;  if it's throttled we can requeue
	// the pointer and back off.  A question here is enqueuing new items onto the partition
	// will reset the pointer update, leading to thrash.
	now := q.clock.Now()
	leaseExpires := now.Add(duration).UTC().Truncate(time.Millisecond)
	leaseID, err := ulid.New(ulid.Timestamp(leaseExpires), rnd)
	if err != nil {
		return nil, 0, fmt.Errorf("error generating id: %w", err)
	}

	fnMetaKey := uuid.Nil
	if p.FunctionID != nil {
		fnMetaKey = *p.FunctionID
	}

	disableLeaseChecks := p.IsSystem() && q.disableLeaseChecksForSystemQueues
	if !p.IsSystem() && q.disableLeaseChecks != nil && p.AccountID != uuid.Nil {
		disableLeaseChecks = q.disableLeaseChecks(ctx, p.AccountID)
	}

	disableLeaseChecksVal := "0"
	if disableLeaseChecks {
		disableLeaseChecksVal = "1"
	}

	keys := []string{
		kg.PartitionItem(),
		kg.GlobalPartitionIndex(),
		kg.GlobalAccountIndex(),
		// NOTE: Old partitions will _not_ have an account ID until the next enqueue on the new code.
		// Until this, we may not use account queues at all, as we cannot properly clean up
		// here without knowing the Account ID
		kg.AccountPartitionIndex(p.AccountID),
		kg.FnMetadata(fnMetaKey),

		// These concurrency keys are for fast checking of partition
		// concurrency limits prior to leasing, as an optimization.
		p.acctConcurrencyKey(kg),
		p.fnConcurrencyKey(kg),
		p.customConcurrencyKey(kg),
	}

	args, err := StrSlice([]any{
		p.Queue(),
		leaseID.String(),
		now.UnixMilli(),
		leaseExpires.Unix(),
		accountLimit,
		functionLimit,
		customKeyLimit,
		now.Add(PartitionConcurrencyLimitRequeueExtension).Unix(),
		p.AccountID.String(),
		disableLeaseChecksVal,
	})
	if err != nil {
		return nil, 0, err
	}

	result, err := scripts["queue/partitionLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "partitionLease"),
		shard.RedisClient.unshardedRc,
		keys,
		args,
	).AsIntSlice()
	if err != nil {
		return nil, 0, fmt.Errorf("error leasing partition: %w", err)
	}
	if len(result) == 0 {
		return nil, 0, fmt.Errorf("unknown partition lease result: %v", result)
	}

	q.log.Trace("leased partition",
		"partition", p.Queue(),
		"lease_id", leaseID.String(),
		"status", result[0],
		"expires", leaseExpires.Format(time.StampMilli),
	)

	switch result[0] {
	case -1:
		return nil, 0, ErrAccountConcurrencyLimit
	case -2:
		return nil, 0, ErrPartitionConcurrencyLimit
	case -3:
		return nil, 0, ErrConcurrencyLimitCustomKey
	case -4:
		return nil, 0, ErrPartitionNotFound
	case -5:
		return nil, 0, ErrPartitionAlreadyLeased
	case -6:
		return nil, 0, ErrPartitionPaused
	default:
		limit := functionLimit
		if len(result) == 2 {
			limit = int(result[1])
		}

		// Update the partition's last indicator.
		if result[0] > p.Last {
			p.Last = result[0]
		}

		// result is the available concurrency within this partition
		return &leaseID, limit, nil
	}
}

// GlobalPartitionPeek returns up to PartitionSelectionMax partition items from the queue. This
// returns the indexes of partitions.
//
// If sequential is set to true this returns partitions in order from earliest to latest
// available lease times. Otherwise, this shuffles all partitions and picks partitions
// randomly, with higher priority partitions more likely to be selected.  This reduces
// lease contention amongst multiple shared-nothing workers.
func (q *queue) PartitionPeek(ctx context.Context, sequential bool, until time.Time, limit int64) ([]*QueuePartition, error) {
	return q.partitionPeek(ctx, q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex(), sequential, until, limit, nil)
}

func (q *queue) partitionSize(ctx context.Context, partitionKey string, until time.Time) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "partitionSize"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return 0, fmt.Errorf("unsupported queue shard kind for partitionSize: %s", q.primaryQueueShard.Kind)
	}

	cmd := q.primaryQueueShard.RedisClient.Client().B().Zcount().Key(partitionKey).Min("-inf").Max(strconv.Itoa(int(until.Unix()))).Build()
	return q.primaryQueueShard.RedisClient.Client().Do(ctx, cmd).AsInt64()
}

// cleanupNilPartitionInAccount is invoked when we peek a missing partition in the account partitions pointer zset.
// This happens when old executors process default function partitions that were enqueued on a new new-runs instance,
// which, in addition to the global partition pointer, enqueued the partition in the account partitions queue of queues.
// This ensures we gracefully handle inconsistencies created by the backwards compatible (keep using global partitions pointer _and_ account partitions) key queues implementation.
func (q *queue) cleanupNilPartitionInAccount(ctx context.Context, accountId uuid.UUID, partitionKey string) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "cleanupNilPartitionInAccount"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for cleanupNilPartitionInAccount: %s", q.primaryQueueShard.Kind)
	}

	// Log because this should only happen as long as we run old code
	q.log.Warn("removing account partitions pointer to missing partition",
		"partition", partitionKey,
		"account_id", accountId.String(),
	)

	cmd := q.primaryQueueShard.RedisClient.Client().B().Zrem().Key(q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(accountId)).Member(partitionKey).Build()
	if err := q.primaryQueueShard.RedisClient.Client().Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("failed to remove nil partition from account partitions pointer queue: %w", err)
	}

	// Atomically check whether account partitions is empty and remove from global accounts ZSET
	err := q.cleanupEmptyAccount(ctx, accountId)
	if err != nil {
		return fmt.Errorf("failed to check for and clean up empty account: %w", err)
	}

	return nil
}

// cleanupEmptyAccount is invoked when we peek an account without any partitions in the account pointer zset.
// This happens when old executors process default function partitions and .
// This ensures we gracefully handle inconsistencies created by the backwards compatible (keep using global partitions pointer _and_ account partitions) key queues implementation.
func (q *queue) cleanupEmptyAccount(ctx context.Context, accountId uuid.UUID) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "cleanupEmptyAccount"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for cleanupEmptyAccount: %s", q.primaryQueueShard.Kind)
	}

	if accountId == uuid.Nil {
		q.log.Warn("attempted to clean up empty account pointer with nil account ID")
		return nil
	}

	status, err := scripts["queue/cleanupEmptyAccount"].Exec(
		redis_telemetry.WithScriptName(ctx, "cleanupEmptyAccount"),
		q.primaryQueueShard.RedisClient.Client(),
		[]string{
			q.primaryQueueShard.RedisClient.kg.GlobalAccountIndex(),
			q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(accountId),
		},
		[]string{
			accountId.String(),
		},
	).ToInt64()
	if err != nil {
		return fmt.Errorf("failed to check for empty account: %w", err)
	}

	if status == 1 {
		// Log because this should only happen as long as we run old code
		q.log.Warn("removed empty account pointer", "account_id", accountId.String())
	}

	return nil
}

// partitionPeek returns pending queue partitions within the global partition pointer _or_ account partition pointer ZSET.
func (q *queue) partitionPeek(ctx context.Context, partitionKey string, sequential bool, until time.Time, limit int64, accountId *uuid.UUID) ([]*QueuePartition, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "partitionPeek"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for partitionPeek: %s", q.primaryQueueShard.Kind)
	}

	shard := q.primaryQueueShard
	client := shard.RedisClient.Client()
	kg := shard.RedisClient.kg

	if limit > PartitionPeekMax {
		return nil, ErrPartitionPeekMaxExceedsLimits
	}
	if limit <= 0 {
		limit = PartitionPeekMax
	}

	// TODO(tony): If this is an allowlist, only peek the given partitions.  Use ZMSCORE
	// to fetch the scores for all allowed partitions, then filter where score <= until.
	// Call an HMGET to get the partitions.
	ms := until.UnixMilli()

	isSequential := 0
	if sequential {
		isSequential = 1
	}

	args, err := StrSlice([]any{
		ms,
		limit,
		isSequential,
	})
	if err != nil {
		return nil, err
	}

	peekRet, err := scripts["queue/partitionPeek"].Exec(
		redis_telemetry.WithScriptName(ctx, "partitionPeek"),
		client,
		[]string{
			partitionKey,
			kg.PartitionItem(),
		},
		args,
	).ToAny()
	// NOTE: We use ToAny to force return a []any, allowing us to update the slice value with
	// a JSON-decoded item without allocations
	if err != nil {
		return nil, fmt.Errorf("error peeking partition items: %w", err)
	}
	returnedSet, ok := peekRet.([]any)
	if !ok {
		return nil, fmt.Errorf("unknown return type from partitionPeek: %T", peekRet)
	}

	var potentiallyMissingPartitions, allPartitionIds []any
	if len(returnedSet) == 3 {
		potentiallyMissingPartitions, ok = returnedSet[1].([]any)
		if !ok {
			return nil, fmt.Errorf("unexpected second item in set returned from partitionPeek: %T", peekRet)
		}

		allPartitionIds, ok = returnedSet[2].([]any)
		if !ok {
			return nil, fmt.Errorf("unexpected third item in set returned from partitionPeek: %T", peekRet)
		}
	} else if len(returnedSet) != 0 {
		return nil, fmt.Errorf("expected zero or three items in set returned by partitionPeek: %v", returnedSet)
	}

	encoded := make([]any, 0)
	missingPartitions := make([]string, 0)
	if len(potentiallyMissingPartitions) > 0 {
		for idx, partitionId := range allPartitionIds {
			if potentiallyMissingPartitions[idx] == nil {
				if partitionId == nil {
					return nil, fmt.Errorf("encountered nil partition key in pointer queue %q", partitionKey)
				}

				str, ok := partitionId.(string)
				if !ok {
					return nil, fmt.Errorf("encountered non-string partition key in pointer queue %q", partitionKey)
				}

				missingPartitions = append(missingPartitions, str)
			} else {
				encoded = append(encoded, potentiallyMissingPartitions[idx])
			}
		}
	}

	weights := []float64{}
	items := make([]*QueuePartition, len(encoded))
	fnIDs := make(map[uuid.UUID]bool)
	fnIDsMu := sync.Mutex{}

	migrateIDs := map[uuid.UUID]bool{}

	// Use parallel decoding as per Peek
	partitions, err := util.ParallelDecode(encoded, func(val any) (*QueuePartition, bool, error) {
		if val == nil {
			q.log.Error("encountered nil partition item in pointer queue",
				"encoded", encoded,
				"missing", missingPartitions,
				"key", partitionKey,
			)
			return nil, false, fmt.Errorf("encountered nil partition item in pointer queue %q", partitionKey)
		}

		str, ok := val.(string)
		if !ok {
			return nil, false, fmt.Errorf("unknown type in partition peek: %T", val)
		}

		item := &QueuePartition{}

		if err := json.Unmarshal(unsafe.Slice(unsafe.StringData(str), len(str)), item); err != nil {
			return nil, false, fmt.Errorf("error reading partition item: %w", err)
		}
		// Track the fn ID for partitions seen.  This allows us to do fast lookups of paused functions
		// to prevent peeking/working on these items as an optimization.
		if item.FunctionID != nil {
			fnIDsMu.Lock()
			fnIDs[*item.FunctionID] = false // default not paused
			fnIDsMu.Unlock()
		}
		return item, false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error decoding partitions: %w", err)
	}

	if len(missingPartitions) > 0 {
		if accountId == nil {
			return nil, fmt.Errorf("encountered missing partitions in partition pointer queue %q", partitionKey)
		}

		eg := errgroup.Group{}
		for _, partitionId := range missingPartitions {
			id := partitionId
			eg.Go(func() error {
				return q.cleanupNilPartitionInAccount(ctx, *accountId, id)
			})
		}

		if err := eg.Wait(); err != nil {
			return nil, fmt.Errorf("error cleaning up nil partitions in account pointer queue: %w", err)
		}
	}

	// mget all fn metas
	if len(fnIDs) > 0 {
		keys := make([]string, len(fnIDs))
		n := 0
		for k := range fnIDs {
			keys[n] = kg.FnMetadata(k)
			n++
		}
		vals, err := client.Do(ctx, client.B().Mget().Key(keys...).Build()).ToAny()
		if err == nil {
			// If this is an error, just ignore the error and continue.  The executor should gracefully handle
			// accidental attempts at paused functions, as we cannot do this optimization for account or env-level
			// partitions.
			vals, ok := vals.([]any)
			if !ok {
				return nil, fmt.Errorf("unknown return type from mget fnMeta: %T", vals)
			}

			_, _ = util.ParallelDecode(vals, func(i any) (any, bool, error) {
				str, ok := i.(string)
				if !ok {
					return nil, false, fmt.Errorf("unknown fnMeta type in partition peek: %T", i)
				}
				fnMeta := &FnMetadata{}
				if err := json.Unmarshal(unsafe.Slice(unsafe.StringData(str), len(str)), fnMeta); err != nil {
					return nil, false, fmt.Errorf("could not unmarshal fnMeta: %w", err)
				}

				fnIDsMu.Lock()
				fnIDs[fnMeta.FnID] = fnMeta.Paused
				if fnMeta.Migrate {
					migrateIDs[fnMeta.FnID] = true
				}
				fnIDsMu.Unlock()

				return nil, true, nil
			})
		}
	}

	ignored := 0
	for n, item := range partitions {
		// NOTE: Nil partitions were already reported above. If we got to this point, they're
		// in the account partition pointer and should simply be skipped.
		// This happens when rolling back from a newer deployment with account-queue
		// support to the previous version.
		if item == nil {
			ignored++
			continue
		}

		// check pause
		if item.FunctionID != nil {
			if paused := fnIDs[*item.FunctionID]; paused {
				// Function is pulled up when it is unpaused, so we can push it back for a long time (see SetFunctionPaused)
				err := q.PartitionRequeue(ctx, shard, item, q.clock.Now().Truncate(time.Second).Add(PartitionPausedRequeueExtension), true)
				if err != nil && !errors.Is(err, ErrPartitionGarbageCollected) {
					q.log.Error("failed to push back paused partition", "error", err, "partition", item)
				} else {
					q.log.Trace("pushed back paused partition", "partition", item.Queue())
				}

				ignored++
				continue
			}

			if _, ok := migrateIDs[*item.FunctionID]; ok {
				// skip this since the executor is not responsible for migrating queues
				ignored++
				continue
			}
		}

		// NOTE: The queue does two conflicting things:  we peek ahead of now() to fetch partitions
		// shortly available, and we also requeue partitions if there are concurrency conflicts.
		//
		// We want to ignore any partitions requeued because of conflicts, as this will cause needless
		// churn every peek MS.
		if item.ForceAtMS > ms {
			ignored++
			continue
		}

		// If we have an allowlist, only accept this partition if its in the allowlist.
		if len(q.allowQueues) > 0 && !checkList(item.Queue(), q.allowQueueMap, q.allowQueuePrefixes) {
			// This is not in the allowlist specified, so do not allow this partition to be used.
			ignored++
			continue
		}

		// Ignore any denied queues if they're explicitly in the denylist.  Because
		// we allocate the len(encoded) amount, we also want to track the number of
		// ignored queues to use the correct index when setting our items;  this ensures
		// that we don't access items with an index and get nil pointers.
		if len(q.denyQueues) > 0 && checkList(item.Queue(), q.denyQueueMap, q.denyQueuePrefixes) {
			// This is in the denylist explicitly set, so continue
			ignored++
			continue
		}

		items[n-ignored] = item
		partPriority := q.ppf(ctx, *item)
		weights = append(weights, float64(10-partPriority))
	}

	// Remove any ignored items from the slice.
	items = items[0 : len(items)-ignored]

	// Some scanners run sequentially, ensuring we always work on the functions with
	// the oldest run at times in order, no matter the priority.
	if sequential {
		n := int(math.Min(float64(len(items)), float64(PartitionSelectionMax)))
		return items[0:n], nil
	}

	// We want to weighted shuffle the resulting array random.  This means that many
	// shared nothing scanners can query for outstanding partitions and receive a
	// randomized order favouring higher-priority queue items.  This reduces the chances
	// of contention when leasing.
	w := sampleuv.NewWeighted(weights, rnd)
	result := make([]*QueuePartition, len(items))
	for n := range result {
		idx, ok := w.Take()
		if !ok {
			return nil, util.ErrWeightedSampleRead
		}
		result[n] = items[idx]
	}

	return result, nil
}

func (q *queue) accountPeek(ctx context.Context, sequential bool, until time.Time, limit int64) ([]uuid.UUID, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "accountPeek"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for accountPeek: %s", q.primaryQueueShard.Kind)
	}

	if limit > AccountPeekMax {
		return nil, ErrAccountPeekMaxExceedsLimits
	}
	if limit <= 0 {
		limit = AccountPeekMax
	}

	ms := until.UnixMilli()

	isSequential := 0
	if sequential {
		isSequential = 1
	}

	args, err := StrSlice([]any{
		ms,
		limit,
		isSequential,
	})
	if err != nil {
		return nil, err
	}

	peekRet, err := scripts["queue/accountPeek"].Exec(
		redis_telemetry.WithScriptName(ctx, "accountPeek"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		[]string{
			q.primaryQueueShard.RedisClient.kg.GlobalAccountIndex(),
		},
		args,
	).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("error peeking accounts: %w", err)
	}

	items := make([]uuid.UUID, len(peekRet))

	for i, s := range peekRet {
		parsed, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("could not parse account id from global account queue: %w", err)
		}

		items[i] = parsed
	}

	weights := make([]float64, len(items))
	for i := range items {
		accountPriority := q.apf(ctx, items[i])
		weights[i] = float64(10 - accountPriority)
	}

	// Some scanners run sequentially, ensuring we always work on the accounts with
	// the oldest run at times in order, no matter the priority.
	if sequential {
		n := int(math.Min(float64(len(items)), float64(PartitionSelectionMax)))
		return items[0:n], nil
	}

	// We want to weighted shuffle the resulting array random.  This means that many
	// shared nothing scanners can query for outstanding partitions and receive a
	// randomized order favouring higher-priority queue items.  This reduces the chances
	// of contention when leasing.
	w := sampleuv.NewWeighted(weights, rnd)
	result := make([]uuid.UUID, len(items))
	for n := range result {
		idx, ok := w.Take()
		if !ok {
			return nil, util.ErrWeightedSampleRead
		}
		result[n] = items[idx]
	}

	return result, nil
}

func checkList(check string, exact, prefixes map[string]*struct{}) bool {
	for k := range exact {
		if check == k {
			return true
		}
	}
	for k := range prefixes {
		if strings.HasPrefix(check, k) {
			return true
		}
	}
	return false
}

// PartitionRequeue requeues a parition with a new score, ensuring that the partition will be
// read at (or very close to) the given time.
//
// This is used after peeking and passing all queue items onto workers; we then take the next
// unleased available time for the queue item and requeue the partition.
//
// forceAt is used to enforce the given queue time.  This is used when partitions are at a
// concurrency limit;  we don't want to scan the partition next time, so we force the partition
// to be at a specific time instead of taking the earliest available queue item time
func (q *queue) PartitionRequeue(ctx context.Context, shard QueueShard, p *QueuePartition, at time.Time, forceAt bool) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PartitionRequeue"), redis_telemetry.ScopeQueue)

	if shard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for PartitionRequeue: %s", shard.Kind)
	}

	kg := shard.RedisClient.kg

	functionId := uuid.Nil
	if p.FunctionID != nil {
		functionId = *p.FunctionID
	}

	keys := []string{
		kg.PartitionItem(),
		kg.GlobalPartitionIndex(),
		kg.GlobalAccountIndex(),
		// NOTE: Old partitions will _not_ have an account ID until the next enqueue on the new code.
		// Until this, we may not use account queues at all, as we cannot properly clean up
		// here without knowing the Account ID
		kg.AccountPartitionIndex(p.AccountID),

		// NOTE: Partition metadata was replaced with function metadata and is being phased out
		// We clean up all remaining partition metadata on completely empty partitions here
		// and are adding function metadata on enqueue to migrate to the new system
		kg.PartitionMeta(p.Queue()),
		kg.FnMetadata(functionId),

		p.zsetKey(kg), // Partition ZSET itself
		p.concurrencyKey(kg),
		kg.QueueItem(),

		// Backlogs in shadow partition
		kg.ShadowPartitionSet(p.ID),
	}
	force := 0
	if forceAt {
		force = 1
	}
	args, err := StrSlice([]any{
		p.Queue(),
		at.UnixMilli(),
		force,
		p.AccountID.String(),
	})

	if err != nil {
		return err
	}
	status, err := scripts["queue/partitionRequeue"].Exec(
		redis_telemetry.WithScriptName(ctx, "partitionRequeue"),
		shard.RedisClient.Client(),
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error requeueing partition: %w", err)
	}

	leaseID := "n/a"
	if p.LeaseID != nil {
		leaseID = p.LeaseID.String()
	}

	q.log.Trace("requeued partition",
		"partition", p.Queue(),
		"status", status,
		"lease_id", leaseID,
		"at", at.Format(time.StampMilli),
	)

	switch status {
	case 0:
		return nil
	case 1:
		return ErrPartitionNotFound
	case 2, 3:
		return ErrPartitionGarbageCollected
	default:
		return fmt.Errorf("unknown response requeueing item: %d", status)
	}
}

// PartitionReprioritize reprioritizes a workflow's QueueItems within the queue.
func (q *queue) PartitionReprioritize(ctx context.Context, queueName string, priority uint) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PartitionReprioritize"), redis_telemetry.ScopeQueue)

	if priority > PriorityMin {
		return ErrPriorityTooLow
	}
	if priority < PriorityMax {
		return ErrPriorityTooHigh
	}

	args, err := StrSlice([]any{
		queueName,
		priority,
	})
	if err != nil {
		return err
	}

	keys := []string{q.primaryQueueShard.RedisClient.kg.PartitionItem()}
	status, err := scripts["queue/partitionReprioritize"].Exec(
		redis_telemetry.WithScriptName(ctx, "partitionReprioritize"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error enqueueing item: %w", err)
	}
	switch status {
	case 0:
		return nil
	case 1:
		return ErrPartitionNotFound
	default:
		return fmt.Errorf("unknown response reprioritizing partition: %d", status)
	}
}

func (q *queue) InProgress(ctx context.Context, prefix string, concurrencyKey string) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "InProgress"), redis_telemetry.ScopeQueue)

	s := q.clock.Now().UnixMilli()
	cmd := q.primaryQueueShard.RedisClient.unshardedRc.B().Zcount().
		Key(q.primaryQueueShard.RedisClient.kg.Concurrency(prefix, concurrencyKey)).
		Min(fmt.Sprintf("%d", s)).
		Max("+inf").
		Build()
	return q.primaryQueueShard.RedisClient.unshardedRc.Do(ctx, cmd).AsInt64()
}

func (q *queue) Instrument(ctx context.Context) error {
	// Check on global partition and queue partition sizes
	var offset, total int64
	chunkSize := int64(1000)

	r := q.primaryQueueShard.RedisClient.unshardedRc
	// iterate through all the partitions in the global partitions in chunks
	wg := sync.WaitGroup{}
	for {
		// grab the global partition by chunks
		cmd := r.B().Zrange().
			Key(q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex()).
			Min("-inf").
			Max("+inf").
			Byscore().
			Limit(offset, chunkSize).
			Build()

		pkeys, err := r.Do(ctx, cmd).AsStrSlice()
		if err != nil {
			return fmt.Errorf("error retrieving partitions for instrumentation: %w", err)
		}

		for _, pk := range pkeys {
			wg.Add(1)

			// check each partition concurrently
			go func(ctx context.Context, pkey string) {
				defer wg.Done()

				// If this is not a fully-qualified key, assume that this is an old (system) partition queue
				queueKey := pkey
				if !isKeyConcurrencyPointerItem(pkey) {
					queueKey = q.primaryQueueShard.RedisClient.kg.PartitionQueueSet(enums.PartitionTypeDefault, pkey, "")
				}

				cntCmd := r.B().Zcount().Key(queueKey).Min("-inf").Max("+inf").Build()
				count, err := q.primaryQueueShard.RedisClient.unshardedRc.Do(ctx, cntCmd).AsInt64()
				if err != nil {
					q.log.Warn("error checking partition count", "pkey", pkey, "context", "instrumentation")
					return
				}

				metrics.GaugePartitionSize(ctx, count, metrics.GaugeOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						// NOTE: potentially high cardinality but this gives better clarify of stuff
						"partition":   pkey,
						"queue_shard": q.primaryQueueShard.Name,
					},
				})

				atomic.AddInt64(&total, 1)
			}(ctx, pk)

		}
		// end of pagination, exit
		if len(pkeys) < int(chunkSize) {
			break
		}

		offset += chunkSize
	}

	// instrument the total count of global partition
	metrics.GaugeGlobalPartitionSize(ctx, atomic.LoadInt64(&total), metrics.GaugeOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard": q.primaryQueueShard.Name,
		},
	})

	wg.Wait()

	return nil
}

// isKeyPreviousConcurrencyPointerItem checks whether given string conforms to fully-qualified key as concurrency index item
func isKeyConcurrencyPointerItem(partition string) bool {
	return strings.HasPrefix(partition, "{")
}

func (q *queue) randomScavengeOffset(seed int64, count int64, limit int) int64 {
	// only apply random offset if there are more total items to scavenge than the limit
	if count > int64(limit) {
		r := mrand.New(mrand.NewSource(seed))

		// the result of count-limit must be greater than 0 as we have already checked count > limit
		// we increase the argument by 1 to make the highest possible index accessible
		// example: for count = 9, limit = 3, we want to access indices 0 through 6, not 0 through 5
		return r.Int63n(count - int64(limit) + 1)
	}

	return 0
}

// Scavenge attempts to find jobs that may have been lost due to killed workers.  Workers are shared
// nothing, and each item in a queue has a lease.  If a worker dies, it will not finish the job and
// cannot renew the item's lease.
//
// We scan all partition concurrency queues - queues of leases - to find leases that have expired.
func (q *queue) Scavenge(ctx context.Context, limit int) (int, error) {
	shard := q.primaryQueueShard

	if shard.Kind != string(enums.QueueShardKindRedis) {
		return 0, fmt.Errorf("unsupported queue shard kind for Scavenge: %s", shard.Kind)
	}

	client := shard.RedisClient.unshardedRc
	kg := shard.RedisClient.KeyGenerator()

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Scavenge"), redis_telemetry.ScopeQueue)

	// Find all items that have an expired lease - eg. where the min time for a lease is between
	// (0-now] in unix milliseconds.
	now := fmt.Sprintf("%d", q.clock.Now().UnixMilli())

	count, err := client.Do(ctx, client.B().Zcount().Key(kg.ConcurrencyIndex()).Min("-inf").Max(now).Build()).AsInt64()
	if err != nil {
		return 0, fmt.Errorf("error counting concurrency index: %w", err)
	}

	cmd := client.B().Zrange().
		Key(kg.ConcurrencyIndex()).
		Min("-inf").
		Max(now).
		Byscore().
		Limit(q.randomScavengeOffset(q.clock.Now().UnixMilli(), count, limit), int64(limit)).
		Build()

	// NOTE: Received keys can be legacy (workflow IDs or system/internal queue names) or new (full Redis keys)
	pKeys, err := client.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return 0, fmt.Errorf("error scavenging for lost items: %w", err)
	}

	counter := 0

	// Each of the items is a concurrency queue with lost items.
	var resultErr error
	for _, partition := range pKeys {
		// NOTE: If this is not a fully-qualified Redis key to a concurrency queue,
		// assume that this is an old queueName or function ID
		// This is for backwards compatibility with the previous concurrency index item format
		queueKey := partition
		if !isKeyConcurrencyPointerItem(partition) {
			queueKey = kg.Concurrency("p", partition)
		}

		// Drop key queues from concurrency pointer - these should not be in here
		if strings.HasPrefix(queueKey, "{q:v1}:concurrency:custom:") {
			err := client.Do(ctx, client.B().Zrem().Key(kg.ConcurrencyIndex()).Member(partition).Build()).Error()
			if err != nil {
				resultErr = multierror.Append(resultErr, fmt.Errorf("error removing key queue '%s' from concurrency pointer: %w", partition, err))
			}
			continue
		}

		cmd := client.B().Zrange().
			Key(queueKey).
			Min("-inf").
			Max(now).
			Byscore().
			Limit(0, ScavengeConcurrencyQueuePeekSize).
			Build()
		itemIDs, err := client.Do(ctx, cmd).AsStrSlice()
		if err != nil && err != rueidis.Nil {
			resultErr = multierror.Append(resultErr, fmt.Errorf("error querying partition concurrency queue '%s' during scavenge: %w", partition, err))
			continue
		}
		if len(itemIDs) == 0 {
			// Atomically attempt to drop empty pointer to prevent spinning on this item
			err := q.dropPartitionPointerIfEmpty(
				ctx,
				shard,
				kg.ConcurrencyIndex(),
				queueKey,
				partition,
			)
			if err != nil {
				resultErr = multierror.Append(resultErr, fmt.Errorf("error dropping empty pointer %q for partition %q: %w", partition, queueKey, err))
			}
			continue
		}

		// Fetch the queue item, then requeue.
		cmd = client.B().Hmget().Key(kg.QueueItem()).Field(itemIDs...).Build()
		jobs, err := client.Do(ctx, cmd).AsStrSlice()
		if err != nil && err != rueidis.Nil {
			resultErr = multierror.Append(resultErr, fmt.Errorf("error fetching jobs for concurrency queue '%s' during scavenge: %w", partition, err))
			continue
		}
		for i, item := range jobs {
			itemID := itemIDs[i]
			if item == "" {
				q.log.Error("missing queue item in concurrency queue",
					"index_partition", partition,
					"concurrency_queue_key", queueKey,
					"item_id", itemID,
				)

				// Drop item reference to prevent spinning on this item
				err := client.Do(ctx, client.B().Zrem().Key(queueKey).Member(itemID).Build()).Error()
				if err != nil {
					resultErr = multierror.Append(resultErr, fmt.Errorf("error removing missing item '%s' from concurrency queue '%s': %w", itemID, partition, err))
				}
				continue
			}

			qi := osqueue.QueueItem{}
			if err := json.Unmarshal([]byte(item), &qi); err != nil {
				resultErr = multierror.Append(resultErr, fmt.Errorf("error unmarshalling job '%s': %w", item, err))
				continue
			}
			if err := q.Requeue(ctx, q.primaryQueueShard, qi, q.clock.Now()); err != nil {
				resultErr = multierror.Append(resultErr, fmt.Errorf("error requeueing job '%s': %w", item, err))
				continue
			}
			counter++
		}

		if len(itemIDs) < ScavengeConcurrencyQueuePeekSize {
			// Atomically attempt to drop empty pointer if we've processed all items
			err := q.dropPartitionPointerIfEmpty(
				ctx,
				shard,
				kg.ConcurrencyIndex(),
				queueKey,
				partition,
			)
			if err != nil {
				resultErr = multierror.Append(resultErr, fmt.Errorf("error dropping potentially empty pointer %q for partition %q: %w", partition, queueKey, err))
			}
			continue
		}
	}

	return counter, resultErr
}

// ConfigLease allows a worker to lease config keys for sequential or scavenger processing.
// Leasing this key works similar to leasing partitions or queue items:
//
//   - If the key isn't leased, a new lease is accepted.
//   - If the lease is expired, a new lease is accepted.
//   - If the key is leased, you must pass in the existing lease ID to renew the lease.  Mismatches do not
//     grant a lease.
//
// This returns the new lease ID on success.
//
// If the sequential key is leased, this allows a worker to peek partitions sequentially.
func (q *queue) ConfigLease(ctx context.Context, key string, duration time.Duration, existingLeaseID ...*ulid.ULID) (*ulid.ULID, error) {
	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return nil, fmt.Errorf("unsupported queue shard kind for ConfigLease: %s", q.primaryQueueShard.Kind)
	}

	if duration > ConfigLeaseMax {
		return nil, ErrConfigLeaseExceedsLimits
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "ConfigLease"), redis_telemetry.ScopeQueue)

	now := q.clock.Now()
	newLeaseID, err := ulid.New(ulid.Timestamp(now.Add(duration)), rnd)
	if err != nil {
		return nil, err
	}

	var existing string
	if len(existingLeaseID) > 0 && existingLeaseID[0] != nil {
		existing = existingLeaseID[0].String()
	}

	args, err := StrSlice([]any{
		now.UnixMilli(),
		newLeaseID.String(),
		existing,
	})
	if err != nil {
		return nil, err
	}

	status, err := scripts["queue/configLease"].Exec(
		redis_telemetry.WithScriptName(ctx, "configLease"),
		q.primaryQueueShard.RedisClient.unshardedRc,
		[]string{key},
		args,
	).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error claiming config lease: %w", err)
	}
	switch status {
	case 0:
		return &newLeaseID, nil
	case 1:
		return nil, ErrConfigAlreadyLeased
	default:
		return nil, fmt.Errorf("unknown response claiming config lease: %d", status)
	}
}

// peekEWMA returns the calculated EWMA value from the list
// nolint:unused // this code remains to be enabled on demand
func (q *queue) peekEWMA(ctx context.Context, fnID uuid.UUID) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "peekEWMA"), redis_telemetry.ScopeQueue)

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return 0, fmt.Errorf("unsupported queue shard kind for peekEWMA: %s", q.primaryQueueShard.Kind)
	}

	// retrieves the list from redis
	cmd := q.primaryQueueShard.RedisClient.Client().B().Lrange().Key(q.primaryQueueShard.RedisClient.KeyGenerator().ConcurrencyFnEWMA(fnID)).Start(0).Stop(-1).Build()
	strlist, err := q.primaryQueueShard.RedisClient.Client().Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return 0, fmt.Errorf("error reading function concurrency EWMA values: %w", err)
	}

	// return early
	if len(strlist) == 0 {
		return 0, nil
	}

	hasNonZero := false
	vals := make([]float64, len(strlist))
	for i, s := range strlist {
		v, _ := strconv.ParseFloat(s, 64)
		vals[i] = v
		if v > 0 {
			hasNonZero = true
		}
	}

	if !hasNonZero {
		// short-circuit.
		return 0, nil
	}

	// create a simple EWMA, add all the numbers in it and get the final value
	// NOTE: we don't need variable since we don't want to maintain this in memory
	mavg := ewma.NewMovingAverage()
	for _, v := range vals {
		mavg.Add(v)
	}

	// round up to the nearest integer
	return int64(math.Round(mavg.Value())), nil
}

// setPeekEWMA add the new value to the existing list.
// if the length of the list exceeds the predetermined size, pop out the first item
func (q *queue) setPeekEWMA(ctx context.Context, fnID *uuid.UUID, val int64) error {
	if fnID == nil {
		return nil
	}

	if q.primaryQueueShard.Kind != string(enums.QueueShardKindRedis) {
		return fmt.Errorf("unsupported queue shard kind for setPeekEWMA: %s", q.primaryQueueShard.Kind)
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "setPeekEWMA"), redis_telemetry.ScopeQueue)

	listSize := q.peekEWMALen
	if listSize == 0 {
		listSize = QueuePeekEWMALen
	}

	keys := []string{
		q.primaryQueueShard.RedisClient.kg.ConcurrencyFnEWMA(*fnID),
	}
	args, err := StrSlice([]any{
		val,
		listSize,
	})
	if err != nil {
		return err
	}

	_, err = scripts["queue/setPeekEWMA"].Exec(
		redis_telemetry.WithScriptName(ctx, "setPeekEWMA"),
		q.primaryQueueShard.RedisClient.Client(),
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error updating function concurrency EWMA: %w", err)
	}

	return nil
}

// addContinue adds a continuation for the given partition.  This hints that the queue should
// peek and process this partition on the next loop, allowing us to hint that a partition
// should be processed when a step finishes (to decrease inter-step latency on non-connect
// workloads).
func (q *queue) addContinue(ctx context.Context, p *QueuePartition, ctr uint) {
	if !q.runMode.Continuations {
		// continuations are not enabled.
		return
	}

	if ctr >= q.continuationLimit {
		q.removeContinue(ctx, p, true)
		return
	}

	q.continuesLock.Lock()
	defer q.continuesLock.Unlock()

	// If this is the first continuation, check if we're on a cooldown, or if we're
	// beyond capacity.
	if ctr == 1 {
		if len(q.continues) > consts.QueueContinuationMaxPartitions {
			metrics.IncrQueueContinuationMaxCapcityCounter(ctx, metrics.CounterOpt{PkgName: pkgName})
			return
		}
		if t, ok := q.continueCooldown[p.Queue()]; ok && t.After(time.Now()) {
			metrics.IncrQueueContinuationCooldownCounter(ctx, metrics.CounterOpt{PkgName: pkgName})
			return
		}

		// Remove the continuation cooldown.
		delete(q.continueCooldown, p.Queue())
	}

	c, ok := q.continues[p.Queue()]
	if !ok || c.count < ctr {
		// Update the continue count if it doesn't exist, or the current counter
		// is higher.  This ensures that we always have the highest continuation
		// count stored for queue processing.
		q.continues[p.Queue()] = continuation{partition: p, count: ctr}
		metrics.IncrQueueContinuationAddedCounter(ctx, metrics.CounterOpt{PkgName: pkgName})
	}
}

func (q *queue) removeContinue(ctx context.Context, p *QueuePartition, cooldown bool) {
	if !q.runMode.Continuations {
		// continuations are not enabled.
		return
	}

	// This is over the limit for conntinuing the partition, so force it to be
	// removed in every case.
	q.continuesLock.Lock()
	defer q.continuesLock.Unlock()

	metrics.IncrQueueContinuationRemovedCounter(ctx, metrics.CounterOpt{PkgName: pkgName})

	delete(q.continues, p.Queue())

	if cooldown {
		// Add a cooldown, preventing this partition from being added as a continuation
		// for a given period of time.
		//
		// Note that this isn't shared across replicas;  cooldowns
		// only exist in the current replica.
		q.continueCooldown[p.Queue()] = time.Now().Add(
			consts.QueueContinuationCooldownPeriod,
		)
	}
}

//nolint:all
func (q *queue) readFnMetadata(ctx context.Context, fnID uuid.UUID) (*FnMetadata, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "readFnMetadata"), redis_telemetry.ScopeQueue)

	cmd := q.primaryQueueShard.RedisClient.unshardedRc.B().Get().Key(q.primaryQueueShard.RedisClient.kg.FnMetadata(fnID)).Build()
	retv := FnMetadata{}
	err := q.primaryQueueShard.RedisClient.unshardedRc.Do(ctx, cmd).DecodeJSON(&retv)
	if err != nil {
		return nil, fmt.Errorf("error reading function metadata: %w", err)
	}
	return &retv, nil
}

func newLeaseDenyList() *leaseDenies {
	return &leaseDenies{
		lock:        &sync.RWMutex{},
		concurrency: map[string]struct{}{},
		throttle:    map[string]struct{}{},
	}
}

// leaseDenies stores a mapping of keys that must not be leased.
//
// When iterating over a list of peeked queue items, each queue item may have the same
// or different concurrency keys.  As soon as one of these concurrency keys reaches its
// limit, any next queue items with the same keys must _never_ be considered for leasing.
//
// This has two benefits:  we prevent wasted work, and we prevent out of order work.
type leaseDenies struct {
	lock *sync.RWMutex

	concurrency map[string]struct{}
	throttle    map[string]struct{}
}

func (l *leaseDenies) addThrottled(err error) {
	var key keyError
	if !errors.As(err, &key) {
		return
	}
	l.lock.Lock()
	l.throttle[key.key] = struct{}{}
	l.lock.Unlock()
}

func (l *leaseDenies) addConcurrency(err error) {
	var key keyError
	if !errors.As(err, &key) {
		return
	}
	l.lock.Lock()
	l.concurrency[key.key] = struct{}{}
	l.lock.Unlock()
}

func (l *leaseDenies) denyConcurrency(key string) bool {
	l.lock.RLock()
	_, ok := l.concurrency[key]
	l.lock.RUnlock()
	return ok
}

func (l *leaseDenies) denyThrottle(key string) bool {
	l.lock.RLock()
	_, ok := l.throttle[key]
	l.lock.RUnlock()
	return ok
}
