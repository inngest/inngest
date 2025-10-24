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

	"github.com/VividCortex/ewma"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
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
		item.Metadata = map[string]any{}
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

	l := q.log.With(
		"item", qi,
		"account_id", item.Identifier.AccountID,
		"env_id", item.WorkspaceID,
		"app_id", item.Identifier.AppID,
		"fn_id", item.Identifier.WorkflowID,
		"queue_shard", q.primaryQueueShard.Name,
	)

	if item.QueueName == nil && qi.FunctionID == uuid.Nil {
		err := fmt.Errorf("queue name or function ID must be set")
		l.ReportError(err, "attempted to enqueue QueueItem without function ID or queueName override")
		return err
	}

	// Pass optional idempotency period to queue item
	if opts.IdempotencyPeriod != nil {
		qi.IdempotencyPeriod = opts.IdempotencyPeriod
	}

	// Use the queue item's score, ensuring we process older function runs first
	// (eg. before at)
	next := time.UnixMilli(qi.Score(q.clock.Now()))

	if factor := qi.Data.GetPriorityFactor(); factor != 0 {
		// Ensure we mutate the AtMS time by the given priority factor.
		qi.AtMS -= factor
	}

	shard, err := q.selectShard(ctx, opts.ForceQueueShardName, qi)
	if err != nil {
		return err
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

		// XXX: If we've enqueued a user queue item (sleep, retry, step, etc.) and it's in the future,
		// we want to ensure that we schedule a rebalance job which takes the queue item and places it
		// at the correct score based off of the item's run ID when it becomes available.
		//
		// Without this, step.sleep or retries for a very old workflow may still lag behind steps from
		// later workflows when scheduled in the future.  This can, worst case, cause never-ending runs.
		if !q.enableJobPromotion || !qi.RequiresPromotionJob(q.clock.Now()) {
			// scheule a rebalance job automatically.
			return nil
		}

		// This is to prevent infinite recursion in case RequiresPromotion is accidentally refactored
		// to include the below job kind.
		if qi.Data.Kind == osqueue.KindJobPromote {
			return nil
		}

		// This is the fudge job.  What a name!
		//
		// If we're processing a user function and the sleep duration is in the future,
		// enqueue a sleep scavenge system queue item that will Requeue the original sleep queue item.
		// We do this to fudge the original queue item at the exact time, the run was scheduled for to ensure
		// sleeps for existing function runs are picked up earlier than items for later function runs.
		promoteAt := time.UnixMilli(qi.AtMS).Add(consts.FutureAtLimit * -1)
		promoteJobID := fmt.Sprintf("promote-%s", qi.ID)
		promoteQueueName := fmt.Sprintf("job-promote:%s", qi.FunctionID)
		err = q.Enqueue(ctx, osqueue.Item{
			JobID:          &promoteJobID,
			WorkspaceID:    qi.Data.WorkspaceID,
			QueueName:      &promoteQueueName,
			Kind:           osqueue.KindJobPromote,
			Identifier:     qi.Data.Identifier,
			PriorityFactor: qi.Data.PriorityFactor,
			Attempt:        0,
			Payload: osqueue.PayloadJobPromote{
				PromoteJobID: qi.ID,
				ScheduledAt:  qi.AtMS,
			},
		}, promoteAt, osqueue.EnqueueOpts{})
		if err != nil && err != ErrQueueItemExists {
			// This is best effort, and shouldn't fail the OG enqueue.
			l.ReportError(err, "error scheduling promotion job")
		}
		return nil
	default:
		return fmt.Errorf("unknown shard kind: %s", shard.Kind)
	}
}

func (q *queue) selectShard(ctx context.Context, shardName string, qi osqueue.QueueItem) (QueueShard, error) {
	shard := q.primaryQueueShard
	switch {
	// If the caller wants us to enqueue the job to a specific queue shard, use that.
	case shardName != "":
		foundShard, ok := q.queueShardClients[shardName]
		if !ok {
			return shard, fmt.Errorf("tried to force invalid queue shard %q", shardName)
		}

		shard = foundShard
	// Otherwise, invoke the shard selector, if configured.
	case q.shardSelector != nil:
		// QueueName should be consistently specified on both levels. This safeguard ensures
		// we'll check for both places, just in case.
		qn := qi.Data.QueueName
		if qn == nil {
			qn = qi.QueueName
		}

		selected, err := q.shardSelector(ctx, qi.Data.Identifier.AccountID, qn)
		if err != nil {
			q.log.Error("error selecting shard", "error", err, "item", qi)
			return shard, fmt.Errorf("could not select shard: %w", err)
		}

		shard = selected
	}
	return shard, nil
}

func (q *queue) Run(ctx context.Context, f osqueue.RunFunc) error {
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

func (q *queue) executionScan(ctx context.Context, f osqueue.RunFunc) error {
	l := q.log.With(
		"queue_shard", q.primaryQueueShard.Name,
	)

	for i := int32(0); i < q.numWorkers; i++ {
		go q.worker(ctx, f)
	}

	if !q.runMode.Partition && !q.runMode.Account {
		return fmt.Errorf("need to specify either partition, account, or both in queue run mode")
	}

	tick := q.clock.NewTicker(q.pollTick)
	l.Debug("starting queue worker", "poll", q.pollTick.String())

	backoff := time.Millisecond * 250

	var err error
LOOP:
	for {
		select {
		case <-ctx.Done():
			// Kill signal
			tick.Stop()
			break LOOP
		case err = <-q.quit:
			// An inner function received an error which was deemed irrecoverable, so
			// we're quitting the queue.
			q.log.ReportError(err, "quitting runner internally")
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

			if err = q.scan(ctx); err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					q.log.Warn("deadline exceeded scanning partition pointers")
					<-time.After(backoff)

					// Backoff doubles up to 3 seconds.
					backoff = time.Duration(math.Min(float64(backoff*2), float64(time.Second*5)))
					continue
				}

				// On scan errors, halt the worker entirely.
				if !errors.Is(err, context.Canceled) {
					q.log.ReportError(err, "error scanning partition pointers")
				}
				break LOOP
			}

			backoff = time.Millisecond * 250
		}
	}

	// Wait for all in-progress items to complete.
	q.log.Info("queue waiting to quit")
	q.wg.Wait()

	return err
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
				q.log.Error("error claiming sequential lease", "error", err)
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
					q.log.Error("error scavenging", "error", err)
				}
				if count > 0 {
					q.log.Info("scavenged lost jobs", "len", count)
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
				q.log.Error("error claiming scavenger lease", "error", err)
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

func (q *queue) runActiveChecker(ctx context.Context) {
	// Attempt to claim the lease immediately.
	leaseID, err := q.ConfigLease(ctx, q.primaryQueueShard.RedisClient.kg.ActiveChecker(), ConfigLeaseDuration, q.activeCheckerLease())
	if err != ErrConfigAlreadyLeased && err != nil {
		q.quit <- err
		return
	}

	q.activeCheckerLeaseLock.Lock()
	q.activeCheckerLeaseID = leaseID // no-op if not leased
	q.activeCheckerLeaseLock.Unlock()

	tick := q.clock.NewTicker(ConfigLeaseDuration / 3)
	checkTick := q.clock.NewTicker(q.activeCheckTick)

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			checkTick.Stop()
			return
		case <-checkTick.Chan():
			// Active check backlogs
			if q.isActiveChecker() {
				count, err := q.ActiveCheck(ctx)
				if err != nil {
					q.log.Error("error checking active jobs", "error", err)
				}
				if count > 0 {
					q.log.Trace("checked active jobs", "len", count)
				}
			}
		case <-tick.Chan():
			// Attempt to re-lease the lock.
			leaseID, err := q.ConfigLease(ctx, q.primaryQueueShard.RedisClient.kg.ActiveChecker(), ConfigLeaseDuration, q.activeCheckerLease())
			if err == ErrConfigAlreadyLeased {
				// This is expected; every time there is > 1 runner listening to the
				// queue there will be contention.
				q.activeCheckerLeaseLock.Lock()
				q.activeCheckerLeaseID = nil
				q.activeCheckerLeaseLock.Unlock()
				continue
			}
			if err != nil {
				q.log.Error("error claiming active checker lease", "error", err)
				q.activeCheckerLeaseLock.Lock()
				q.activeCheckerLeaseID = nil
				q.activeCheckerLeaseLock.Unlock()
				continue
			}

			q.activeCheckerLeaseLock.Lock()
			q.activeCheckerLeaseID = leaseID
			q.activeCheckerLeaseLock.Unlock()
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
	instr := q.clock.NewTicker(q.instrumentInterval)

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			instr.Stop()
			return
		case <-instr.Chan():
			if q.isInstrumentator() {
				if err := q.Instrument(ctx); err != nil {
					q.log.Error("error running instrumentation", "error", err)
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
				q.log.Error("error claiming instrumentation lease", "error", err)
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

		case i := <-q.workers:
			// Create a new context which isn't cancelled by the parent, when quit.
			// XXX: When jobs can have their own cancellation signals, move this into
			// process itself.
			processCtx, cancel := context.WithCancel(context.Background())
			err := q.process(processCtx, i, f)
			q.sem.Release(1)
			metrics.WorkerQueueCapacityCounter(ctx, -1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})
			cancel()
			if err == nil {
				continue
			}

			// We handle the error individually within process, requeueing
			// the item into the queue.  Here, the worker can continue as
			// usual to process the next item.
			q.log.Error("error processing queue item", "error", err, "item", i)
		}
	}
}

func (q *queue) scanPartition(ctx context.Context, partitionKey string, peekLimit int64, peekUntil time.Time, metricShardName string, accountId *uuid.UUID, reportPeekedPartitions *int64) error {
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

	if len(partitions) > 0 {
		q.log.Trace("processing partitions",
			"partition_key", partitionKey,
			"peek_until", peekUntil.Format(time.StampMilli),
			"partition", len(partitions),
		)
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
			if err := q.processPartition(ctx, &p, 0, false); err != nil {
				if err == ErrPartitionNotFound || err == ErrPartitionGarbageCollected {
					// Another worker grabbed the partition, or the partition was deleted
					// during the scan by an another worker.
					// TODO: Increase internal metrics
					return nil
				}
				if !errors.Is(err, context.Canceled) {
					q.log.Error("error processing partition", "error", err)
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

	// If there are continuations, process those immediately.
	if err := q.scanContinuations(ctx); err != nil {
		return fmt.Errorf("error scanning continuations: %w", err)
	}

	// Store the shard that we processed, allowing us to eventually pass this
	// down to the job for stat tracking.
	metricShardName := "<global>" // default global name for metrics in this function

	peekUntil := q.clock.Now().Add(PartitionLookahead)

	processAccount := false
	if q.runMode.Account && (!q.runMode.Partition || rand.Intn(100) <= q.runMode.AccountWeight) {
		processAccount = true
	}

	if len(q.runMode.ExclusiveAccounts) > 0 {
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

		var peekedAccounts []uuid.UUID
		if len(q.runMode.ExclusiveAccounts) > 0 {
			peekedAccounts = q.runMode.ExclusiveAccounts
		} else {
			peeked, err := duration(ctx, q.primaryQueueShard.Name, "account_peek", q.clock.Now(), func(ctx context.Context) ([]uuid.UUID, error) {
				return q.accountPeek(ctx, q.isSequential(), peekUntil, AccountPeekMax)
			})
			if err != nil {
				return fmt.Errorf("could not peek accounts: %w", err)
			}
			peekedAccounts = peeked
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

				if err := q.scanPartition(ctx, partitionKey, accountPartitionPeekMax, peekUntil, metricShardName, &account, &actualScannedPartitions); err != nil {
					q.log.Error("error processing account partitions", "error", err)
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
	err := q.scanPartition(ctx, partitionKey, PartitionPeekMax, peekUntil, metricShardName, nil, &actualScannedPartitions)
	if err != nil {
		return fmt.Errorf("error scanning partition: %w", err)
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

func (q *queue) scanContinuations(ctx context.Context) error {
	if !q.runMode.Continuations {
		// continuations are not enabled.
		return nil
	}

	// Have some chance of skipping continuations in this iteration.
	if rand.Float64() <= consts.QueueContinuationSkipProbability {
		return nil
	}

	eg := errgroup.Group{}
	// If we have continued partitions, process those immediately.
	q.continuesLock.Lock()
	for _, c := range q.continues {
		cont := c
		eg.Go(func() error {
			p := cont.partition
			if q.capacity() == 0 {
				// no longer any available workers for partition, so we can skip
				// work
				metrics.IncrQueueScanNoCapacityCounter(ctx, metrics.CounterOpt{PkgName: pkgName})
				return nil
			}
			if p.PartitionType != int(enums.PartitionTypeDefault) {
				return nil
			}

			q.log.Trace("continue partition processing", "partition_id", p.ID, "count", c.count)

			if err := q.processPartition(ctx, p, cont.count, false); err != nil {
				if err == ErrPartitionNotFound || err == ErrPartitionGarbageCollected {
					q.removeContinue(ctx, p, false)
					return nil
				}
				if errors.Unwrap(err) != context.Canceled {
					q.log.Error("error processing partition", "error", err)
				}
				return err
			}

			metrics.IncrQueuePartitionProcessedCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
			})
			return nil
		})
	}
	q.continuesLock.Unlock()
	return eg.Wait()
}

// processPartition processes a given partition, peeking jobs from the partition to run.
//
// It accepts a uint continuationCount which represents the number of times that the partition
// has been continued;  this occurs when a job enqueues another job to the same partition and
// hints that we have more work to do, which lowers inter-step latency on job-per-step execution
// models.
//
// randomOffset allows us to peek jobs out-of-order, and occurs when we hit concurrency key issues
// such that we can attempt to work on other jobs not blocked by heading concurrency key issues.
func (q *queue) processPartition(ctx context.Context, p *QueuePartition, continuationCount uint, randomOffset bool) error {
	if p.AccountID != uuid.Nil && q.capacityManager != nil && q.useConstraintAPI != nil {
		// If Constraint API should be used, check constraints before leasing partition
		useAPI, _ := q.useConstraintAPI(ctx, p.AccountID)
		if useAPI {
			res, _, err := q.capacityManager.Check(ctx, &constraintapi.CapacityCheckRequest{
				AccountID: p.AccountID,
				// TODO: Supply constraint items
			})
			if err != nil {
				return fmt.Errorf("could not check capacity: %w", err)
			}

			// TODO: Check capacity
			_ = res
		}
	}

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
		l, capacity, err := q.PartitionLease(ctx, p, PartitionLeaseDuration)
		p.LeaseID = l
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
		// If this is a continuation, remove it from the continuation counter.
		// This prevents us from keeping partitions as continuations forever until
		// we hit the max limit.
		q.removeContinue(ctx, p, false)
		return nil
	}
	if errors.Is(err, ErrPartitionNotFound) || errors.Is(err, ErrPartitionGarbageCollected) {
		// Another worker must have processed this partition between
		// this worker's peek and process.  Increase partition
		// contention metric and continue.  This is unsolvable.

		// If this is a continuation, remove it from the continuation counter.
		// This prevents us from keeping partitions as continuations forever until
		// we hit the max limit.
		q.removeContinue(ctx, p, false)

		metrics.IncrPartitionGoneCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name}})
		return nil
	}
	if errors.Is(err, ErrPartitionPaused) {
		// Don't return an error and remove continuations;  this isn't workable.
		q.removeContinue(ctx, p, false)
		return nil
	}

	if err != nil {
		return fmt.Errorf("error leasing partition: %w", err)
	}

	begin := q.clock.Now()
	defer func() {
		metrics.HistogramProcessPartitionDuration(ctx, q.clock.Since(begin).Milliseconds(), metrics.HistogramOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"queue_shard":     q.primaryQueueShard.Name,
				"is_continuation": continuationCount > 0,
			},
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

	metrics.HistogramQueuePeekSize(ctx, int64(len(queue)), metrics.HistogramOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard":     q.primaryQueueShard.Name,
			"is_continuation": continuationCount > 0,
		},
	})

	// Record the number of partitions we're leasing.
	metrics.IncrQueuePartitionLeasedCounter(ctx, metrics.CounterOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"queue_shard":     q.primaryQueueShard.Name,
			"is_continuation": continuationCount > 0,
		},
	})

	// parallel all queue names with internal mappings for now.
	// XXX: Allow parallel partitions for all functions except for fns opting into FIFO
	_, isSystemFn := q.queueKindMapping[p.Queue()]
	_, parallelFn := q.disableFifoForFunctions[p.Queue()]
	_, parallelAccount := q.disableFifoForAccounts[p.AccountID.String()]

	parallel := parallelFn || parallelAccount || isSystemFn

	iter := processor{
		partition:            p,
		items:                queue,
		partitionContinueCtr: continuationCount,
		queue:                q,
		denies:               newLeaseDenyList(),
		staticTime:           q.clock.Now(),
		parallel:             parallel,
	}

	if processErr := iter.iterate(ctx); processErr != nil {
		// Report the eerror.
		q.log.Error("error iterating queue items", "error", processErr, "partition", p)
		return processErr

	}

	if q.usePeekEWMA {
		if err := q.setPeekEWMA(ctx, p.FunctionID, int64(iter.ctrConcurrency+iter.ctrRateLimit)); err != nil {
			q.log.Warn("error recording concurrency limit for EWMA", "error", err)
		}
	}

	if iter.isRequeuable() && iter.isCustomKeyLimitOnly && !randomOffset && parallel {
		// We hit custom concurrency key issues.  Re-process this partition at a random offset, as long
		// as random offset is currently false (so we don't loop forever)

		// Note: we must requeue the partition to remove the lease.
		err := q.PartitionRequeue(ctx, q.primaryQueueShard, p, q.clock.Now().Truncate(time.Second).Add(PartitionConcurrencyLimitRequeueExtension), true)
		if err != nil {
			q.log.Warn("error requeuieng partition for random peek", "error", err)
		}

		return q.processPartition(ctx, p, continuationCount, true)
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
		err = q.PartitionRequeue(ctx, q.primaryQueueShard, p, q.clock.Now().Truncate(time.Second).Add(requeue), true)
		if errors.Is(err, ErrPartitionGarbageCollected) {
			q.removeContinue(ctx, p, false)
		}
		return err
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
		q.removeContinue(ctx, p, false)
		// Safe;  we're preventing this from wasting cycles in the future.
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

func (q *queue) process(
	ctx context.Context,
	i processItem,
	f osqueue.RunFunc,
) error {
	qi := i.I
	p := i.P
	continuationCtr := i.PCtr
	capacityLeaseID := i.capacityLeaseID

	var err error
	leaseID := qi.LeaseID

	// Allow the main runner to block until this work is done
	q.wg.Add(1)
	defer q.wg.Done()

	// Continually the lease while this job is being processed.
	extendLeaseTick := q.clock.NewTicker(QueueLeaseDuration / 2)
	defer extendLeaseTick.Stop()

	extendCapacityLeaseTick := q.clock.NewTicker(QueueLeaseDuration / 2)
	defer extendCapacityLeaseTick.Stop()

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
					q.log.Error("cannot extend lease since lease ID is nil", "qi", qi, "partition", p)
					// Don't extend lease since one doesn't exist
					errCh <- fmt.Errorf("cannot extend lease since lease ID is nil")
					return
				}

				// Once a job has started, use a BG context to always renew.
				leaseID, err = q.ExtendLease(context.Background(), qi, *leaseID, QueueLeaseDuration)
				if err != nil {
					// log error if unexpected; the queue item may be removed by a Dequeue() operation
					// invoked by finalize() (Cancellations, Parallelism)
					if !errors.Is(ErrQueueItemNotFound, err) {
						q.log.Error("error extending lease", "error", err, "qi", qi, "partition", p)
					}

					// always stop processing the queue item if lease cannot be extended
					errCh <- fmt.Errorf("error extending lease while processing: %w", err)
					return
				}
			case <-extendCapacityLeaseTick.Chan():
				if ctx.Err() != nil {
					// Don't extend lease when the ctx is done.
					return
				}

				// If no capacity lease is used, no-op
				if i.capacityLeaseID == ulid.Zero {
					continue
				}

				if capacityLeaseID == ulid.Zero {
					q.log.Error("cannot extend capacity lease since capacity lease ID is nil", "qi", qi, "partition", p)
					// Don't extend lease since one doesn't exist
					errCh <- fmt.Errorf("cannot extend lease since lease ID is nil")
					return
				}

				// TODO: Check if this idempotency key makes sense
				idempotencyKey := capacityLeaseID.String()

				res, err := q.capacityManager.ExtendLease(context.Background(), &constraintapi.CapacityExtendLeaseRequest{
					AccountID:      p.AccountID,
					IdempotencyKey: idempotencyKey,
					LeaseID:        capacityLeaseID,
				})
				if err != nil {
					// log error if unexpected; the queue item may be removed by a Dequeue() operation
					// invoked by finalize() (Cancellations, Parallelism)
					if !errors.Is(ErrQueueItemNotFound, err) {
						q.log.ReportError(
							err,
							"error extending capacity lease",
							logger.WithErrorReportLog(true),
							logger.WithErrorReportTags(map[string]string{
								"partitionID": p.ID,
								"accountID":   p.AccountID.String(),
								"item":        qi.ID,
								"leaseID":     capacityLeaseID.String(),
							}),
						)
					}

					// always stop processing the queue item if lease cannot be extended
					errCh <- fmt.Errorf("error extending lease while processing: %w", err)
					return
				}

				if res.LeaseID == nil {
					// Lease could not be extended
					errCh <- fmt.Errorf("failed to extend capacity lease, no new lease ID received")
					return
				}

				capacityLeaseID = *res.LeaseID
			}
		}
	}()

	// XXX: Add a max job time here, configurable.
	jobCtx, jobCancel := context.WithCancel(context.WithoutCancel(ctx))
	defer jobCancel()

	// Add the job ID to the queue context.  This allows any logic that handles the run function
	// to inspect job IDs, eg. for tracing or logging, without having to thread this down as
	// arguments.
	jobCtx = osqueue.WithJobID(jobCtx, qi.ID)
	// Same with the group ID, if it exists.
	if qi.Data.GroupID != "" {
		jobCtx = state.WithGroupID(jobCtx, qi.Data.GroupID)
	}

	startedAt := q.clock.Now()
	go func() {
		longRunningJobStatusTick := q.clock.NewTicker(5 * time.Minute)
		defer longRunningJobStatusTick.Stop()

		for {
			select {
			case <-jobCtx.Done():
				return
			case <-longRunningJobStatusTick.Chan():
			}

			q.log.Debug("long running queue job tick", "item", qi, "dur", q.clock.Now().Sub(startedAt).String())
		}
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Always retry this job.
				stack := debug.Stack()
				q.log.Error("job panicked", "error", fmt.Errorf("%v", r), "stack", string(stack))
				errCh <- osqueue.AlwaysRetryError(fmt.Errorf("job panicked: %v", r))
			}
		}()

		// This job may be up to 1999 ms in the future, as explained in processPartition.
		// Just... wait until the job is available.
		delay := time.UnixMilli(qi.AtMS).Sub(q.clock.Now())

		if delay > 0 {
			<-q.clock.After(delay)
			q.log.Trace("delaying job in memory",
				"at", qi.AtMS,
				"ms", delay.Milliseconds(),
			)
		}
		n := q.clock.Now()

		// Track the sojourn (concurrency) latency.
		sojourn := qi.SojournLatency(n)
		doCtx := context.WithValue(jobCtx, sojournKey, sojourn)

		// Track the latency on average globally.  Do this in a goroutine so that it doesn't
		// at all delay the job during concurrenty locking contention.
		if qi.WallTimeMS == 0 {
			qi.WallTimeMS = qi.AtMS // backcompat while WallTimeMS isn't valid.
		}
		latency := qi.Latency(n)
		doCtx = context.WithValue(doCtx, latencyKey, latency)

		// store started at and latency in ctx
		doCtx = context.WithValue(doCtx, startedAtKey, n)

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
			Latency:             latency,
			SojournDelay:        sojourn,
			Priority:            q.ppf(ctx, p),
			QueueShardName:      q.primaryQueueShard.Name,
			ContinueCount:       continuationCtr,
			RefilledFromBacklog: qi.RefilledFrom,
		}

		// Call the run func.
		res, err := f(doCtx, runInfo, qi.Data)
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

		if res.ScheduledImmediateJob {
			// Add the partition to be continued again.  Note that if we've already
			// continued beyond the limit this is a noop.
			q.addContinue(ctx, &p, continuationCtr+1)
		}

		// Closing this channel prevents the goroutine which extends lease from leaking,
		// and dequeues the job
		close(doneCh)
	}()

	// When capacity is leased, release it after requeueing/dequeueing the item.
	// This MUST happen to free up concurrency capacity in a timely manner for
	// the next worker to lease a queue item.
	if capacityLeaseID != ulid.Zero {
		defer func() {
			res, err := q.capacityManager.Release(ctx, &constraintapi.CapacityReleaseRequest{
				AccountID:      p.AccountID,
				IdempotencyKey: qi.ID,
				LeaseID:        capacityLeaseID,
			})
			if err != nil {
				q.log.ReportError(err, "failed to release capacity")
			}

			q.log.Trace("released capacity", "res", res)
		}()
	}

	select {
	case err := <-errCh:
		// Job errored or extending lease errored.  Cancel the job ASAP.
		jobCancel()

		if osqueue.ShouldRetry(err, qi.Data.Attempt, qi.Data.GetMaxAttempts()) {
			at := q.backoffFunc(qi.Data.Attempt)

			// Attempt to find any RetryAtSpecifier in the error tree.
			if specifier := osqueue.AsRetryAtError(err); specifier != nil {
				next := specifier.NextRetryAt()
				at = *next
			}

			if !osqueue.IsAlwaysRetryable(err) {
				qi.Data.Attempt += 1
			}

			qi.AtMS = at.UnixMilli()
			if err := q.Requeue(context.WithoutCancel(ctx), q.primaryQueueShard, qi, at); err != nil {
				if err == ErrQueueItemNotFound {
					// Safe. The executor may have dequeued.
					return nil
				}

				q.log.Error("error requeuing job", "error", err, "item", qi)
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
			q.log.Warn("received queue quit error", "error", err)
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

func (q *queue) activeCheckerLease() *ulid.ULID {
	q.activeCheckerLeaseLock.RLock()
	defer q.activeCheckerLeaseLock.RUnlock()
	if q.activeCheckerLeaseID == nil {
		return nil
	}
	copied := *q.activeCheckerLeaseID
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
	if peekSize, ok := q.peekSizeForFunctions[p.ID]; ok {
		return peekSize
	}
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

//nolint:unused // this code remains to be enabled on demand
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

func (q *queue) isActiveChecker() bool {
	l := q.activeCheckerLease()
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
	partition *QueuePartition
	items     []*osqueue.QueueItem
	// partitionContinueCtr is the number of times the partition has currently been
	// continued already in the chain.  we must record this such that a partition isn't
	// forced indefinitely.
	partitionContinueCtr uint

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
			logger.StdlibLogger(ctx).Error("nil queue item in partition", "partition", p.partition)
			continue
		}

		if p.parallel {
			item := *i
			eg.Go(func() error {
				err := p.process(ctx, &item)
				if err != nil {
					// NOTE: ignore if the queue item is not found
					if errors.Is(err, ErrQueueItemNotFound) {
						return nil
					}
				}
				return err
			})
			continue
		}

		// non-parallel (sequential fifo) processing.
		if err = p.process(ctx, i); err != nil {
			// NOTE: ignore if the queue item is not found
			if errors.Is(err, ErrQueueItemNotFound) {
				continue
			}
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
	l := p.queue.log.With("partition", p.partition, "item", item)

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

	var leaseOptions []leaseOptionFn

	constraintRes, err := p.queue.itemLeaseConstraintCheck(ctx, *p.partition, item, p.staticTime)
	if err != nil {
		p.queue.sem.Release(1)
		metrics.WorkerQueueCapacityCounter(ctx, -1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": p.queue.primaryQueueShard.Name}})

		return fmt.Errorf("could not check constraints to lease item: %w", err)
	}

	if constraintRes.fallbackIdempotencyKey != "" {
		leaseOptions = append(leaseOptions, LeaseOptionFallbackIdempotencyKey(constraintRes.fallbackIdempotencyKey))
	}

	if constraintRes.skipConstraintChecks {
		leaseOptions = append(leaseOptions, LeaseOptionDisableConstraintChecks(true))
	}

	var leaseID *ulid.ULID

	switch constraintRes.limitingConstraint {
	case enums.QueueConstraintNotLimited:

		// Attempt to lease this item before passing this to a worker.  We have to do this
		// synchronously as we need to lease prior to requeueing the partition pointer. If
		// we don't do this here, the workers may not lease the items before calling Peek
		// to re-enqeueu the pointer, which then increases contention - as we requeue a
		// pointer too early.
		//
		// This is safe:  only one process runs scan(), and we guard the total number of
		// available workers with the above semaphore.
		leaseID, err = duration(ctx, p.queue.primaryQueueShard.Name, "lease", p.queue.clock.Now(), func(ctx context.Context) (*ulid.ULID, error) {
			return p.queue.Lease(
				ctx,
				*item,
				QueueLeaseDuration,
				p.staticTime,
				p.denies,
				leaseOptions...,
			)
		})
		// NOTE: If this loop ends in an error, we must _always_ release an item from the
		// semaphore to free capacity.  This will happen automatically when the worker
		// finishes processing a queue item on success.
		if err != nil {
			// Continue on and handle the error below.
			p.queue.sem.Release(1)
			metrics.WorkerQueueCapacityCounter(ctx, -1, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": p.queue.primaryQueueShard.Name}})
		}
	case enums.QueueConstraintThrottle:
	case enums.QueueConstraintAccountConcurrency:
	case enums.QueueConstraintFunctionConcurrency:
	case enums.QueueConstraintCustomConcurrencyKey1:
	case enums.QueueConstraintCustomConcurrencyKey2:
	default:
		// Limited but the constraint is unknown?

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

	l = l.With(
		"cause", cause,
		"item_id", item.ID,
		"account_id", item.Data.Identifier.AccountID.String(),
		"env_id", item.WorkspaceID.String(),
		"app_id", item.Data.Identifier.AppID.String(),
		"fn_id", item.FunctionID.String(),
		"queue_shard", p.queue.primaryQueueShard.Name,
	)

	// used for error reporting
	errTags := map[string]string{}
	if cause != nil {
		errTags["cause"] = cause.Error()
	}
	if leaseID != nil {
		errTags["lease"] = leaseID.String()
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

		if p.queue.itemEnableKeyQueues(ctx, *item) {
			err := p.queue.Requeue(ctx, p.queue.primaryQueueShard, *item, time.UnixMilli(item.AtMS))
			if err != nil && !errors.Is(err, ErrQueueItemNotFound) {
				l.ReportError(err, "could not requeue item to backlog after hitting throttle limit",
					logger.WithErrorReportTags(errTags),
				)
				return fmt.Errorf("could not requeue to backlog: %w", err)
			}

			metrics.IncrRequeueExistingToBacklogCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": p.queue.primaryQueueShard.Name,
					// "partition_id": item.FunctionID.String(),
					"status": "throttled",
				},
			})
		}

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

		if p.queue.itemEnableKeyQueues(ctx, *item) {
			err := p.queue.Requeue(ctx, p.queue.primaryQueueShard, *item, time.UnixMilli(item.AtMS))
			if err != nil && !errors.Is(err, ErrQueueItemNotFound) {
				l.ReportError(err, "could not requeue item to backlog after hitting concurrency limit",
					logger.WithErrorReportTags(errTags),
				)
				return fmt.Errorf("could not requeue to backlog: %w", err)
			}

			metrics.IncrRequeueExistingToBacklogCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": p.queue.primaryQueueShard.Name,
					// "partition_id": item.FunctionID.String(),
					"status": status,
				},
			})
		}

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

		if p.queue.itemEnableKeyQueues(ctx, *item) {
			err := p.queue.Requeue(ctx, p.queue.primaryQueueShard, *item, time.UnixMilli(item.AtMS))
			if err != nil && !errors.Is(err, ErrQueueItemNotFound) {
				l.ReportError(err, "could not requeue item to backlog after hitting custom concurrency limit",
					logger.WithErrorReportTags(errTags),
				)
				return fmt.Errorf("could not requeue to backlog: %w", err)
			}

			metrics.IncrRequeueExistingToBacklogCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"queue_shard": p.queue.primaryQueueShard.Name,
					// "partition_id": item.FunctionID.String(),
					"status": "custom_key_concurrency_limit",
				},
			})
		}
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
	if err != nil || leaseID == nil {
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
	p.queue.workers <- processItem{
		P:    *p.partition,
		I:    *item,
		PCtr: p.partitionContinueCtr,

		capacityLeaseID: capacityLeaseID,
	}

	return nil
}

func (p *processor) isRequeuable() bool {
	// if we have concurrency OR we hit rate limiting/throttling.
	return p.ctrConcurrency > 0 || (p.ctrRateLimit > 0 && p.ctrConcurrency == 0 && p.ctrSuccess == 0)
}
