package queue

import (
	"context"
	"fmt"
	"iter"
	"sync"
	"time"

	"github.com/VividCortex/ewma"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/util"
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

func NewQueueProcessor(
	ctx context.Context,
	name string,
	primaryQueueShard QueueShard,
	queueShardClients map[string]QueueShard,
	shardSelector ShardSelector,
	options ...QueueOpt,
) (*queueProcessor, error) {
	o := NewQueueOptions(ctx, options...)

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
		workers: make(chan ProcessItem, o.numWorkers),
		quit:    make(chan error, o.numWorkers),

		primaryQueueShard: primaryQueueShard,
		queueShardClients: queueShardClients,
		shardSelector:     shardSelector,
	}

	return qp, nil
}

type queueProcessor struct {
	*QueueOptions

	// name is the identifiable name for this worker, for logging.
	name string

	primaryQueueShard QueueShard
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

// BacklogSize implements QueueManager.
func (q *queueProcessor) BacklogSize(ctx context.Context, queueShard QueueShard, backlogID string) (int64, error) {
	panic("unimplemented")
}

// BacklogsByPartition implements QueueManager.
func (q *queueProcessor) BacklogsByPartition(ctx context.Context, queueShard QueueShard, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueBacklog], error) {
	panic("unimplemented")
}

// Dequeue implements QueueManager.
func (q *queueProcessor) Dequeue(ctx context.Context, queueShard QueueShard, i QueueItem, opts ...DequeueOptionFn) error {
	panic("unimplemented")
}

// DequeueByJobID implements QueueManager.
func (q *queueProcessor) DequeueByJobID(ctx context.Context, jobID string) error {
	panic("unimplemented")
}

// ItemByID implements QueueManager.
func (q *queueProcessor) ItemByID(ctx context.Context, shard QueueShard, jobID string) (*QueueItem, error) {
	panic("unimplemented")
}

// ItemExists implements QueueManager.
func (q *queueProcessor) ItemExists(ctx context.Context, shard QueueShard, jobID string) (bool, error) {
	panic("unimplemented")
}

// ItemsByBacklog implements QueueManager.
func (q *queueProcessor) ItemsByBacklog(ctx context.Context, queueShard QueueShard, backlogID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error) {
	panic("unimplemented")
}

// ItemsByPartition implements QueueManager.
func (q *queueProcessor) ItemsByPartition(ctx context.Context, queueShard QueueShard, partitionID string, from time.Time, until time.Time, opts ...QueueIterOpt) (iter.Seq[*QueueItem], error) {
	panic("unimplemented")
}

// ItemsByRunID implements QueueManager.
func (q *queueProcessor) ItemsByRunID(ctx context.Context, shard QueueShard, runID ulid.ULID) ([]*QueueItem, error) {
	panic("unimplemented")
}

// LoadQueueItem implements QueueManager.
func (q *queueProcessor) LoadQueueItem(ctx context.Context, shard string, itemID string) (*QueueItem, error) {
	panic("unimplemented")
}

// PartitionBacklogSize implements QueueManager.
func (q *queueProcessor) PartitionBacklogSize(ctx context.Context, shard QueueShard, partitionID string) (int64, error) {
	panic("unimplemented")
}

// PartitionByID implements QueueManager.
func (q *queueProcessor) PartitionByID(ctx context.Context, queueShard QueueShard, partitionID string) (*PartitionInspectionResult, error) {
	panic("unimplemented")
}

// RemoveQueueItem implements QueueManager.
func (q *queueProcessor) RemoveQueueItem(ctx context.Context, shard string, partitionKey string, itemID string) error {
	panic("unimplemented")
}

// Requeue implements QueueManager.
func (q *queueProcessor) Requeue(ctx context.Context, queueShard QueueShard, i QueueItem, at time.Time, opts ...RequeueOptionFn) error {
	panic("unimplemented")
}

// RequeueByJobID implements QueueManager.
func (q *queueProcessor) RequeueByJobID(ctx context.Context, queueShard QueueShard, jobID string, at time.Time) error {
	panic("unimplemented")
}

// TotalSystemQueueDepth implements QueueManager.
func (q *queueProcessor) TotalSystemQueueDepth(ctx context.Context, shard QueueShard) (int64, error) {
	panic("unimplemented")
}

// OutstandingJobCount implements Queue.
func (q *queueProcessor) OutstandingJobCount(ctx context.Context, envID uuid.UUID, fnID uuid.UUID, runID ulid.ULID) (int, error) {
	panic("unimplemented")
}

// ResetAttemptsByJobID implements Queue.
func (q *queueProcessor) ResetAttemptsByJobID(ctx context.Context, shard string, jobID string) error {
	panic("unimplemented")
}

func (q *queueProcessor) Run(ctx context.Context, f RunFunc) error {
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
	shard, ok := q.queueShardClients[sourceShard]
	if !ok {
		return fmt.Errorf("could not find shard %q", sourceShard)
	}

	return shard.SetFunctionMigrate(ctx, fnID, migrateLockUntil)
}

// StatusCount implements Queue.
func (q *queueProcessor) StatusCount(ctx context.Context, workflowID uuid.UUID, status string) (int64, error) {
	panic("unimplemented")
}

// UnpauseFunction implements Queue.
func (q *queueProcessor) UnpauseFunction(ctx context.Context, shard string, acctID uuid.UUID, fnID uuid.UUID) error {
	panic("unimplemented")
}

func (q *queueProcessor) capacity() int64 {
	return int64(q.numWorkers) - q.sem.Count()
}
