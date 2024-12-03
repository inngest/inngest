package redis_state

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/enums"

	"github.com/google/uuid"

	"github.com/VividCortex/ewma"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

const (
	minWorkersFree = 5
)

var (
	latencyAvg ewma.MovingAverage
	latencySem *sync.Mutex

	startedAtKey = startedAtCtxKey{}
	sojournKey   = sojournCtxKey{}
	latencyKey   = latencyCtxKey{}

	errProcessNoCapacity   = fmt.Errorf("no capacity")
	errProcessStopIterator = fmt.Errorf("stop iterator")
)

func init() {
	latencyAvg = ewma.NewMovingAverage()
	latencySem = &sync.Mutex{}
}

// startedAtCtxKey is a context key which records when the queue item starts,
// available via context.
type startedAtCtxKey struct{}

// latencyCtxKey is a context key which records when the queue item starts,
// available via context.
type latencyCtxKey struct{}

// sojournCtxKey is a context key which records when the queue item starts,
// available via context.
type sojournCtxKey struct{}

func GetItemStart(ctx context.Context) (time.Time, bool) {
	t, ok := ctx.Value(startedAtKey).(time.Time)
	return t, ok
}

func GetItemSystemLatency(ctx context.Context) (time.Duration, bool) {
	t, ok := ctx.Value(latencyKey).(time.Duration)
	return t, ok
}

func GetItemConcurrencyLatency(ctx context.Context) (time.Duration, bool) {
	t, ok := ctx.Value(sojournKey).(time.Duration)
	return t, ok
}

// Enqueue adds an item to the queue to be processed at the given time.
// TODO: Lift this function and the queue interface to a higher level, so that it's disconnected from the
// concrete Redis implementation.
func (q *queue) Enqueue(ctx context.Context, item osqueue.Item, at time.Time, opts osqueue.EnqueueOpts) error {
	// propagate
	if item.Metadata == nil {
		item.Metadata = map[string]string{}
	}

	id := ""
	if item.JobID != nil {
		id = *item.JobID
	}

	if item.QueueName == nil {
		// Check if we have a kind mapping.
		if name, ok := q.queueKindMapping[item.Kind]; ok {
			item.QueueName = &name
		}
	}

	qi := osqueue.QueueItem{
		ID:          id,
		AtMS:        at.UnixMilli(),
		WorkspaceID: item.WorkspaceID,
		FunctionID:  item.Identifier.WorkflowID,
		Data:        item,
		QueueName:   item.QueueName,
		WallTimeMS:  at.UnixMilli(),
	}

	if item.QueueName == nil && qi.FunctionID == uuid.Nil {
		q.logger.Error().Interface("qi", qi).Msg("attempted to enqueue QueueItem without function ID or queueName override")
		return fmt.Errorf("queue name or function ID must be set")
	}

	// Use the queue item's score, ensuring we process older function runs first
	// (eg. before at)
	next := time.UnixMilli(qi.Score(q.clock.Now()))

	if factor := qi.Data.GetPriorityFactor(); factor != 0 {
		// Ensure we mutate the AtMS time by the given priority factor.
		qi.AtMS -= factor
	}

	shard := q.primaryQueueShard
	if q.shardSelector != nil {
		qn := qi.Data.QueueName
		if qn == nil {
			qn = qi.QueueName
		}
		selected, err := q.shardSelector(ctx, qi.Data.Identifier.AccountID, qn)
		if err != nil {
			q.logger.Error().Err(err).Interface("qi", qi).Msg("error selecting shard")
			return fmt.Errorf("could not select shard: %w", err)
		}

		shard = selected
	}

	metrics.IncrQueueItemStatusCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"status":      "enqueued",
			"kind":        item.Kind,
			"queue_shard": shard.Name,
		},
	})

	switch shard.Kind {
	case string(enums.QueueShardKindRedis):
		_, err := q.EnqueueItem(ctx, shard, qi, next, opts)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unknown shard kind: %s", shard.Kind)
	}
}

func (q *queue) Run(ctx context.Context, f osqueue.RunFunc) error {

	for i := int32(0); i < q.numWorkers; i++ {
		go q.worker(ctx, f)
	}

	if q.runMode.GuaranteedCapacity {
		go q.claimUnleasedGuaranteedCapacity(ctx, q.guaranteedCapacityScanTickTime, q.guaranteedCapacityLeaseTickTime)
	}

	if q.runMode.Sequential {
		go q.claimSequentialLease(ctx)
	}

	if q.runMode.Scavenger {
		go q.runScavenger(ctx)
	}

	go q.runInstrumentation(ctx)

	if !q.runMode.Partition && !q.runMode.Account {
		return fmt.Errorf("need to specify either partition, account, or both in queue run mode")
	}

	tick := q.clock.NewTicker(q.pollTick)

	q.logger.Debug().
		Str("poll", q.pollTick.String()).
		Msg("starting queue worker")

LOOP:
	for {
		select {
		case <-ctx.Done():
			// Kill signal
			tick.Stop()
			break LOOP
		case err := <-q.quit:
			// An inner function received an error which was deemed irrecoverable, so
			// we're quitting the queue.
			q.logger.Error().Err(err).Msg("quitting runner internally")
			tick.Stop()
			break LOOP
		case <-tick.Chan():
			if q.capacity() < minWorkersFree {
				// Wait until we have more workers free.  This stops us from
				// claiming a partition to work on a single job, ensuring we
				// have capacity to run at least MinWorkersFree concurrent
				// QueueItems.  This reduces latency of enqueued items when
				// there are lots of enqueued and available jobs.
				continue
			}

			if err := q.scan(ctx); err != nil {
				// On scan errors, halt the worker entirely.
				if !errors.Is(err, context.Canceled) {
					q.logger.Error().Err(err).Msg("error scanning partition pointers")
				}
				break LOOP
			}
		}
	}

	// Wait for all in-progress items to complete.
	q.logger.Info().Msg("queue waiting to quit")
	q.wg.Wait()

	return nil
}

// claimSequentialLease is a process which continually runs while listening to the queue,
// attempting to claim a lease on sequential processing.  Only one worker is allowed to
// work on partitions sequentially;  this reduces contention.
func (q *queue) claimSequentialLease(ctx context.Context) {
	// Workers with an allowlist can never claim sequential queues.
	if len(q.allowQueues) > 0 {
		return
	}

	// Attempt to claim the lease immediately.
	leaseID, err := q.ConfigLease(ctx, q.primaryQueueShard.RedisClient.kg.Sequential(), ConfigLeaseDuration, q.sequentialLease())
	if err != ErrConfigAlreadyLeased && err != nil {
		q.quit <- err
		return
	}

	q.seqLeaseLock.Lock()
	q.seqLeaseID = leaseID
	q.seqLeaseLock.Unlock()

	tick := q.clock.NewTicker(ConfigLeaseDuration / 3)
	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return
		case <-tick.Chan():
			leaseID, err := q.ConfigLease(ctx, q.primaryQueueShard.RedisClient.kg.Sequential(), ConfigLeaseDuration, q.sequentialLease())
			if err == ErrConfigAlreadyLeased {
				// This is expected; every time there is > 1 runner listening to the
				// queue there will be contention.
				q.seqLeaseLock.Lock()
				q.seqLeaseID = nil
				q.seqLeaseLock.Unlock()
				continue
			}
			if err != nil {
				q.logger.Error().Err(err).Msg("error claiming sequential lease")
				q.seqLeaseLock.Lock()
				q.seqLeaseID = nil
				q.seqLeaseLock.Unlock()
				continue
			}

			q.seqLeaseLock.Lock()
			if q.seqLeaseID == nil {
				// Only track this if we're creating a new lease, not if we're renewing
				// a lease.
				metrics.IncrQueueSequentialLeaseClaimsCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})
			}
			q.seqLeaseID = leaseID
			q.seqLeaseLock.Unlock()
		}
	}
}

func (q *queue) runScavenger(ctx context.Context) {
	// Attempt to claim the lease immediately.
	leaseID, err := q.ConfigLease(ctx, q.primaryQueueShard.RedisClient.kg.Scavenger(), ConfigLeaseDuration, q.scavengerLease())
	if err != ErrConfigAlreadyLeased && err != nil {
		q.quit <- err
		return
	}

	q.scavengerLeaseLock.Lock()
	q.scavengerLeaseID = leaseID // no-op if not leased
	q.scavengerLeaseLock.Unlock()

	tick := q.clock.NewTicker(ConfigLeaseDuration / 3)
	scavenge := q.clock.NewTicker(30 * time.Second)

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			scavenge.Stop()
			return
		case <-scavenge.Chan():
			// Scavenge the items
			if q.isScavenger() {
				count, err := q.Scavenge(ctx, ScavengePeekSize)
				if err != nil {
					q.logger.Error().Err(err).Msg("error scavenging")
				}
				if count > 0 {
					q.logger.Info().Int("len", count).Msg("scavenged lost jobs")
				}
			}
		case <-tick.Chan():
			// Attempt to re-lease the lock.
			leaseID, err := q.ConfigLease(ctx, q.primaryQueueShard.RedisClient.kg.Scavenger(), ConfigLeaseDuration, q.scavengerLease())
			if err == ErrConfigAlreadyLeased {
				// This is expected; every time there is > 1 runner listening to the
				// queue there will be contention.
				q.scavengerLeaseLock.Lock()
				q.scavengerLeaseID = nil
				q.scavengerLeaseLock.Unlock()
				continue
			}
			if err != nil {
				q.logger.Error().Err(err).Msg("error claiming scavenger lease")
				q.scavengerLeaseLock.Lock()
				q.scavengerLeaseID = nil
				q.scavengerLeaseLock.Unlock()
				continue
			}

			q.scavengerLeaseLock.Lock()
			if q.scavengerLeaseID == nil {
				// Only track this if we're creating a new lease, not if we're renewing
				// a lease.
				metrics.IncrQueueSequentialLeaseClaimsCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})
			}
			q.scavengerLeaseID = leaseID
			q.scavengerLeaseLock.Unlock()
		}
	}
}

func (q *queue) runInstrumentation(ctx context.Context) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Instrument"), redis_telemetry.ScopeQueue)

	leaseID, err := q.ConfigLease(ctx, q.primaryQueueShard.RedisClient.kg.Instrumentation(), ConfigLeaseMax, q.instrumentationLease())
	if err != ErrConfigAlreadyLeased && err != nil {
		q.quit <- err
		return
	}

	setLease := func(lease *ulid.ULID) {
		q.instrumentationLeaseLock.Lock()
		defer q.instrumentationLeaseLock.Unlock()
		q.instrumentationLeaseID = lease

		if lease != nil && q.instrumentationLeaseID == nil {
			metrics.IncrInstrumentationLeaseClaimsCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})
		}
	}

	setLease(leaseID)

	tick := q.clock.NewTicker(ConfigLeaseMax / 3)
	instr := q.clock.NewTicker(20 * time.Second)

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			instr.Stop()
			return
		case <-instr.Chan():
			if q.isInstrumentator() {
				if err := q.Instrument(ctx); err != nil {
					q.logger.Error().Err(err).Msg("error running instrumentation")
				}
			}
		case <-tick.Chan():
			metrics.GaugeWorkerQueueCapacity(ctx, int64(q.numWorkers), metrics.GaugeOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})

			leaseID, err := q.ConfigLease(ctx, q.primaryQueueShard.RedisClient.kg.Instrumentation(), ConfigLeaseMax, q.instrumentationLease())
			if err == ErrConfigAlreadyLeased {
				setLease(nil)
				continue
			}

			if err != nil {
				q.logger.Error().Err(err).Msg("error claiming instrumentation lease")
				setLease(nil)
				continue
			}

			setLease(leaseID)
		}
	}
}

// worker runs a blocking process that listens to items being pushed into the
// worker channel.  This allows us to process an individual item from a queue.
func (q *queue) worker(ctx context.Context, f osqueue.RunFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-q.quit:
			return
		case i := <-q.workers:
			// Create a new context which isn't cancelled by the parent, when quit.
			// XXX: When jobs can have their own cancellation signals, move this into
			// process itself.
			processCtx, cancel := context.WithCancel(context.Background())
			err := q.process(processCtx, i.P, i.I, i.G, f)
			q.sem.Release(1)
			metrics.WorkerQueueCapacityCounter(ctx, -1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})
			cancel()
			if err == nil {
				continue
			}

			// We handle the error individually within process, requeueing
			// the item into the queue.  Here, the worker can continue as
			// usual to process the next item.
			q.logger.Error().Err(err).Msg("error processing queue item")
		}
	}
}

func (q *queue) scanPartition(ctx context.Context, partitionKey string, peekLimit int64, peekUntil time.Time, guaranteedCapacity *GuaranteedCapacity, metricShardName string, accountId *uuid.UUID, reportPeekedPartitions *int64) error {
	// Peek 1s into the future to pull jobs off ahead of time, minimizing 0 latency

	partitions, err := durationWithTags(ctx, q.primaryQueueShard.Name, "partition_peek", q.clock.Now(), func(ctx context.Context) ([]*QueuePartition, error) {
		return q.partitionPeek(ctx, partitionKey, q.isSequential(), peekUntil, peekLimit, accountId)
	}, map[string]any{
		"is_global_partition_peek": fmt.Sprintf("%t", accountId == nil),
	})
	if err != nil {
		return err
	}

	if reportPeekedPartitions != nil {
		atomic.AddInt64(reportPeekedPartitions, int64(len(partitions)))
	}

	eg := errgroup.Group{}

	for _, ptr := range partitions {
		p := *ptr
		eg.Go(func() error {
			if q.capacity() == 0 {
				// no longer any available workers for partition, so we can skip
				// work
				metrics.IncrQueueScanNoCapacityCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"shard": metricShardName, "queue_shard": q.primaryQueueShard.Name}})
				return nil
			}
			if err := q.processPartition(ctx, &p, guaranteedCapacity, false); err != nil {
				if err == ErrPartitionNotFound || err == ErrPartitionGarbageCollected {
					// Another worker grabbed the partition, or the partition was deleted
					// during the scan by an another worker.
					// TODO: Increase internal metrics
					return nil
				}
				if !errors.Is(err, context.Canceled) {
					q.logger.Error().Err(err).Msg("error processing partition")
				}
				return err
			}

			metrics.IncrQueuePartitionProcessedCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"shard": metricShardName, "queue_shard": q.primaryQueueShard.Name},
			})
			return nil
		})
	}

	return eg.Wait()
}

func (q *queue) scan(ctx context.Context) error {
	if q.capacity() == 0 {
		return nil
	}

	// Store the shard that we processed, allowing us to eventually pass this
	// down to the job for stat tracking.
	var (
		guaranteedCapacity *GuaranteedCapacity
		metricShardName    = "<global>" // default global name for metrics in this function
	)

	peekUntil := q.clock.Now().Add(PartitionLookahead)

	// If this worker has leased accounts, those take priority 95% of the time.  There's a 5% chance that the
	// worker still works on the global queue.
	existingLeases := q.getAccountLeases()
	if len(existingLeases) > 0 {
		// Pick a random guaranteed capacity if we leased multiple
		i := rand.Intn(len(existingLeases))
		guaranteedCapacity = &existingLeases[i].GuaranteedCapacity

		metrics.IncrQueueScanCounter(ctx,
			metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"kind":        "guaranteed_capacity",
					"account_id":  guaranteedCapacity.AccountID.String(),
					"queue_shard": q.primaryQueueShard.Name,
				},
			},
		)

		// Backwards-compatible metrics names
		metricShardName = "<guaranteed-capacity>:" + guaranteedCapacity.Key()

		// When account is leased, process it
		partitionKey := q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(guaranteedCapacity.AccountID)
		var actualScannedPartitions int64

		err := q.scanPartition(ctx, partitionKey, PartitionPeekMax, peekUntil, guaranteedCapacity, metricShardName, &guaranteedCapacity.AccountID, &actualScannedPartitions)
		if err != nil {
			return err
		}

		metrics.IncrQueuePartitionScannedCounter(ctx,
			actualScannedPartitions,
			metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"kind":        "guaranteed_capacity",
					"queue_shard": q.primaryQueueShard.Name,
				},
			},
		)

		return nil

	}

	processAccount := false
	if q.runMode.Account && (!q.runMode.Partition || rand.Intn(100) <= q.runMode.AccountWeight) {
		processAccount = true
	}

	if processAccount {
		metrics.IncrQueueScanCounter(ctx,
			metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"kind":        "accounts",
					"queue_shard": q.primaryQueueShard.Name,
				},
			},
		)

		peekedAccounts, err := duration(ctx, q.primaryQueueShard.Name, "account_peek", q.clock.Now(), func(ctx context.Context) ([]uuid.UUID, error) {
			return q.accountPeek(ctx, q.isSequential(), peekUntil, AccountPeekMax)
		})
		if err != nil {
			return fmt.Errorf("could not peek accounts: %w", err)
		}

		if len(peekedAccounts) == 0 {
			return nil
		}

		// Reduce number of peeked partitions as we're processing multiple accounts in parallel
		// Note: This is not optimal as some accounts may have fewer partitions than others and
		// we're leaving capacity on the table. We'll need to find a better way to determine the
		// optimal peek size in this case.
		accountPartitionPeekMax := int64(math.Round(float64(PartitionPeekMax / int64(len(peekedAccounts)))))

		var actualScannedPartitions int64

		// Scan and process account partitions in parallel
		wg := sync.WaitGroup{}
		for _, account := range peekedAccounts {
			account := account

			wg.Add(1)
			go func(account uuid.UUID) {
				defer wg.Done()
				partitionKey := q.primaryQueueShard.RedisClient.kg.AccountPartitionIndex(account)

				if err := q.scanPartition(ctx, partitionKey, accountPartitionPeekMax, peekUntil, nil, metricShardName, &account, &actualScannedPartitions); err != nil {
					q.logger.Error().Err(err).Msg("error processing account partitions")
				}
			}(account)
		}

		wg.Wait()

		metrics.IncrQueuePartitionScannedCounter(ctx,
			actualScannedPartitions,
			metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"kind":        "accounts",
					"queue_shard": q.primaryQueueShard.Name,
				},
			},
		)

		return nil
	}

	metrics.IncrQueueScanCounter(ctx,
		metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"kind":        "partitions",
				"queue_shard": q.primaryQueueShard.Name,
			},
		},
	)

	// By default, use the global partition
	partitionKey := q.primaryQueueShard.RedisClient.kg.GlobalPartitionIndex()

	var actualScannedPartitions int64
	err := q.scanPartition(ctx, partitionKey, PartitionPeekMax, peekUntil, nil, metricShardName, nil, &actualScannedPartitions)
	if err != nil {
		return err
	}

	metrics.IncrQueuePartitionScannedCounter(ctx,
		actualScannedPartitions,
		metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"kind":        "partitions",
				"queue_shard": q.primaryQueueShard.Name,
			},
		},
	)

	return nil
}

// NOTE: Shard is only passed as a reference if the partition was peeked from
// a shard.  It exists for accounting and tracking purposes only, eg. to report shard metrics.
func (q *queue) processPartition(ctx context.Context, p *QueuePartition, guaranteedCapacity *GuaranteedCapacity, randomOffset bool) error {
	// Attempt to lease items.  This checks partition-level concurrency limits
	//
	// For optimization, because this is the only thread that can be leasing
	// jobs for this partition, we store the partition limit and current count
	// as a variable and iterate in the loop without loading keys from the state
	// store.
	//
	// There's no way to know when queue items finish processing;  we don't
	// store average runtimes for queue items (and we don't know because
	// items are dynamic generators).  This means that we have to delay
	// processing the partition by N seconds, meaning the latency is increased by
	// up to this period for scheduled items behind the concurrency limits.
	_, err := duration(ctx, q.primaryQueueShard.Name, "partition_lease", q.clock.Now(), func(ctx context.Context) (int, error) {
		_, capacity, err := q.PartitionLease(ctx, p, PartitionLeaseDuration)
		return capacity, err
	})
	if errors.Is(err, ErrPartitionConcurrencyLimit) {
		if p.FunctionID != nil {
			q.lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *p.FunctionID)
		}
		metrics.IncrQueuePartitionConcurrencyLimitCounter(ctx,
			metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"kind": "function", "queue_shard": q.primaryQueueShard.Name},
			},
		)
		return q.PartitionRequeue(ctx, q.primaryQueueShard, p, q.clock.Now().Truncate(time.Second).Add(PartitionConcurrencyLimitRequeueExtension), true)
	}
	if errors.Is(err, ErrAccountConcurrencyLimit) {
		// For backwards compatibility, we report on the function level as well
		if p.FunctionID != nil {
			q.lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *p.FunctionID)
		}
		q.lifecycles.OnAccountConcurrencyLimitReached(
			context.WithoutCancel(ctx),
			p.AccountID,
			p.EnvID,
		)
		metrics.IncrQueuePartitionConcurrencyLimitCounter(ctx,
			metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"kind": "account", "queue_shard": q.primaryQueueShard.Name},
			},
		)
		return q.PartitionRequeue(ctx, q.primaryQueueShard, p, q.clock.Now().Truncate(time.Second).Add(PartitionConcurrencyLimitRequeueExtension), true)
	}
	if errors.Is(err, ErrConcurrencyLimitCustomKey) {
		// For backwards compatibility, we report on the function level as well
		if p.FunctionID != nil {
			q.lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *p.FunctionID)
		}
		q.lifecycles.OnCustomKeyConcurrencyLimitReached(context.WithoutCancel(ctx), p.EvaluatedConcurrencyKey)
		metrics.IncrQueuePartitionConcurrencyLimitCounter(ctx,
			metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"kind": "custom", "queue_shard": q.primaryQueueShard.Name},
			},
		)
		return q.PartitionRequeue(ctx, q.primaryQueueShard, p, q.clock.Now().Truncate(time.Second).Add(PartitionConcurrencyLimitRequeueExtension), true)
	}
	if errors.Is(err, ErrPartitionAlreadyLeased) {
		metrics.IncrQueuePartitionLeaseContentionCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})
		return nil
	}
	if errors.Is(err, ErrPartitionNotFound) {
		// Another worker must have processed this partition between
		// this worker's peek and process.  Increase partition
		// contention metric and continue.  This is unsolvable.
		metrics.IncrPartitionGoneCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})
		return nil
	}
	if err != nil {
		return fmt.Errorf("error leasing partition: %w", err)
	}

	begin := q.clock.Now()
	defer func() {
		metrics.HistogramProcessPartitionDuration(ctx, q.clock.Since(begin).Milliseconds(), metrics.HistogramOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"queue_shard": q.primaryQueueShard.Name},
		})
	}()

	// Ensure that peek doesn't take longer than the partition lease, to
	// reduce contention.
	peekCtx, cancel := context.WithTimeout(ctx, PartitionLeaseDuration)
	defer cancel()

	// We need to round ourselves up to the nearest second, then add another second
	// to peek for jobs in the next <= 1999 milliseconds.
	//
	// There's a really subtle issue:  if two jobs contend for a pause and are scheduled
	// within 5ms of each other, we fetch them in order but we may process them out of
	// order, depending on how long it takes for the item to pass through the channel
	// to the worker, how long Redis takes to lease the item, etc.
	fetch := q.clock.Now().Truncate(time.Second).Add(PartitionLookahead)

	queue, err := duration(peekCtx, q.primaryQueueShard.Name, "peek", q.clock.Now(), func(ctx context.Context) ([]*osqueue.QueueItem, error) {
		peek := q.peekSize(ctx, p)
		// NOTE: would love to instrument this value to see it over time per function but
		// it's likely too high of a cardinality
		go metrics.HistogramQueuePeekEWMA(ctx, peek, metrics.HistogramOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})

		if randomOffset {
			return q.PeekRandom(peekCtx, p, fetch, peek)
		}

		return q.Peek(peekCtx, p, fetch, peek)
	})
	if err != nil {
		return err
	}
	metrics.HistogramQueuePeekSize(ctx, int64(len(queue)), metrics.HistogramOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})

	// Record the number of partitions we're leasing.
	metrics.IncrQueuePartitionLeasedCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})

	// parallel all queue names with internal mappings for now.
	// XXX: Allow parallel partitions for all functions except for fns opting into FIFO
	_, isSystemFn := q.queueKindMapping[p.Queue()]
	_, parallelFn := q.disableFifoForFunctions[p.Queue()]
	_, parallelAccount := q.disableFifoForAccounts[p.AccountID.String()]

	parallel := parallelFn || parallelAccount || isSystemFn

	iter := processor{
		partition:          p,
		items:              queue,
		guaranteedCapacity: guaranteedCapacity,
		queue:              q,
		denies:             newLeaseDenyList(),
		staticTime:         q.clock.Now(),
		parallel:           parallel,
	}

	if processErr := iter.iterate(ctx); processErr != nil {
		// Report the eerror.
		q.logger.Error().Err(processErr).Interface("partition", p).Msg("error iterating queue items")
		return processErr

	}

	if q.usePeekEWMA {
		if err := q.setPeekEWMA(ctx, p.FunctionID, int64(iter.ctrConcurrency+iter.ctrRateLimit)); err != nil {
			log.From(ctx).Warn().Err(err).Msg("error recording concurrency limit for EWMA")
		}
	}

	if iter.isRequeuable() && iter.isCustomKeyLimitOnly && !randomOffset && parallel {
		// We hit custom concurrency key issues.  Re-process this partition at a random offset, as long
		// as random offset is currently false (so we don't loop forever)

		// Note: we must requeue the partition to remove the lease.
		err := q.PartitionRequeue(ctx, q.primaryQueueShard, p, q.clock.Now().Truncate(time.Second).Add(PartitionConcurrencyLimitRequeueExtension), true)
		if err != nil {
			log.From(ctx).Warn().Err(err).Msg("error requeuieng partition for random peek")
		}

		return q.processPartition(ctx, p, guaranteedCapacity, true)
	}

	// If we've hit concurrency issues OR we've only hit rate limit issues, re-enqueue the partition
	// with a force:  ensure that we won't re-scan it until 2 seconds in the future.
	if iter.isRequeuable() {
		requeue := PartitionConcurrencyLimitRequeueExtension
		if iter.ctrConcurrency == 0 {
			// This has been throttled only.  Don't requeue so far ahead, otherwise we'll be waiting longer
			// than the minimum throttle.
			//
			// TODO: When we create throttle queues, requeue this appropriately depending on the throttle
			//       period.
			requeue = PartitionThrottleLimitRequeueExtension
		}

		// Requeue this partition as we hit concurrency limits.
		metrics.IncrQueuePartitionConcurrencyLimitCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})
		return q.PartitionRequeue(ctx, q.primaryQueueShard, p, q.clock.Now().Truncate(time.Second).Add(requeue), true)
	}

	// XXX: If we haven't been able to lease a single item, ensure we enqueue this
	// for a minimum of 5 seconds.

	// Requeue the partition, which reads the next unleased job or sets a time of
	// 30 seconds.  This is why we have to lease items above, else this may return an item that is
	// about to be leased and processed by the worker.
	_, err = duration(ctx, q.primaryQueueShard.Name, "partition_requeue", q.clock.Now(), func(ctx context.Context) (any, error) {
		err = q.PartitionRequeue(ctx, q.primaryQueueShard, p, q.clock.Now().Add(PartitionRequeueExtension), false)
		return nil, err
	})
	if err == ErrPartitionGarbageCollected {
		// Safe;  we're preventing this from wasting cycles in the future.
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

func (q *queue) process(ctx context.Context, p QueuePartition, qi osqueue.QueueItem, s *GuaranteedCapacity, f osqueue.RunFunc) error {
	var err error
	leaseID := qi.LeaseID

	// Allow the main runner to block until this work is done
	q.wg.Add(1)
	defer q.wg.Done()

	// Continually the lease while this job is being processed.
	extendLeaseTick := q.clock.NewTicker(QueueLeaseDuration / 2)
	defer extendLeaseTick.Stop()

	errCh := make(chan error)
	doneCh := make(chan struct{})

	// Continually extend lease in the background while we're working on this job
	go func() {
		for {
			select {
			case <-doneCh:
				return
			case <-extendLeaseTick.Chan():
				if ctx.Err() != nil {
					// Don't extend lease when the ctx is done.
					return
				}
				if leaseID == nil {
					log.From(ctx).Error().
						Str("account_id", p.AccountID.String()).
						Str("qi", qi.ID).
						Str("fn_id", qi.FunctionID.String()).
						Str("partition_id", p.ID).
						Msg("cannot extend lease since lease ID is nil")
					// Don't extend lease since one doesn't exist
					return
				}
				leaseID, err = q.ExtendLease(ctx, qi, *leaseID, QueueLeaseDuration)
				if err != nil && err != ErrQueueItemNotFound && errors.Unwrap(err) != context.Canceled {
					// XXX: Increase counter here.
					q.logger.Error().
						Err(err).
						Str("account_id", p.AccountID.String()).
						Str("qi", qi.ID).
						Str("fn_id", qi.FunctionID.String()).
						Str("partition_id", p.ID).
						Msg("error extending lease")
					errCh <- fmt.Errorf("error extending lease while processing: %w", err)
					return
				}
			}
		}
	}()

	// XXX: Add a max job time here, configurable.
	jobCtx, jobCancel := context.WithCancel(ctx)
	defer jobCancel()
	// Add the job ID to the queue context.  This allows any logic that handles the run function
	// to inspect job IDs, eg. for tracing or logging, without having to thread this down as
	// arguments.
	jobCtx = osqueue.WithJobID(jobCtx, qi.ID)
	// Same with the group ID, if it exists.
	if qi.Data.GroupID != "" {
		jobCtx = state.WithGroupID(jobCtx, qi.Data.GroupID)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Always retry this job.
				stack := debug.Stack()
				q.logger.Error().Err(fmt.Errorf("%v", r)).Str("stack", string(stack)).Msg("job panicked")
				errCh <- osqueue.AlwaysRetryError(fmt.Errorf("job panicked: %v", r))
			}
		}()

		// This job may be up to 1999 ms in the future, as explained in processPartition.
		// Just... wait until the job is available.
		delay := time.UnixMilli(qi.AtMS).Sub(q.clock.Now())

		if delay > 0 {
			<-q.clock.After(delay)
			q.logger.Trace().
				Int64("at", qi.AtMS).
				Int64("ms", delay.Milliseconds()).
				Msg("delaying job in memory")
		}

		n := q.clock.Now()

		// Track the sojourn (concurrency) latency.
		var sojourn time.Duration
		if qi.EarliestPeekTime > 0 {
			sojourn = n.Sub(time.UnixMilli(qi.EarliestPeekTime))
		}
		jobCtx = context.WithValue(jobCtx, sojournKey, sojourn)

		// Track the latency on average globally.  Do this in a goroutine so that it doesn't
		// at all delay the job during concurrenty locking contention.
		if qi.WallTimeMS == 0 {
			qi.WallTimeMS = qi.AtMS // backcompat while WallTimeMS isn't valid.
		}
		latency := n.Sub(time.UnixMilli(qi.WallTimeMS)) - sojourn
		jobCtx = context.WithValue(jobCtx, latencyKey, latency)

		// store started at and latency in ctx
		jobCtx = context.WithValue(jobCtx, startedAtKey, n)

		go func() {
			// Update the ewma
			latencySem.Lock()
			latencyAvg.Add(float64(latency))
			metrics.GaugeQueueItemLatencyEWMA(ctx, int64(latencyAvg.Value()/1e6), metrics.GaugeOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"kind": qi.Data.Kind, "queue_shard": q.primaryQueueShard.Name},
			})
			latencySem.Unlock()

			// Set the metrics historgram and gauge, which reports the ewma value.
			metrics.HistogramQueueItemLatency(ctx, latency.Milliseconds(), metrics.HistogramOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"kind": qi.Data.Kind, "queue_shard": q.primaryQueueShard.Name},
			})
		}()

		metrics.IncrQueueItemStatusCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "started", "queue_shard": q.primaryQueueShard.Name},
		})

		runInfo := osqueue.RunInfo{
			Latency:        latency,
			SojournDelay:   sojourn,
			Priority:       q.ppf(ctx, p),
			QueueShardName: q.primaryQueueShard.Name,
		}
		if s != nil {
			runInfo.GuaranteedCapacityKey = s.Name
			if runInfo.GuaranteedCapacityKey == "" {
				runInfo.GuaranteedCapacityKey = s.Key()
			}
		}

		// Call the run func.
		err := f(jobCtx, runInfo, qi.Data)
		extendLeaseTick.Stop()
		if err != nil {
			metrics.IncrQueueItemStatusCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"status": "errored", "queue_shard": q.primaryQueueShard.Name},
			})
			errCh <- err
			return
		}
		metrics.IncrQueueItemStatusCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "completed", "queue_shard": q.primaryQueueShard.Name},
		})

		// Closing this channel prevents the goroutine which extends lease from leaking,
		// and dequeues the job
		close(doneCh)
	}()

	select {
	case err := <-errCh:
		// Job errored or extending lease errored.  Cancel the job ASAP.
		jobCancel()

		if osqueue.ShouldRetry(err, qi.Data.Attempt, qi.Data.GetMaxAttempts()) {
			at := q.backoffFunc(qi.Data.Attempt)

			// Attempt to find any RetryAtSpecifier in the error tree.
			unwrapped := err
			for unwrapped != nil {
				// If the error contains a NextRetryAt method, use that to indicate
				// when we should retry.
				if specifier, ok := unwrapped.(osqueue.RetryAtSpecifier); ok {
					next := specifier.NextRetryAt()
					if next != nil {
						at = *next
					}
					break
				}
				unwrapped = errors.Unwrap(unwrapped)
			}

			if !osqueue.IsAlwaysRetryable(err) {
				qi.Data.Attempt += 1
			}

			qi.AtMS = at.UnixMilli()
			if err := q.Requeue(context.WithoutCancel(ctx), q.primaryQueueShard, qi, at); err != nil {
				q.logger.Error().Err(err).Interface("item", qi).Msg("error requeuing job")
				return err
			}
			if _, ok := err.(osqueue.QuitError); ok {
				q.quit <- err
				return err
			}
			return nil
		}

		// Dequeue this entirely, as this permanently failed.
		// XXX: Increase permanently failed counter here.
		if err := q.Dequeue(context.WithoutCancel(ctx), q.primaryQueueShard, qi); err != nil {
			if err == ErrQueueItemNotFound {
				// Safe. The executor may have dequeued.
				return nil
			}
			return err
		}

		if _, ok := err.(osqueue.QuitError); ok {
			q.logger.Warn().Err(err).Msg("received queue quit error")
			q.quit <- err
			return err
		}

	case <-doneCh:
		if err := q.Dequeue(context.WithoutCancel(ctx), q.primaryQueueShard, qi); err != nil {
			if err == ErrQueueItemNotFound {
				// Safe. The executor may have dequeued.
				return nil
			}
			return err
		}
	}

	return nil
}

// sequentialLease is a helper method for concurrently reading the sequential
// lease ID.
func (q *queue) sequentialLease() *ulid.ULID {
	q.seqLeaseLock.RLock()
	defer q.seqLeaseLock.RUnlock()
	if q.seqLeaseID == nil {
		return nil
	}
	copied := *q.seqLeaseID
	return &copied
}

// instrumentationLease is a helper method for concurrently reading the
// instrumentation lease ID.
func (q *queue) instrumentationLease() *ulid.ULID {
	q.instrumentationLeaseLock.RLock()
	defer q.instrumentationLeaseLock.RUnlock()
	if q.instrumentationLeaseID == nil {
		return nil
	}
	copied := *q.instrumentationLeaseID
	return &copied
}

// scavengerLease is a helper method for concurrently reading the sequential
// lease ID.
func (q *queue) scavengerLease() *ulid.ULID {
	q.scavengerLeaseLock.RLock()
	defer q.scavengerLeaseLock.RUnlock()
	if q.scavengerLeaseID == nil {
		return nil
	}
	copied := *q.scavengerLeaseID
	return &copied
}

func (q *queue) capacity() int64 {
	return int64(q.numWorkers) - atomic.LoadInt64(&q.sem.counter)
}

// peekSize returns the number of items to peek for the queue based on a couple of factors
// 1. EWMA of concurrency limit hits
// 2. configured min, max of peek size range
// 3. worker capacity
func (q *queue) peekSize(ctx context.Context, p *QueuePartition) int64 {
	if q.usePeekEWMA {
		return q.ewmaPeekSize(ctx, p)
	}
	return q.peekSizeRandom(ctx, p)
}

func (q *queue) peekSizeRandom(_ context.Context, _ *QueuePartition) int64 {
	// set ranges
	pmin := q.peekMin
	if pmin == 0 {
		pmin = q.peekMin
	}
	pmax := q.peekMax
	if pmax == 0 {
		pmax = q.peekMax
	}

	// Take a random amount between our range.
	size := int64(rand.Intn(int(pmax-pmin))) + pmin
	// Limit to capacity
	cap := q.capacity()
	if size > cap {
		size = cap
	}
	return size
}

//nolint:golint,unused // this code remains to be enabled on demand
func (q *queue) ewmaPeekSize(ctx context.Context, p *QueuePartition) int64 {
	if p.FunctionID == nil {
		return q.peekMin
	}

	// retrieve the EWMA value
	ewma, err := q.peekEWMA(ctx, *p.FunctionID)
	if err != nil {
		// return the minimum if there's an error
		return q.peekMin
	}

	// set multiplier
	multiplier := q.peekCurrMultiplier
	if multiplier == 0 {
		multiplier = QueuePeekCurrMultiplier
	}

	// set ranges
	pmin := q.peekMin
	if pmin == 0 {
		pmin = DefaultQueuePeekMin
	}
	pmax := q.peekMax
	if pmax == 0 {
		pmax = DefaultQueuePeekMax
	}

	// calculate size with EWMA and multiplier
	size := ewma * multiplier
	switch {
	case size < pmin:
		size = pmin
	case size > pmax:
		size = pmax
	}

	dur := time.Hour * 24
	qsize, _ := q.partitionSize(ctx, p.zsetKey(q.primaryQueueShard.RedisClient.kg), q.clock.Now().Add(dur))
	if qsize > size {
		size = qsize
	}

	// add 10% expecting for some workflow that will finish in the mean time
	cap := int64(float64(q.capacity()) * 1.1)
	if size > cap {
		size = cap
	}

	return size
}

func (q *queue) isSequential() bool {
	l := q.sequentialLease()
	if l == nil {
		return false
	}
	return ulid.Time(l.Time()).After(q.clock.Now())
}

func (q *queue) isScavenger() bool {
	l := q.scavengerLease()
	if l == nil {
		return false
	}
	return ulid.Time(l.Time()).After(q.clock.Now())
}

func (q *queue) isInstrumentator() bool {
	l := q.instrumentationLease()
	if l == nil {
		return false
	}
	return ulid.Time(l.Time()).After(q.clock.Now())
}

// duration is a helper function to record durations of queue operations.
func duration[T any](ctx context.Context, queueShardName string, op string, start time.Time, f func(ctx context.Context) (T, error)) (T, error) {
	return durationWithTags(ctx, queueShardName, op, start, f, nil)
}

// durationWithTags is a helper function to record durations of queue operations.
func durationWithTags[T any](ctx context.Context, queueShardName string, op string, start time.Time, f func(ctx context.Context) (T, error), tags map[string]any) (T, error) {
	if start.IsZero() {
		start = time.Now()
	}

	finalTags := map[string]any{
		"operation":   op,
		"queue_shard": queueShardName,
	}
	for k, v := range tags {
		finalTags[k] = v
	}

	res, err := f(ctx)
	metrics.HistogramQueueOperationDuration(
		ctx,
		time.Since(start).Milliseconds(),
		metrics.HistogramOpt{
			PkgName: pkgName,
			Tags:    finalTags,
		},
	)
	return res, err
}

// trackingSemaphore returns a semaphore that tracks closely - but not atomically -
// the total number of items in the semaphore.  This is best effort, and is loosely
// accurate to reduce further contention.
//
// This is only used as an indicator as to whether to scan.
type trackingSemaphore struct {
	*semaphore.Weighted
	counter int64
}

func (t *trackingSemaphore) TryAcquire(n int64) bool {
	if !t.Weighted.TryAcquire(n) {
		return false
	}
	atomic.AddInt64(&t.counter, n)
	return true
}

func (t *trackingSemaphore) Acquire(ctx context.Context, n int64) error {
	if err := t.Weighted.Acquire(ctx, n); err != nil {
		return err
	}
	atomic.AddInt64(&t.counter, n)
	return nil
}

func (t *trackingSemaphore) Release(n int64) {
	t.Weighted.Release(n)
	atomic.AddInt64(&t.counter, -n)
}

type processor struct {
	partition          *QueuePartition
	items              []*osqueue.QueueItem
	guaranteedCapacity *GuaranteedCapacity

	// queue is the queue that owns this processor.
	queue *queue

	// denies records a denylist as keys hit concurrency and throttling limits.
	// this lets us prevent lease attempts for consecutive keys, as soon as the first
	// key is denied.
	denies *leaseDenies

	// error returned when processing
	err error

	// staticTime is used as the processing time for all items in the queue.
	// We process queue items sequentially, and time progresses linearly as each
	// queue item is processed.  We want to use a static time to prevent out-of-order
	// processing with regards to things like rate limiting;  if we use time.Now(),
	// queue items later in the array may be processed before queue items earlier in
	// the array depending on eg. a rate limit becoming available half way through
	// iteration.
	staticTime time.Time

	// parallel indicates whether the partition's jobs can be processed in parallel.
	// parallel processing breaks best effort fifo but increases throughput.
	parallel bool

	// These flags are used to handle partition rqeueueing.
	ctrSuccess     int32
	ctrConcurrency int32
	ctrRateLimit   int32

	// isCustomKeyLimitOnly records whether we ONLY hit custom concurrency key limits.
	// This lets us know whether to peek from a random offset if we have FIFO disabled
	// to attempt to find other possible functions outside of the key(s) with issues.
	isCustomKeyLimitOnly bool
}

func (p *processor) iterate(ctx context.Context) error {
	var err error

	// set flag to true to begin with
	p.isCustomKeyLimitOnly = true

	eg := errgroup.Group{}
	for _, i := range p.items {
		if i == nil {
			// THIS SHOULD NEVER HAPPEN. Skip gracefully and log error
			log.From(ctx).Error().Msg("nil queue item in partition")
			continue
		}

		if p.parallel {
			item := *i
			eg.Go(func() error {
				return p.process(ctx, &item)
			})
			continue
		}

		// non-parallel (sequential fifo) processing.
		if err = p.process(ctx, i); err != nil {
			// always break on the first error;  if processing returns an error we
			// always assume that we stop iterating.
			//
			// we return errors when:
			// * there's no capacity (so dont continue, because FIFO)
			// * we hit fn concurrency limits (so don't continue, because FIFO too)
			// * some other error, which means something went wrong.
			break
		}
	}

	if p.parallel {
		// normalize errors from parallel
		err = eg.Wait()
	}

	if errors.Is(err, errProcessStopIterator) {
		// This is safe;  it's stopping safely but isn't an error.
		return nil
	}
	if errors.Is(err, errProcessNoCapacity) {
		// This is safe;  it's stopping safely but isn't an error.
		return nil
	}

	// someting went wrong.  report the error.
	return err
}

func (p *processor) process(ctx context.Context, item *osqueue.QueueItem) error {
	// TODO: Create an in-memory mapping of rate limit keys that have been hit,
	//       and don't bother to process if the queue item has a limited key.  This
	//       lessens work done in the queue, as we can `continue` immediately.
	if item.IsLeased(p.queue.clock.Now()) {
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "lease_contention", "queue_shard": p.queue.primaryQueueShard.Name},
		})
		return nil
	}

	// Check if there's capacity from our local workers atomically prior to leasing our items.
	if !p.queue.sem.TryAcquire(1) {
		metrics.IncrQueuePartitionProcessNoCapacityCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": p.queue.primaryQueueShard.Name}})
		// Break the entire loop to prevent out of order work.
		return errProcessNoCapacity
	}

	metrics.WorkerQueueCapacityCounter(ctx, 1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": p.queue.primaryQueueShard.Name}})

	// Attempt to lease this item before passing this to a worker.  We have to do this
	// synchronously as we need to lease prior to requeueing the partition pointer. If
	// we don't do this here, the workers may not lease the items before calling Peek
	// to re-enqeueu the pointer, which then increases contention - as we requeue a
	// pointer too early.
	//
	// This is safe:  only one process runs scan(), and we guard the total number of
	// available workers with the above semaphore.
	leaseID, err := duration(ctx, p.queue.primaryQueueShard.Name, "lease", p.queue.clock.Now(), func(ctx context.Context) (*ulid.ULID, error) {
		return p.queue.Lease(ctx, *item, QueueLeaseDuration, p.staticTime, p.denies)
	})

	// NOTE: If this loop ends in an error, we must _always_ release an item from the
	// semaphore to free capacity.  This will happen automatically when the worker
	// finishes processing a queue item on success.
	if err != nil {
		// Continue on and handle the error below.
		p.queue.sem.Release(1)
		metrics.WorkerQueueCapacityCounter(ctx, -1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": p.queue.primaryQueueShard.Name}})
	}

	// Check the sojourn delay for this item in the queue. Tracking system latency vs
	// sojourn latency from concurrency is important.
	//
	// Firstly, we check:  does the job store the first peek time?  If so, the
	// delta between now and that time is the sojourn latency.  If not, this is either
	// one of two cases:
	//   - This is a new job in the queue, and we're peeking it for the first time.
	//     Sojourn latency is 0.  Easy.
	//   - We've peeked the queue since adding the job.  At this point, the only
	//     conclusion is that the job wasn't peeked because of concurrency/capacity
	//     issues, so the delta between now - job added is sojourn latency.
	//
	// NOTE: You might see that we use tracking semaphores and the worker itself has
	// a maximum capacity.  We must ALWAYS peek the available capacity in our worker
	// via the above Peek() call so that worker capacity doesn't prevent us from accessing
	// all jobs in a peek.  This would break sojourn latency:  it only works if we know
	// we're quitting early because of concurrency issues in a user's function, NOT because
	// of capacity issues in our system.
	//
	// Anyway, here we set the first peek item to the item's start time if there was a
	// peek since the job was added.
	if p.partition.Last > 0 && p.partition.Last > item.AtMS {
		// Fudge the earliest peek time because we know this wasn't peeked and so
		// the peek time wasn't set;  but, as we were still processing jobs after
		// the job was added this item was concurrency-limited.
		item.EarliestPeekTime = item.AtMS
	}

	// We may return a keyError, which masks the actual error underneath.  If so,
	// grab the cause.
	cause := err
	var key keyError
	if errors.As(err, &key) {
		cause = key.cause
	}

	switch cause {
	case ErrQueueItemThrottled:
		p.isCustomKeyLimitOnly = false
		// Here we denylist each throttled key that's been limited here, then ignore
		// any other jobs from being leased as we continue to iterate through the loop.
		// This maintains FIFO ordering amongst all custom concurrency keys.
		p.denies.addThrottled(err)

		p.ctrRateLimit++
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "throttled", "queue_shard": p.queue.primaryQueueShard.Name},
		})
		return nil
	case ErrPartitionConcurrencyLimit, ErrAccountConcurrencyLimit, ErrSystemConcurrencyLimit:
		p.isCustomKeyLimitOnly = false

		p.ctrConcurrency++
		// Since the queue is at capacity on a fn or account level, no
		// more jobs in this loop should be worked on - so break.
		//
		// Even if we have capacity for the next job in the loop we do NOT
		// want to claim the job, as this breaks ordering guarantees.  The
		// only safe thing to do when we hit a function or account level
		// concurrency key.
		var status string
		switch cause {
		case ErrSystemConcurrencyLimit:
			status = "system_concurrency_limit"
		case ErrPartitionConcurrencyLimit:
			status = "partition_concurrency_limit"
			if p.partition.FunctionID != nil {
				p.queue.lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *p.partition.FunctionID)
			}
		case ErrAccountConcurrencyLimit:
			status = "account_concurrency_limit"
			// For backwards compatibility, we report on the function level as well
			if p.partition.FunctionID != nil {
				p.queue.lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *p.partition.FunctionID)
			}

			p.queue.lifecycles.OnAccountConcurrencyLimitReached(
				context.WithoutCancel(ctx),
				p.partition.AccountID,
				p.partition.EnvID,
			)
		}

		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": status, "queue_shard": p.queue.primaryQueueShard.Name},
		})

		return fmt.Errorf("concurrency hit: %w", errProcessStopIterator)
	case ErrConcurrencyLimitCustomKey:
		p.ctrConcurrency++

		// Custom concurrency keys are different.  Each job may have a different key,
		// so we cannot break the loop in case the next job has a different key and
		// has capacity.
		//
		// Here we denylist each concurrency key that's been limited here, then ignore
		// any other jobs from being leased as we continue to iterate through the loop.
		p.denies.addConcurrency(err)

		// For backwards compatibility, we report on the function level as well
		if p.partition.FunctionID != nil {
			p.queue.lifecycles.OnFnConcurrencyLimitReached(context.WithoutCancel(ctx), *p.partition.FunctionID)
		}

		p.queue.lifecycles.OnCustomKeyConcurrencyLimitReached(context.WithoutCancel(ctx), p.partition.EvaluatedConcurrencyKey)

		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "custom_key_concurrency_limit", "queue_shard": p.queue.primaryQueueShard.Name},
		})
		return nil
	case ErrQueueItemNotFound:
		// This is an okay error.  Move to the next job item.
		p.ctrSuccess++ // count as a success for stats purposes.
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "success", "queue_shard": p.queue.primaryQueueShard.Name},
		})
		return nil
	case ErrQueueItemAlreadyLeased:
		// This is an okay error.  Move to the next job item.
		p.ctrSuccess++ // count as a success for stats purposes.
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "success", "queue_shard": p.queue.primaryQueueShard.Name},
		})
		return nil
	}

	// Handle other errors.
	if err != nil {
		p.err = fmt.Errorf("error leasing in process: %w", err)
		metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"status": "error", "queue_shard": p.queue.primaryQueueShard.Name},
		})
		return p.err
	}

	// Assign the lease ID and pass this to be handled by the available worker.
	// There should always be capacity on this queue as we track capacity via
	// a semaphore.
	item.LeaseID = leaseID

	// increase success counter.
	p.ctrSuccess++
	metrics.IncrQueueItemProcessedCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags:    map[string]any{"status": "success", "queue_shard": p.queue.primaryQueueShard.Name},
	})
	p.queue.workers <- processItem{P: *p.partition, I: *item, G: p.guaranteedCapacity}
	return nil
}

func (p *processor) isRequeuable() bool {
	// if we have concurrency OR we hit rate limiting/throttling.
	return p.ctrConcurrency > 0 || (p.ctrRateLimit > 0 && p.ctrConcurrency == 0 && p.ctrSuccess == 0)
}
