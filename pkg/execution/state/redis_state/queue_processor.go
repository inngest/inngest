package redis_state

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VividCortex/ewma"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/uber-go/tally/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"gonum.org/v1/gonum/stat/sampleuv"
)

const (
	minWorkersFree = 5

	// Mtric consts
	counterQueueItemsStarted                = "queue_items_started_total"              // Queue item started
	counterQueueItemsErrored                = "queue_items_errored_total"              // Queue item errored
	counterQueueItemsComplete               = "queue_items_complete_total"             // Queue item finished
	counterQueueItemsEnqueued               = "queue_items_enqueued_total"             // Item enqueued
	counterQueueItemsProcessLeaseExists     = "queue_items_process_lease_exists_total" // Scanned an item with an exisitng lease
	counterQueueItemsLeaseConflict          = "queue_items_lease_conflict_total"       // Attempt to lease an item with an existing lease
	counterQueueItemsGone                   = "queue_items_gone_total"                 // Attempt to lease a dequeued item
	counterSequentialLeaseClaims            = "queue_sequential_lease_claims_total"    // Sequential lease claimed by worker
	counterPartitionProcessNoCapacity       = "partition_process_no_capacity_total"    // Processing items but there's no more capacity
	counterPartitionProcessItems            = "partition_process_items_total"          // Leased a queue item within a partition to begin work
	counterConcurrencyLimit                 = "concurrency_limit_processing_total"
	counterPartitionProcess                 = "partition_process_total"
	counterPartitionLeaseConflict           = "partition_lease_conflict_total"
	counterPartitionConcurrencyLimitReached = "partition_concurrency_limit_reached_total"
	counterPartitionGone                    = "partition_gone_total"
	gaugeQueueItemLatencyEWMA               = "queue_item_latency_ewma"
	histogramItemLatency                    = "queue_item_latency_duration"
)

var (
	// ShardTickTime is the duration in which we periodically check shards for
	// lease information, etc.
	ShardTickTime = 15 * time.Second
	// ShardLeaseTime is how long shards are leased.
	ShardLeaseTime = 10 * time.Second

	maxShardLeaseAttempts = 10
)

var (
	latencyAvg ewma.MovingAverage
	latencySem *sync.Mutex

	latencyBuckets = tally.DurationBuckets{
		time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
		25 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		250 * time.Millisecond,
		500 * time.Millisecond,
		time.Second,
		10 * time.Second,
		time.Minute,
	}

	startedAtKey = startedAtCtxKey{}
	sojournKey   = sojournCtxKey{}
	latencyKey   = latencyCtxKey{}
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

func (q *queue) Enqueue(ctx context.Context, item osqueue.Item, at time.Time) error {
	// propagate
	if item.Metadata == nil {
		item.Metadata = map[string]string{}
	}

	id := ""
	if item.JobID != nil {
		id = *item.JobID
	}

	var queueName *string
	if name, ok := q.queueKindMapping[item.Kind]; ok {
		queueName = &name
	}
	// item.QueueName takes precedence if not nil
	if item.QueueName != nil {
		queueName = item.QueueName
	}

	go q.scope.Tagged(map[string]string{
		"kind": item.Kind,
	}).Counter(counterQueueItemsEnqueued).Inc(1)

	qi := QueueItem{
		ID:          id,
		AtMS:        at.UnixMilli(),
		WorkspaceID: item.WorkspaceID,
		WorkflowID:  item.Identifier.WorkflowID,
		Data:        item,
		// Only use the queue name if provided by queueKindMapping.
		// Otherwise, this defaults to WorkflowID.
		QueueName:  queueName,
		WallTimeMS: at.UnixMilli(),
	}

	// Use the queue item's score, ensuring we process older function runs first
	// (eg. before at)
	next := time.UnixMilli(qi.Score())

	if factor := qi.Data.GetPriorityFactor(); factor != 0 {
		// Ensure we mutate the AtMS time by the given priority factor.
		qi.AtMS -= factor
	}

	_, err := q.EnqueueItem(ctx, qi, next)
	if err != nil {
		return err
	}

	return nil
}

func (q *queue) Run(ctx context.Context, f osqueue.RunFunc) error {
	for i := int32(0); i < q.numWorkers; i++ {
		go q.worker(ctx, f)
	}

	go q.queueGauges(ctx)
	go q.claimShards(ctx)
	go q.claimSequentialLease(ctx)
	go q.runScavenger(ctx)

	tick := time.NewTicker(q.pollTick)

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
		case <-tick.C:
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
				if errors.Unwrap(err) != context.Canceled {
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

func (q *queue) claimShards(ctx context.Context) {
	if q.sf == nil {
		// TODO: Inspect denylists and whether this worker is capable of leasing
		// shards.  Note that shards should only be created for SDK-based workers;
		// if you use the queue for anything else other than step jobs, the worker
		// cannot lease jobs.  To this point, we should make this opt-in instead of
		// opt-out to prevent errors.
		//
		// For now, you have to provide a shardFinder to lease shards.
		q.logger.Info().Msg("no shard finder;  skipping shard claiming")
		return
	}

	// Report shard gauge metrics.
	go q.shardGauges(ctx)

	scanTick := time.NewTicker(ShardTickTime)
	leaseTick := time.NewTicker(ShardLeaseTime / 2)

	// records whether we're leasing
	var leasing int32

	for {
		if q.isSequential() {
			// Sequential workers never lease shards.  They always run in order
			// on the global partition queue.
			<-scanTick.C
			continue
		}

		select {
		case <-ctx.Done():
			// TODO: Remove leases immediately from backing store.
			scanTick.Stop()
			leaseTick.Stop()
			return
		case <-scanTick.C:
			go func() {
				if !atomic.CompareAndSwapInt32(&leasing, 0, 1) {
					// Only one lease can occur at once.
					q.logger.Debug().Msg("already leasing shards")
					return
				}

				// Always reset the leasing op to zero, allowing us to lease again.
				defer func() { atomic.StoreInt32(&leasing, 0) }()

				// Retry claiming leases until all shards have been taken.  All operations
				// must succeed, even if it leaves us spinning.  Note that scanShards filters
				// out unnecessary leases and shards that have already been leased.
				retry := true
				n := 0
				for retry && n < maxShardLeaseAttempts {
					n++
					var err error
					retry, err = q.scanShards(ctx)
					if err != nil {
						q.logger.Error().Err(err).Msg("error scanning and leasing shards")
						return
					}
					if retry {
						<-time.After(time.Duration(rand.Intn(50)) * time.Millisecond)
					}
				}
			}()
		case <-leaseTick.C:
			// Copy the slice to prevent locking/concurrent access.
			existingLeases := q.getShardLeases()

			for _, s := range existingLeases {
				// Attempt to lease all ASAP, even if the backing store is single threaded.
				go func(ls leasedShard) {
					nextLeaseID, err := q.renewShardLease(ctx, &ls.Shard, ShardLeaseTime, ls.Lease)
					if err != nil {
						q.logger.Error().Err(err).Msg("error renewing shard lease")
						return
					}
					q.logger.Debug().Interface("shard", ls).Msg("renewed shard lease")
					// Update the lease ID so that we have this stored appropriately for
					// the next renewal.
					q.addLeasedShard(ctx, &ls.Shard, *nextLeaseID)
				}(s)
			}
		}
	}
}

func (q *queue) scanShards(ctx context.Context) (retry bool, err error) {
	// TODO: Make instances of *queue register worker information when calling
	//       Run().
	//       Fetch this information, and correctly assign workers to shard maps
	//       based on the distribution of items in the queue here.  This lets
	//       us oversubscribe appropriately.
	shardMap, err := q.getShards(ctx)
	if err != nil {
		q.logger.Error().Err(err).Msg("error fetching shards")
		return
	}
	shards, err := q.filterShards(ctx, shardMap)
	if err != nil {
		q.logger.Error().Err(err).Msg("error filtering shards")
		return
	}

	if len(shards) == 0 {
		return
	}

	for _, shard := range shards {
		leaseID, err := q.leaseShard(ctx, shard, ShardLeaseTime, len(shard.Leases))
		if err == nil {
			// go q.counter(ctx, "queue_shard_lease_success_total", 1, map[string]any{
			// 	"shard_name": shard.Name,
			// })
			q.addLeasedShard(ctx, shard, *leaseID)
			q.logger.Debug().Interface("shard", shard).Str("lease_id", leaseID.String()).Msg("leased shard")
		} else {
			q.logger.Debug().Interface("shard", shard).Err(err).Msg("failed to lease shard")
		}

		// go q.counter(ctx, "queue_shard_lease_conflict_total", 1, map[string]any{
		// 	"shard_name": shard.Name,
		// })

		switch err {
		case errShardNotFound:
			// This is okay;  the shard was removed when trying to lease
			continue
		case errShardIndexLeased:
			// This is okay;  another worker grabbed the lease.  No need to retry
			// as another worker grabbed this.
			continue
		case errShardIndexInvalid:
			// A lease expired while trying to lease — try again.
			retry = true
		default:
			return true, err
		}
	}

	return retry, nil
}

func (q *queue) addLeasedShard(ctx context.Context, shard *QueueShard, lease ulid.ULID) {
	for i, n := range q.shardLeases {
		if n.Shard.Name == shard.Name {
			// Updated in place.
			q.shardLeaseLock.Lock()
			q.shardLeases[i] = leasedShard{
				Lease: lease,
				Shard: *shard,
			}
			q.shardLeaseLock.Unlock()
			return
		}
	}
	// Not updated in place, so add to the list and return.
	q.shardLeaseLock.Lock()
	q.shardLeases = append(q.shardLeases, leasedShard{
		Lease: lease,
		Shard: *shard,
	})
	q.shardLeaseLock.Unlock()
}

// filterShards filters shards during assignment, removing any shards that this worker
// has already leased;  any shards that have already had their leasing requirements met;
// and priority shuffles shards to lease in a non-deterministic (but prioritized) order.
//
// The returned shards are safe to be leased, and should be attempted in-order.
func (q *queue) filterShards(ctx context.Context, shards map[string]*QueueShard) ([]*QueueShard, error) {
	if len(shards) == 0 {
		return nil, nil
	}

	// Copy the slice to prevent locking/concurrent access.
	for _, v := range q.getShardLeases() {
		delete(shards, v.Shard.Name)
	}

	weights := []float64{}
	shuffleIdx := []*QueueShard{}
	for _, v := range shards {
		// XXX: Here we can add latency targets, etc.

		validLeases := []ulid.ULID{}
		for _, l := range v.Leases {
			if time.UnixMilli(int64(l.Time())).After(getNow()) {
				validLeases = append(validLeases, l)
			}
		}
		// Replace leases with the # of valid leases.
		v.Leases = validLeases

		if len(validLeases) >= int(v.GuaranteedCapacity) {
			continue
		}

		weights = append(weights, float64(v.Priority))
		shuffleIdx = append(shuffleIdx, v)
	}

	if len(shuffleIdx) == 1 {
		return shuffleIdx, nil
	}

	// Reduce the likelihood of all workers attempting to claim shards by
	// randomly shuffling.  Note that high priority shards will still be
	// likely to come first with some contention.
	w := sampleuv.NewWeighted(weights, rnd)
	result := make([]*QueueShard, len(weights))
	for n := range result {
		idx, ok := w.Take()
		if !ok && len(result) < len(weights)-1 {
			return result, ErrWeightedSampleRead
		}
		result[n] = shuffleIdx[idx]
	}

	return result, nil
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
	leaseID, err := q.ConfigLease(ctx, q.kg.Sequential(), ConfigLeaseDuration, q.sequentialLease())
	if err != ErrConfigAlreadyLeased && err != nil {
		q.quit <- err
		return
	}

	q.seqLeaseLock.Lock()
	q.seqLeaseID = leaseID
	q.seqLeaseLock.Unlock()

	tick := time.NewTicker(ConfigLeaseDuration / 3)
	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return
		case <-tick.C:
			leaseID, err := q.ConfigLease(ctx, q.kg.Sequential(), ConfigLeaseDuration, q.sequentialLease())
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
				go q.scope.Counter(counterSequentialLeaseClaims).Inc(1)
			}
			q.seqLeaseID = leaseID
			q.seqLeaseLock.Unlock()
		}
	}
}

func (q *queue) runScavenger(ctx context.Context) {
	// Attempt to claim the lease immediately.
	leaseID, err := q.ConfigLease(ctx, q.kg.Scavenger(), ConfigLeaseDuration, q.scavengerLease())
	if err != ErrConfigAlreadyLeased && err != nil {
		q.quit <- err
		return
	}

	q.scavengerLeaseLock.Lock()
	q.scavengerLeaseID = leaseID // no-op if not leased
	q.scavengerLeaseLock.Unlock()

	tick := time.NewTicker(ConfigLeaseDuration / 3)
	scavenge := time.NewTicker(30 * time.Second)

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			scavenge.Stop()
			return
		case <-scavenge.C:
			// Scavenge the items
			if q.isScavenger() {
				count, err := q.Scavenge(ctx)
				if err != nil {
					q.logger.Error().Err(err).Msg("error claiming scavenger lease")
				}
				if count > 0 {
					q.logger.Info().Int("len", count).Msg("scavenged lost jobs")
				}
			}
		case <-tick.C:
			// Attempt to re-lease the lock.
			leaseID, err := q.ConfigLease(ctx, q.kg.Scavenger(), ConfigLeaseDuration, q.scavengerLease())
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
				go q.scope.Counter(counterSequentialLeaseClaims).Inc(1)
			}
			q.scavengerLeaseID = leaseID
			q.scavengerLeaseLock.Unlock()
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
			err := q.process(processCtx, i.P, i.I, i.S, f)
			q.sem.Release(1)
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

func (q *queue) scan(ctx context.Context) error {
	if q.capacity() == 0 {
		return nil
	}

	// Store the shard that we processed, allowing us to eventually pass this
	// down to the job for stat tracking.
	var (
		shard           *QueueShard
		metricShardName = "<global>" // default global name for metrics in this function
	)

	// By default, use the global partition
	partitionKey := q.kg.GlobalPartitionIndex()

	// If this worker has leased shards, those take priority 95% of the time.  There's a 5% chance that the
	// worker still works on the global queue.
	existingLeases := q.getShardLeases()

	if len(existingLeases) > 0 {
		// Pick a random item between the shards.
		i := rand.Intn(len(existingLeases))
		shard = &existingLeases[i].Shard
		// Use the shard partition
		partitionKey = q.kg.ShardPartitionIndex(shard.Name)
		metricShardName = "<shard>:" + shard.Name
	}

	// Peek 1s into the future to pull jobs off ahead of time, minimizing 0 latency
	partitions, err := q.partitionPeek(ctx, partitionKey, q.isSequential(), getNow().Add(PartitionLookahead), PartitionPeekMax)
	if err != nil {
		return err
	}

	eg := errgroup.Group{}

	for _, ptr := range partitions {
		p := *ptr
		eg.Go(func() error {
			if q.capacity() == 0 {
				// no longer any available workers for partition, so we can skip
				// work
				q.int64counter(ctx, "inngest_queue_scan_no_capacity_total", 1)
				return nil
			}
			if err := q.processPartition(ctx, &p, shard); err != nil {
				if err == ErrPartitionNotFound || err == ErrPartitionGarbageCollected {
					// Another worker grabbed the partition, or the partition was deleted
					// during the scan by an another worker.
					// TODO: Increase internal metrics
					return nil
				}
				if errors.Unwrap(err) != context.Canceled {
					q.logger.Error().Err(err).Msg("error processing partition")
				}
				return err
			}

			q.int64counter(ctx, "inngest_queue_partition_processed_total", 1, attribute.KeyValue{
				Key:   attribute.Key("shard"),
				Value: attribute.StringValue(metricShardName),
			})
			return nil
		})
	}

	return eg.Wait()
}

// NOTE: Shard is only passed as a reference if the partition was peeked from
// a shard.  It exists for accounting and tracking purposes only, eg. to report shard metrics.
func (q *queue) processPartition(ctx context.Context, p *QueuePartition, shard *QueueShard) error {
	q.scope.Counter(counterPartitionProcess).Inc(1)

	// Attempt to lease items.  This checks partition-level concurrency limits
	//
	// For oprimization, because this is the only thread that can be leasing
	// jobs for this partition, we store the partition limit and current count
	// as a variable and iterate in the loop without loading keys from the state
	// store.
	//
	// There's no way to know when queue items finish processing;  we don't
	// store average runtimes for queue items (and we don't know because
	// items are dynamic generators).  This means that we have to delay
	// processing the partition by N seconds, meaning the latency is increased by
	// up to this period for scheduled items behind the concurrency limits.
	_, err := q.PartitionLease(ctx, p, PartitionLeaseDuration)
	if err == ErrPartitionConcurrencyLimit {
		for _, l := range q.lifecycles {
			// Track lifecycles; this function hit a partition limit ahead of
			// even being leased, meaning the function is at max capacity and we skio
			// scanning of jobs altogether.
			go l.OnConcurrencyLimitReached(context.WithoutCancel(ctx), p.WorkflowID)
		}
		q.scope.Counter(counterPartitionConcurrencyLimitReached).Inc(1)
		return q.PartitionRequeue(ctx, p, getNow().Truncate(time.Second).Add(PartitionConcurrencyLimitRequeueExtension), true)
	}
	if err == ErrPartitionAlreadyLeased {
		q.scope.Counter(counterPartitionLeaseConflict).Inc(1)
		return nil
	}
	if err == ErrPartitionNotFound {
		// Another worker must have pocessed this partition between
		// this worker's peek and process.  Increase partition
		// contention metric and continue.  This is unsolvable.
		q.scope.Counter(counterPartitionGone).Inc(1)
		return nil
	}
	if err != nil {
		return fmt.Errorf("error leasing partition: %w", err)
	}

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
	fetch := getNow().Truncate(time.Second).Add(PartitionLookahead)
	queue, err := q.Peek(peekCtx, p.Queue(), fetch, q.peekSize())
	if err != nil {
		return err
	}

	var (
		processErr error

		// These flags are used to handle partition rqeueueing.
		ctrSuccess     int32
		ctrConcurrency int32
		ctrRateLimit   int32
	)

	// Record the number of partitions we're leasing.
	q.int64counter(ctx, "inngest_queue_partition_lease_total", 1)

	// staticTime is used as the processing time for all items in the queue.
	// We process queue items sequentially, and time progresses linearly as each
	// queue item is processed.  We want to use a static time to prevent out-of-order
	// processing with regards to things like rate limiting;  if we use time.Now(),
	// queue items later in the array may be processed before queue items earlier in
	// the array depending on eg. a rate limit becoming available half way through
	// iteration.
	staticTime := getNow()

ProcessLoop:
	for _, qi := range queue {
		// TODO: Create an in-memory mapping of rate limit keys that have been hit,
		//       and don't bother to process if the queue item has a limited key.  This
		//       lessens work done in the queue, as we can `continue` immediately.

		if q.capacity() == 0 {
			// no longer any available workers for partition, so we can skip
			// work for now.
			q.int64counter(ctx, "inngest_queue_process_no_capacity_total", 1)
			break ProcessLoop
		}

		item := qi
		if item.IsLeased(getNow()) {
			q.int64counter(ctx, "inngest_queue_partition_lease_contention_total", 1)
			continue
		}

		// Cbeck if there's capacity from our local workers atomically prior to leasing our tiems.
		if !q.sem.TryAcquire(1) {
			q.int64counter(ctx, "inngest_queue_partition_process_no_capacity_total", 1)
			break
		}

		// Attempt to lease this item before passing this to a worker.  We have to do this
		// synchronously as we need to lease prior to requeueing the partition pointer. If
		// we don't do this here, the workers may not lease the items before calling Peek
		// to re-enqeueu the pointer, which then increases contention - as we requeue a
		// pointer too early.
		//
		// This is safe:  only one process runs scan(), and we guard the total number of
		// available workers with the above semaphore.
		leaseID, err := q.Lease(ctx, *p, *item, QueueLeaseDuration, staticTime)

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

		if p.Last > 0 && p.Last > item.AtMS {
			// Fudge the earliest peek time because we know this wasn't peeked and so
			// the peek time wasn't set;  but, as we were still processing jobs after
			// the job was added this item was concurrency-limited.
			item.EarliestPeekTime = item.AtMS
		}

		// NOTE: If this loop ends in an error, we must _always_ release an item from the
		// semaphore to free capacity.  This will happen automatically when the worker
		// finishes processing a queue item on success.
		if err != nil {
			q.sem.Release(1)
		}

		switch err {
		case ErrQueueItemThrottled:
			ctrRateLimit++
			processErr = nil
			continue
		case ErrPartitionConcurrencyLimit, ErrAccountConcurrencyLimit:
			ctrConcurrency++
			// Since the queue is at capacity on a fn or account level, no
			// more jobs in this loop should be worked on - so break.
			//
			// Even if we have capacity for the next job in the loop we do NOT
			// want to claim the job, as this breaks ordering guarantees.  The
			// only safe thing to do when we hit a function or account level
			// concurrency key.
			processErr = nil
			break ProcessLoop
		case ErrConcurrencyLimitCustomKey0, ErrConcurrencyLimitCustomKey1:
			ctrConcurrency++
			// Custom concurrency keys are different.  Each job may have a different key,
			// so we cannot break the loop in case the next job has a different key and
			// has capacity.
			//
			// In an ideal world we'd denylist each concurrency key that's been
			// limited here, then ignore any other jobs from being leased as we continue
			// to iterate through the loop.
			//
			// This maintains FIFO ordering amongst all custom concurrency keys.  For now,
			// we move to the next job.  Note that capacity may be available for another job
			// in the loop, meaning out of order jobs for now.

			// TODO: Grab the key from the custom limit
			//       Set key in denylist lookup table.
			//       Modify above loop entrance:
			//         Check lookup table for all concurrency keys
			//         If present, skip item
			processErr = nil
			continue
		case ErrQueueItemNotFound:
			// This is an okay error.  Move to the next job item.
			ctrSuccess++ // count as a success for stats purposes.
			processErr = nil
			continue
		case ErrQueueItemAlreadyLeased:
			// This is an okay error.  Move to the next job item.
			ctrSuccess++ // count as a success for stats purposes.
			processErr = nil
			continue
		}

		// Handle other errors.
		if err != nil {
			processErr = fmt.Errorf("error leasing in process: %w", err)
			break ProcessLoop
		}

		// Assign the lease ID and pass this to be handled by the available worker.
		// There should always be capacity on this queue as we track capacity via
		// a semaphore.
		item.LeaseID = leaseID

		// increase success counter.
		ctrSuccess++

		q.workers <- processItem{P: *p, I: *item, S: shard}
	}

	// If we've hit concurrency issues OR we've only hit rate limit issues, re-enqueue the partition
	// with a force:  ensure that we won't re-scan it until 2 seconds in the future.
	if ctrConcurrency > 0 || (ctrRateLimit > 0 && ctrConcurrency == 0 && ctrSuccess == 0) {
		for _, l := range q.lifecycles {
			go l.OnConcurrencyLimitReached(context.WithoutCancel(ctx), p.WorkflowID)
		}
		// Requeue this partition as we hit concurrency limits.
		q.int64counter(ctx, "inngest_queue_partition_concurrency_limit_total", 1)
		return q.PartitionRequeue(ctx, p, getNow().Truncate(time.Second).Add(PartitionConcurrencyLimitRequeueExtension), true)
	}

	if processErr != nil {
		// This wasn't a concurrency error so handle things separately.
		return processErr
	}

	// XXX: If we haven't been able to lease a single item, ensure we enqueue this
	// for a minimum of 5 seconds.

	// Requeue the partition, which reads the next unleased job or sets a time of
	// 30 seconds.  This is why we have to lease items above, else this may return an item that is
	// about to be leased and processed by the worker.
	err = q.PartitionRequeue(ctx, p, getNow().Add(PartitionRequeueExtension), false)
	if err == ErrPartitionGarbageCollected {
		// Safe;  we're preventing this from wasting cycles in the future.
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

func (q *queue) process(ctx context.Context, p QueuePartition, qi QueueItem, s *QueueShard, f osqueue.RunFunc) error {
	var err error
	leaseID := qi.LeaseID

	scope := q.scope.Tagged(map[string]string{
		"kind": qi.Data.Kind,
	})

	// Allow the main runner to block until this work is done
	q.wg.Add(1)
	defer q.wg.Done()

	// Continually the lease while this job is being processed.
	extendLeaseTick := time.NewTicker(QueueLeaseDuration / 2)
	defer extendLeaseTick.Stop()

	errCh := make(chan error)
	doneCh := make(chan struct{})

	// Continually extend lease in the background while we're working on this job
	go func() {
		for {
			select {
			case <-doneCh:
				return
			case <-extendLeaseTick.C:
				if ctx.Err() != nil {
					// Don't extend lease when the ctx is done.
					return
				}
				leaseID, err = q.ExtendLease(ctx, p, qi, *leaseID, QueueLeaseDuration)
				if err != nil && err != ErrQueueItemNotFound && errors.Unwrap(err) != context.Canceled {
					// XXX: Increase counter here.
					q.logger.Error().Err(err).Msg("error extending lease")
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
		delay := time.Until(time.UnixMilli(qi.AtMS))

		if delay > 0 {
			<-time.After(delay)
			q.logger.Trace().
				Int64("at", qi.AtMS).
				Int64("ms", delay.Milliseconds()).
				Msg("delaying job in memory")
		}

		n := getNow()

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
			scope.Gauge(gaugeQueueItemLatencyEWMA).Update(latencyAvg.Value() / 1e6)
			latencySem.Unlock()

			// Set the metrics historgram and gauge, which reports the ewma value.
			scope.Histogram(histogramItemLatency, latencyBuckets).RecordDuration(latency)
		}()

		go scope.Counter(counterQueueItemsStarted).Inc(1)

		runInfo := osqueue.RunInfo{
			Latency:      latency,
			SojournDelay: sojourn,
			Priority:     p.Priority,
			ShardName:    "<global>",
		}
		if s != nil {
			runInfo.ShardName = s.Name
		}

		// Call the run func.
		err := f(jobCtx, runInfo, qi.Data)
		extendLeaseTick.Stop()
		if err != nil {
			go scope.Counter(counterQueueItemsErrored).Inc(1)
			errCh <- err
			return
		}
		go scope.Counter(counterQueueItemsComplete).Inc(1)

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

			qi.Data.Attempt += 1
			qi.AtMS = at.UnixMilli()
			q.logger.Warn().Err(err).
				Str("queue", qi.Queue()).
				Int64("at_ms", at.UnixMilli()).
				Msg("requeuing job")
			if err := q.Requeue(context.WithoutCancel(ctx), p, qi, at); err != nil {
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
		q.logger.Info().Interface("item", qi).Msg("dequeueing failed job")
		if err := q.Dequeue(context.WithoutCancel(ctx), p, qi); err != nil {
			return err
		}

		if _, ok := err.(osqueue.QuitError); ok {
			q.logger.Warn().Err(err).Msg("received queue quit error")
			q.quit <- err
			return err
		}

	case <-doneCh:
		if err := q.Dequeue(context.WithoutCancel(ctx), p, qi); err != nil {
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

// peekSize returns the total number of available workers which can consume individual
// queue items.
func (q *queue) peekSize() int64 {
	f := q.capacity()
	if f > QueuePeekMax {
		return QueuePeekMax
	}
	return f
}

func (q *queue) isSequential() bool {
	l := q.sequentialLease()
	if l == nil {
		return false
	}
	return ulid.Time(l.Time()).After(getNow())
}

func (q *queue) isScavenger() bool {
	l := q.scavengerLease()
	if l == nil {
		return false
	}
	return ulid.Time(l.Time()).After(getNow())
}

func (q *queue) queueGauges(ctx context.Context) {
	// Report gauges to otel.
	_, _ = q.meter.Int64ObservableGauge(
		"inngest_queue_capacity_total",
		metric.WithDescription("Capacity of current worker"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			o.Observe(q.capacity())
			return nil
		}),
	)
	_, _ = q.meter.Int64ObservableGauge(
		"inngest_queue_global_partition_total_count",
		metric.WithDescription("Number of total partitions in the global queue"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			cnt, err := q.partitionSize(ctx, q.kg.GlobalPartitionIndex(), getNow().Add(time.Hour*24*365))
			if err != nil {
				q.logger.Error().Err(err).Msg("error getting global partition total for gauge")
			}
			o.Observe(cnt)
			return nil
		}),
	)
	// Report gauges to otel.
	_, _ = q.meter.Int64ObservableGauge(
		"inngest_queue_global_partition_available_count",
		metric.WithDescription("Number of available partitions in the global queue"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			cnt, err := q.partitionSize(ctx, q.kg.GlobalPartitionIndex(), getNow().Add(PartitionLookahead))
			if err != nil {
				q.logger.Error().Err(err).Msg("error getting global partition available for gauge")
			}
			o.Observe(cnt)
			return nil
		}),
	)
}

// shardGauges reports shard gauges via otel.
func (q *queue) shardGauges(ctx context.Context) {
	var (
		shards map[string]*QueueShard
		err    error
	)
	go func() {
		tick := time.NewTicker(ShardTickTime)
		for {
			select {
			case <-ctx.Done():
				tick.Stop()
				return
			case <-tick.C:
				// Reload shards.
				shards, err = q.getShards(ctx)
			}
		}
	}()

	// Report gauges to otel.
	_, _ = q.meter.Int64ObservableGauge(
		"inngest_queue_shards_count",
		metric.WithDescription("Number of shards in the queue"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			o.Observe(int64(len(shards)))
			return err
		}),
	)
	_, _ = q.meter.Int64ObservableGauge(
		"inngest_queue_shards_guaranteed_capacity_count",
		metric.WithDescription("Shard guaranteed capacity, by shard name"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			for _, shard := range shards {
				o.Observe(int64(shard.GuaranteedCapacity), metric.WithAttributes(
					attribute.KeyValue{Key: attribute.Key("shard_name"), Value: attribute.StringValue(shard.Name)},
				))
			}
			return err
		}),
	)
	_, _ = q.meter.Int64ObservableGauge(
		"inngest_queue_shards_lease_count",
		metric.WithDescription("Shard current lease count, by shard name"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			for _, shard := range shards {
				o.Observe(int64(len(shard.Leases)), metric.WithAttributes(
					attribute.KeyValue{Key: attribute.Key("shard_name"), Value: attribute.StringValue(shard.Name)},
				))
			}
			return err
		}),
	)
	_, _ = q.meter.Int64ObservableGauge(
		"inngest_queue_shard_partition_available_count",
		metric.WithDescription("The number of avaialble partitions by shard"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			for _, shard := range shards {
				cnt, err := q.partitionSize(ctx, q.kg.ShardPartitionIndex(shard.Name), getNow().Add(PartitionLookahead))
				if err != nil {
					q.logger.Error().Err(err).Msg("error getting shard partition size for gauge")
				}
				o.Observe(cnt, metric.WithAttributes(
					attribute.KeyValue{Key: attribute.Key("shard_name"), Value: attribute.StringValue(shard.Name)},
				))
			}
			return nil
		}),
	)
}

func (q *queue) int64counter(ctx context.Context, name string, value int64, attrs ...attribute.KeyValue) {
	if c, err := q.meter.Int64Counter(name); err == nil {
		c.Add(ctx, value, metric.WithAttributes(attrs...))
	}
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
