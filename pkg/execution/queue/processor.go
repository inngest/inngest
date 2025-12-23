package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/VividCortex/ewma"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/backoff"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
)

var (
	latencyAvg ewma.MovingAverage
	latencySem *sync.Mutex
)

func init() {
	latencyAvg = ewma.NewMovingAverage()
	latencySem = &sync.Mutex{}
}

func NewQueueProcessor(
	ctx context.Context,
	name string,
	primaryQueueShard QueueShard,
	options ...QueueOpt,
) (Queue, error) {
	o := &QueueOptions{
		PrimaryQueueShard: primaryQueueShard,
		QueueShardClients: map[string]QueueShard{primaryQueueShard.Name(): primaryQueueShard},
		ppf: func(_ context.Context, _ QueuePartition) uint {
			return PriorityDefault
		},
		apf: func(_ context.Context, _ uuid.UUID) uint {
			return PriorityDefault
		},
		partitionPausedGetter: func(ctx context.Context, fnID uuid.UUID) PartitionPausedInfo {
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
		activeCheckTick:                defaultActiveCheckTick,
		idempotencyTTL:                 defaultIdempotencyTTL,
		queueKindMapping:               make(map[string]string),
		peekSizeForFunctions:           make(map[string]int64),
		log:                            logger.StdlibLogger(ctx),
		instrumentInterval:             DefaultInstrumentInterval,
		PartitionConstraintConfigGetter: func(ctx context.Context, pi PartitionIdentifier) PartitionConstraintConfig {
			def := defaultConcurrency

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
		tenantInstrumentor: func(ctx context.Context, partitionID string) error {
			return nil
		},
		backoffFunc:             backoff.DefaultBackoff,
		Clock:                   clockwork.NewRealClock(),
		continuationLimit:       consts.DefaultQueueContinueLimit,
		shadowContinuesLock:     &sync.Mutex{},
		shadowContinuationLimit: consts.DefaultQueueContinueLimit,
		shadowContinues:         map[string]shadowContinuation{},
		shadowContinueCooldown:  map[string]time.Time{},
		normalizeRefreshItemCustomConcurrencyKeys: func(ctx context.Context, item *QueueItem, existingKeys []state.CustomConcurrency, shadowPartition *QueueShadowPartition) ([]state.CustomConcurrency, error) {
			return existingKeys, nil
		},
		refreshItemThrottle: func(ctx context.Context, item *QueueItem) (*Throttle, error) {
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
		CapacityLeaseExtendInterval:   QueueLeaseDuration / 2,
	}

	// default to using primary queue client for shard selection
	o.shardSelector = func(_ context.Context, _ uuid.UUID, _ *string) (QueueShard, error) {
		return o.PrimaryQueueShard, nil
	}

	for _, qopt := range options {
		qopt(o)
	}

	qp := &queueProcessor{
		name: name,

		QueueOptions: o,

		wg:                       &sync.WaitGroup{},
		seqLeaseLock:             &sync.RWMutex{},
		scavengerLeaseLock:       &sync.RWMutex{},
		activeCheckerLeaseLock:   &sync.RWMutex{},
		instrumentationLeaseLock: &sync.RWMutex{},

		continuesLock:    &sync.Mutex{},
		continues:        map[string]continuation{},
		continueCooldown: map[string]time.Time{},

		sem:     util.NewTrackingSemaphore(int(o.numWorkers)),
		workers: make(chan processItem, o.numWorkers),
		quit:    make(chan error, o.numWorkers),
	}

	return qp, nil
}

type queueProcessor struct {
	*QueueOptions

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
	sem util.TrackingSemaphore

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

// OutstandingJobCount implements Queue.
func (q *queueProcessor) OutstandingJobCount(ctx context.Context, envID uuid.UUID, fnID uuid.UUID, runID ulid.ULID) (int, error) {
	panic("unimplemented")
}

// ResetAttemptsByJobID implements Queue.
func (q *queueProcessor) ResetAttemptsByJobID(ctx context.Context, shard string, jobID string) error {
	panic("unimplemented")
}

// Run implements Queue.
func (q *queueProcessor) Run(context.Context, RunFunc) error {
	panic("unimplemented")
}

// RunJobs implements Queue.
func (q *queueProcessor) RunJobs(ctx context.Context, queueShardName string, workspaceID uuid.UUID, workflowID uuid.UUID, runID ulid.ULID, limit int64, offset int64) ([]JobResponse, error) {
	panic("unimplemented")
}

// RunningCount implements Queue.
func (q *queueProcessor) RunningCount(ctx context.Context, workflowID uuid.UUID) (int64, error) {
	panic("unimplemented")
}

// SetFunctionMigrate implements Queue.
func (q *queueProcessor) SetFunctionMigrate(ctx context.Context, sourceShard string, fnID uuid.UUID, migrateLockUntil *time.Time) error {
	shard, ok := q.QueueShardClients[sourceShard]
	if !ok {
		return fmt.Errorf("could not find shard %q", sourceShard)
	}

	return shard.Processor().SetFunctionMigrate(ctx, sourceShard, fnID uuid.UUID, migrateLockUntil *time.Time)
}

// StatusCount implements Queue.
func (q *queueProcessor) StatusCount(ctx context.Context, workflowID uuid.UUID, status string) (int64, error) {
	panic("unimplemented")
}

// UnpauseFunction implements Queue.
func (q *queueProcessor) UnpauseFunction(ctx context.Context, shard string, acctID uuid.UUID, fnID uuid.UUID) error {
	panic("unimplemented")
}
