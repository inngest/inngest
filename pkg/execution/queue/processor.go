package queue

import (
	"context"
	"fmt"
	"iter"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VividCortex/ewma"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/errgroup"
)

var (
	latencyAvg ewma.MovingAverage
	latencySem *sync.Mutex
)

func init() {
	latencyAvg = ewma.NewMovingAverage()
	latencySem = &sync.Mutex{}
}

func LatencySem() *sync.Mutex {
	return latencySem
}

func LatencyAverage() float64 {
	return latencyAvg.Value()
}

func New(
	ctx context.Context,
	name string,
	primaryQueueShard QueueShard,
	queueShardClients map[string]QueueShard,
	shardSelector ShardSelector,
	options ...QueueOpt,
) (*queueProcessor, error) {
	o := NewQueueOptions(options...)

	qp := &queueProcessor{
		name: name,

		QueueOptions: o,

		wg:                       &sync.WaitGroup{},
		seqLeaseLock:             &sync.RWMutex{},
		scavengerLeaseLock:       &sync.RWMutex{},
		activeCheckerLeaseLock:   &sync.RWMutex{},
		instrumentationLeaseLock: &sync.RWMutex{},
		shardLeaseLock:           &sync.RWMutex{},

		continuesLock:    &sync.Mutex{},
		continues:        map[string]continuation{},
		continueCooldown: map[string]time.Time{},

		sem:     util.NewTrackingSemaphore(int(o.numWorkers)),
		workers: make(chan ProcessItem, o.numWorkers),
		quit:    make(chan error, o.numWorkers),

		primaryQueueShard: primaryQueueShard,
		queueShardClients: queueShardClients,
		shardSelector:     shardSelector,

		qspc: make(chan ShadowPartitionChanMsg),

		shadowContinuesLock:    &sync.Mutex{},
		shadowContinues:        map[string]ShadowContinuation{},
		shadowContinueCooldown: map[string]time.Time{},
	}

	if primaryQueueShard != nil {
		qp.SetPrimaryShard(ctx, primaryQueueShard)
	} else if o.runMode.ShardGroup == "" {
		return nil, fmt.Errorf("must pass either primary queue shard or a valid ShardGroup in runMode")
	} else if len(qp.shardsByGroupName(o.runMode.ShardGroup)) == 0 {
		return nil, fmt.Errorf("No shards found for configured shard group: %s", o.runMode.ShardGroup)
	}

	return qp, nil
}

type queueProcessor struct {
	*QueueOptions

	// name is the identifiable name for this worker, for logging.
	name string

	queueShardClients map[string]QueueShard
	shardSelector     ShardSelector

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
	workers chan ProcessItem

	qspc chan ShadowPartitionChanMsg

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

	primaryQueueShard QueueShard
	// shardLeaseID stores the lease ID for the primaryQueueShard this queue is processing from.
	// all runners attempt to claim this lease on start up.
	shardLeaseID *ulid.ULID
	// shardLeaseLock ensures that there are no data races writing to
	// or reading from shardLeaseID in parallel.
	shardLeaseLock *sync.RWMutex

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

	shadowContinues        map[string]ShadowContinuation
	shadowContinueCooldown map[string]time.Time
	shadowContinuesLock    *sync.Mutex
}

func (q *queueProcessor) GetShadowContinuations() map[string]ShadowContinuation {
	q.shadowContinuesLock.Lock()
	defer q.shadowContinuesLock.Unlock()

	return q.shadowContinues
}

func (q *queueProcessor) ClearShadowContinuations() {
	q.shadowContinuesLock.Lock()
	defer q.shadowContinuesLock.Unlock()

	clear(q.shadowContinues)
	clear(q.shadowContinueCooldown)
}

func (q *queueProcessor) Clock() clockwork.Clock {
	return q.QueueOptions.Clock
}

func (q *queueProcessor) Shard() QueueShard {
	return q.primaryQueueShard
}

// Implements SetPrimaryShard() in ShardAssingmentManager interface
func (q *queueProcessor) SetPrimaryShard(ctx context.Context, queueShard QueueShard) {

	q.primaryQueueShard = queueShard

	if q.queueShardClients == nil {
		q.queueShardClients = map[string]QueueShard{
			queueShard.Name(): queueShard,
		}
	}

	if q.shardSelector == nil {
		q.shardSelector = func(ctx context.Context, accountId uuid.UUID, queueName *string) (QueueShard, error) {
			return queueShard, nil
		}
	}
}

func (q *queueProcessor) Semaphore() util.TrackingSemaphore {
	return q.sem
}

func (q *queueProcessor) Options() *QueueOptions {
	return q.QueueOptions
}

func (q *queueProcessor) Workers() chan ProcessItem {
	return q.workers
}

// BacklogSize implements QueueManager.
func (q *queueProcessor) BacklogSize(ctx context.Context, shard QueueShard, backlogID string) (int64, error) {
	return shard.BacklogSize(ctx, backlogID)
}

// BacklogsByPartition implements QueueManager.
func (q *queueProcessor) BacklogsByPartition(ctx context.Context, shard QueueShard, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueBacklog], error) {
	return shard.BacklogsByPartition(ctx, partitionID, from, until, opts...)
}

// Dequeue implements QueueManager.
func (q *queueProcessor) Dequeue(ctx context.Context, shard QueueShard, i QueueItem, opts ...DequeueOptionFn) error {
	return shard.Dequeue(ctx, i, opts...)
}

// ItemByID implements QueueManager.
func (q *queueProcessor) ItemByID(ctx context.Context, shard QueueShard, jobID string) (*QueueItem, error) {
	return shard.ItemByID(ctx, jobID)
}

// ItemExists implements QueueManager.
func (q *queueProcessor) ItemExists(ctx context.Context, shard QueueShard, jobID string) (bool, error) {
	return shard.ItemExists(ctx, jobID)
}

// ItemsByBacklog implements QueueManager.
func (q *queueProcessor) ItemsByBacklog(ctx context.Context, shard QueueShard, backlogID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error) {
	return shard.ItemsByBacklog(ctx, backlogID, from, until, opts...)
}

// ItemsByPartition implements QueueManager.
func (q *queueProcessor) ItemsByPartition(ctx context.Context, shard QueueShard, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error) {
	return shard.ItemsByPartition(ctx, partitionID, from, until, opts...)
}

// ItemsByRunID implements QueueManager.
func (q *queueProcessor) ItemsByRunID(ctx context.Context, shard QueueShard, runID ulid.ULID) ([]*QueueItem, error) {
	return shard.ItemsByRunID(ctx, runID)
}

func (q *queueProcessor) shardByName(name string) (QueueShard, error) {
	shard, ok := q.queueShardClients[name]
	if !ok {
		return nil, ErrQueueShardNotFound
	}
	return shard, nil
}

func (q *queueProcessor) shardsByGroupName(groupName string) []QueueShard {
	var shards []QueueShard
	for _, shard := range q.queueShardClients {
		if shard.ShardAssignmentConfig().ShardGroup == groupName {
			shards = append(shards, shard)
		}
	}
	return shards
}

// LoadQueueItem implements QueueManager.
func (q *queueProcessor) LoadQueueItem(ctx context.Context, shardName string, itemID string) (*QueueItem, error) {
	shard, err := q.shardByName(shardName)
	if err != nil {
		return nil, err
	}

	return shard.LoadQueueItem(ctx, itemID)
}

// PartitionBacklogSize implements QueueManager.
func (q *queueProcessor) PartitionBacklogSize(ctx context.Context, partitionID string) (int64, error) {
	var totalCount int64

	err := q.AllShards(ctx, func(ctx context.Context, shard QueueShard) error {
		backlogSize, err := shard.PartitionBacklogSize(ctx, partitionID)
		if err != nil {
			return fmt.Errorf("could not load partition backlog size: %w", err)
		}
		l := logger.StdlibLogger(ctx)
		l.Trace("retrieved backlog size", "size", backlogSize)
		atomic.AddInt64(&totalCount, int64(backlogSize))
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("could not load partition backlog size: %w", err)
	}
	return totalCount, nil
}

// PartitionByID implements QueueManager.
func (q *queueProcessor) PartitionByID(ctx context.Context, shard QueueShard, partitionID string) (*PartitionInspectionResult, error) {
	return shard.PartitionByID(ctx, partitionID)
}

// RemoveQueueItem implements QueueManager.
func (q *queueProcessor) RemoveQueueItem(ctx context.Context, shardName string, partitionID string, itemID string) error {
	shard, err := q.shardByName(shardName)
	if err != nil {
		return err
	}

	return shard.RemoveQueueItem(ctx, partitionID, itemID)
}

// Requeue implements QueueManager.
func (q *queueProcessor) Requeue(ctx context.Context, shard QueueShard, i QueueItem, at time.Time, opts ...RequeueOptionFn) error {
	return shard.Requeue(ctx, i, at, opts...)
}

// RequeueByJobID implements QueueManager.
func (q *queueProcessor) RequeueByJobID(ctx context.Context, shard QueueShard, jobID string, at time.Time) error {
	return shard.RequeueByJobID(ctx, jobID, at)
}

// TotalSystemQueueDepth implements QueueManager.
func (q *queueProcessor) TotalSystemQueueDepth(ctx context.Context, shard QueueShard) (int64, error) {
	return shard.TotalSystemQueueDepth(ctx)
}

func (q *queueProcessor) AllShards(ctx context.Context, fn func(context.Context, QueueShard) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	l := logger.StdlibLogger(ctx)

	eg, ctx := errgroup.WithContext(ctx)

	for shardName, qs := range q.queueShardClients {
		eg.Go(func() error {
			l := l.With("shard_name", shardName)
			err := fn(logger.WithStdlib(ctx, l), qs)
			if err != nil {
				return fmt.Errorf("map operation on shard %q failed: %w", shardName, err)
			}
			return nil
		})
	}

	return eg.Wait()
}

// OutstandingJobCount implements Queue.
func (q *queueProcessor) OutstandingJobCount(ctx context.Context, envID uuid.UUID, fnID uuid.UUID, runID ulid.ULID) (int, error) {
	var totalCount int64

	err := q.AllShards(ctx, func(ctx context.Context, shard QueueShard) error {
		outstanding, err := shard.OutstandingJobCount(ctx, envID, fnID, runID)
		if err != nil {
			return fmt.Errorf("could not load outstanding job count: %w", err)
		}
		atomic.AddInt64(&totalCount, int64(outstanding))
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("could not load outstanding count: %w", err)
	}
	return int(totalCount), nil
}

// RunJobs implements Queue.
func (q *queueProcessor) RunJobs(ctx context.Context, shardName string, workspaceID uuid.UUID, workflowID uuid.UUID, runID ulid.ULID, limit int64, offset int64) ([]JobResponse, error) {
	shard, err := q.shardByName(shardName)
	if err != nil {
		return nil, err
	}

	return shard.RunJobs(ctx, workspaceID, workflowID, runID, limit, offset)
}

// RunningCount implements Queue.
func (q *queueProcessor) RunningCount(ctx context.Context, workflowID uuid.UUID) (int64, error) {
	var totalCount int64

	err := q.AllShards(ctx, func(ctx context.Context, shard QueueShard) error {
		running, err := shard.RunningCount(ctx, workflowID)
		if err != nil {
			return fmt.Errorf("could not load running count: %w", err)
		}
		atomic.AddInt64(&totalCount, int64(running))
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("could not load running count: %w", err)
	}
	return totalCount, nil
}

// StatusCount implements Queue.
func (q *queueProcessor) StatusCount(ctx context.Context, workflowID uuid.UUID, status string) (int64, error) {
	var totalCount int64

	err := q.AllShards(ctx, func(ctx context.Context, shard QueueShard) error {
		running, err := shard.StatusCount(ctx, workflowID, status)
		if err != nil {
			return fmt.Errorf("could not load status count: %w", err)
		}
		atomic.AddInt64(&totalCount, int64(running))
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("could not load status count: %w", err)
	}
	return totalCount, nil
}

// ResetAttemptsByJobID implements Queue.
func (q *queueProcessor) ResetAttemptsByJobID(ctx context.Context, shardName string, jobID string) error {
	shard, err := q.shardByName(shardName)
	if err != nil {
		return err
	}

	return shard.ResetAttemptsByJobID(ctx, jobID)
}

func (q *queueProcessor) Run(ctx context.Context, f RunFunc) error {
	// claimShardLease will block until a shard lease is obtained to process the primaryQueueShard.
	l := logger.StdlibLogger(ctx)
	if len(q.runMode.ShardGroup) != 0 {
		l.Info("Executor started in ShardGroup mode, attempting to claim a shard lease", "shard_group", q.runMode.ShardGroup)
		q.claimShardLease(ctx)
	} else {
		l.Info("Executor started in assignedQueueShard Mode", "queue_shard", q.primaryQueueShard.Name())
	}

	if q.runMode.Sequential {
		go q.claimSequentialLease(ctx)
	}

	if q.runMode.Scavenger {
		go q.runScavenger(ctx)
	}

	if q.runMode.ActiveChecker {
		go q.runActiveChecker(ctx)
	}

	go q.runInstrumentation(ctx)

	// start execution and shadow scan concurrently
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return q.executionScan(ctx, f)
	})

	if q.runMode.ShadowPartition {
		eg.Go(func() error {
			return q.shadowScan(ctx)
		})
	}

	if q.runMode.NormalizePartition {
		eg.Go(func() error {
			return q.backlogNormalizationScan(ctx)
		})
	}

	return eg.Wait()
}

// SetFunctionMigrate implements Queue.
func (q *queueProcessor) SetFunctionMigrate(ctx context.Context, sourceShard string, fnID uuid.UUID, migrateLockUntil *time.Time) error {
	shard, ok := q.queueShardClients[sourceShard]
	if !ok {
		return fmt.Errorf("could not find shard %q", sourceShard)
	}

	return shard.SetFunctionMigrate(ctx, fnID, migrateLockUntil)
}

// UnpauseFunction implements Queue.
func (q *queueProcessor) UnpauseFunction(ctx context.Context, shardName string, acctID uuid.UUID, fnID uuid.UUID) error {
	shard, err := q.shardByName(shardName)
	if err != nil {
		return err
	}
	return shard.UnpauseFunction(ctx, acctID, fnID)
}

func (q *queueProcessor) capacity() int64 {
	return int64(q.numWorkers) - q.Semaphore().Count()
}
