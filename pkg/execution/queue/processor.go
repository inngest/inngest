package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/VividCortex/ewma"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
	"github.com/jonboulle/clockwork"
	"github.com/karlseguin/ccache/v3"
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
	shards QueueShardRegistry,
	options ...QueueOpt,
) (*queueProcessor, error) {
	o := NewQueueOptions(options...)

	if shards == nil {
		return nil, fmt.Errorf("shard registry must not be nil")
	}

	qp := &queueProcessor{
		name:         name,
		QueueOptions: o,

		wg:             &sync.WaitGroup{},
		roleLeaseLock:  &sync.RWMutex{},
		roleLeaseIDs:   map[string]*ulid.ULID{},
		shardLeaseLock: &sync.RWMutex{},

		continuesLock:    &sync.Mutex{},
		continues:        map[string]continuation{},
		continueCooldown: map[string]time.Time{},

		sem:          util.NewTrackingSemaphore(int(o.numWorkers)),
		workers:      make(chan ProcessItem, o.numWorkers),
		partitionSem: util.NewTrackingSemaphore(int(o.numPartitionWorkers)),
		quit:         make(chan error, o.numWorkers),

		shards: shards,
		Producer: NewProducer(
			shards,
			WithProducerClock(o.Clock),
			WithProducerKindToQueueMapping(o.queueKindMapping),
			WithProducerJobPromotion(o.enableJobPromotion),
			WithProducerConditionalTracer(o.ConditionalTracer),
		),
		Consumer:        NewConsumer(shards),
		JobQueueReader:  newJobQueueReader(shards, o.AccountShardIterationEnabled),
		Migrator:        newQueueMigrator(shards, o.Clock),
		Unpauser:        newQueueUnpauser(shards),
		AttemptResetter: newAttemptResetter(shards),

		peekSizeCache: ccache.New(ccache.Configure[int64]().MaxSize(50_000)),

		qspc: make(chan ShadowPartitionChanMsg),

		shadowContinuesLock:    &sync.Mutex{},
		shadowContinues:        map[string]ShadowContinuation{},
		shadowContinueCooldown: map[string]time.Time{},
	}

	if shards.Primary() == nil {
		if o.runMode.ShardGroup == "" {
			return nil, fmt.Errorf("must pass either primary queue shard or a valid ShardGroup in runMode")
		}
		if len(shards.ByGroup(o.runMode.ShardGroup)) == 0 {
			return nil, fmt.Errorf("No shards found for configured shard group: %s", o.runMode.ShardGroup)
		}
	}
	if o.queueProducer != nil {
		qp.Producer = o.queueProducer
	}
	if o.queueConsumer != nil {
		qp.Consumer = o.queueConsumer
	}
	qp.configureQueueRoles()

	return qp, nil
}

type queueProcessor struct {
	*QueueOptions

	Producer
	Consumer
	JobQueueReader
	Migrator
	Unpauser
	AttemptResetter

	// name is the identifiable name for this worker, for logging.
	name string

	// shards owns the {shards map, selector, primary} trio. Topology can be
	// mutated at runtime via shards.SetPrimary.
	shards QueueShardRegistry

	// quit is a channel that any method can send on to trigger termination
	// of the Run loop.  This typically accepts an error, but a nil error
	// will still quit the runner.
	quit chan error
	// wg stores a waitgroup for all in-progress jobs
	wg *sync.WaitGroup

	// workers is a buffered channel which allows scanners to send queue items
	// to workers to be processed
	workers chan ProcessItem

	// partitionSem tracks how many partitions are currently being processed.
	partitionSem util.TrackingSemaphore

	qspc chan ShadowPartitionChanMsg

	// sem stores a semaphore controlling the number of jobs currently
	// being processed.  This lets us check whether there's capacity in the queue
	// prior to leasing items.
	sem util.TrackingSemaphore

	// roleLeaseIDs stores per-role lease IDs.
	roleLeaseIDs map[string]*ulid.ULID
	// roleLeaseLock ensures that there are no data races writing to
	// or reading from roleLeaseIDs in parallel.
	roleLeaseLock *sync.RWMutex

	// shardLeaseID stores the lease ID for the primary shard this queue is processing from.
	// all runners attempt to claim this lease on start up.
	shardLeaseID *ulid.ULID
	// shardLeaseLock ensures that there are no data races writing to
	// or reading from shardLeaseID in parallel.
	shardLeaseLock *sync.RWMutex

	// continues stores a map of all partition IDs to continues for a partition.
	// this lets us optimize running consecutive steps for a function, as a continuation, to a specific limit.
	continues        map[string]continuation
	continueCooldown map[string]time.Time

	// peekSizeCache stores ewma peek sizes for partitions.
	peekSizeCache *ccache.Cache[int64]

	// continuesLock protects the continues map.
	continuesLock *sync.Mutex

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

// Shard returns the leased primary shard. Callers in hot paths should
// cache the result locally to avoid repeated registry reads.
func (q *queueProcessor) Shard() QueueShard {
	return q.shards.Primary()
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

func (q *queueProcessor) Queue() Queue {
	return q
}

func (q *queueProcessor) Run(ctx context.Context, f RunFunc) error {
	// claimShardLease will block until a shard lease is obtained to process the primary shard.
	l := logger.StdlibLogger(ctx)
	if len(q.runMode.ShardGroup) != 0 {
		l.Info("Executor started in ShardGroup mode, attempting to claim a shard lease", "shard_group", q.runMode.ShardGroup)
		if err := q.claimShardLease(ctx); err != nil {
			return err
		}
	} else {
		l.Info("Executor started in assignedQueueShard Mode", "queue_shard", q.Shard().Name())
	}

	for _, role := range q.roles {
		go q.runRole(ctx, role)
	}

	wrappedF := q.wrapRunFuncWithLatency(f)

	// start execution and shadow scan concurrently
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return q.executionScan(ctx, wrappedF)
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

func (q *queueProcessor) capacity() int64 {
	return int64(q.numWorkers) - q.Semaphore().Count()
}

func (q *queueProcessor) partitionCapacity() int64 {
	return int64(q.numPartitionWorkers) - q.partitionSem.Count()
}
