package redis_state

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VividCortex/ewma"
	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/pkg/backoff"
	"github.com/inngest/inngest/pkg/execution/concurrency"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/uber-go/tally/v4"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
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
)

func init() {
	latencyAvg = ewma.NewMovingAverage()
	latencySem = &sync.Mutex{}
}

// startedAtCtxKey is a context key which records when the queue item starts,
// available via context.
type startedAtCtxKey struct{}

func GetItemStart(ctx context.Context) (time.Time, bool) {
	t, ok := ctx.Value(startedAtKey).(time.Time)
	return t, ok
}

func (q *queue) Enqueue(ctx context.Context, item osqueue.Item, at time.Time) error {
	id := ""
	if item.JobID != nil {
		id = *item.JobID
	}

	var queueName *string
	if name, ok := q.queueKindMapping[item.Kind]; ok {
		queueName = &name
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
		QueueName: queueName,
	}

	// Use the queue item's score, ensuring we process older function runs first
	// (eg. before at)
	next := time.UnixMilli(qi.Score())
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
			q.seqLeaseLock.RLock()
			if q.capacity() < minWorkersFree {
				q.seqLeaseLock.RUnlock()
				// Wait until we have more workers free.  This stops us from
				// claiming a partition to work on a single job, ensuring we
				// have capacity to run at least MinWorkersFree concurrent
				// QueueItems.  This reduces latency of enqueued items when
				// there are lots of enqueued and available jobs.
				continue
			}
			q.seqLeaseLock.RUnlock()

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

// claimSequentialLease is a process which continually runs while listening to the queue,
// attempting to claim a lease on sequential processing.  Only one worker is allowed to
// work on partitions sequentially;  this reduces contention.
func (q *queue) claimSequentialLease(ctx context.Context) {
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
			err := q.process(processCtx, i.P, i.I, f)
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

	// Peek 1s into the future to pull jobs off ahead of time, minimizing 0 latency
	partitions, err := q.PartitionPeek(ctx, q.isSequential(), time.Now().Add(time.Second), PartitionPeekMax)
	if err != nil {
		return err
	}

	for _, p := range partitions {
		if q.capacity() == 0 {
			// no longer any available workers for partition, so we can skip
			// work
			q.scope.Counter("scan_no_capacity_total").Inc(1)
			return nil
		}
		if err := q.processPartition(ctx, p); err != nil {
			if errors.Unwrap(err) != context.Canceled {
				q.logger.Error().Err(err).Msg("error processing partition")
			}
			return err
		}
	}

	return nil
}

func (q *queue) processPartition(ctx context.Context, p *QueuePartition) error {
	q.scope.Counter(counterPartitionProcess).Inc(1)
	ctx, span := q.tracer.Start(ctx, "processPartition")
	defer span.End()

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
	_, capacity, err := q.PartitionLease(ctx, *p, PartitionLeaseDuration)
	if err == ErrPartitionConcurrencyLimit {
		q.scope.Counter(counterPartitionConcurrencyLimitReached).Inc(1)
		return q.PartitionRequeue(ctx, p.Queue(), time.Now().Truncate(time.Second).Add(PartitionConcurrencyLimitRequeueExtension), true)
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
	fetch := time.Now().Truncate(time.Second).Add(2 * time.Second)
	queue, err := q.Peek(peekCtx, p.Queue(), fetch, q.peekSize())
	if err != nil {
		return err
	}

	eg := errgroup.Group{}
	for _, qi := range queue {
		item := qi
		if item.LeaseID != nil && ulid.Time(item.LeaseID.Time()).After(time.Now()) {
			q.scope.Counter(counterQueueItemsProcessLeaseExists).Inc(1)
			continue
		}

		// Check the local in-memory counters for partition-level concurrency
		// limits here.  If we're at max from this local state, we can stop processing items.
		if capacity == 0 {
			q.scope.Counter(counterConcurrencyLimit).Inc(1)
			break
		}

		// Cbeck if there's capacity from our local workers atomically prior to leasing our tiems.
		if !q.sem.TryAcquire(1) {
			q.scope.Counter(counterPartitionProcessNoCapacity).Inc(1)
			break
		}

		capacity--

		// Within a goroutine attempt to lease this item.  This lets us concurrently lease
		// items within the partition to process as fast as possible.
		eg.Go(func() error {
			// NOTE: If this ends in an error, we must _always_ release an item from the
			// semaphore to free capacity.  This will happen automatically when the worker
			// finishes processing a queue item on success.

			// Ensure there's room within the concurrency queue, first.  This is typically
			// more constrained.
			if q.concurrencyService != nil {
				err := q.concurrencyService.Add(ctx, item.WorkflowID, item.Data)
				if err == concurrency.ErrAtConcurrencyLimit {
					// return this package-specific error.
					q.sem.Release(1)
					return ErrConcurrencyLimit
				}
				if err != nil {
					q.sem.Release(1)
					return fmt.Errorf("error checking concurrency service limits: %w", err)
				}
			}

			q.scope.Counter(counterPartitionProcessItems).Inc(1)
			// Attempt to lease this item before passing this to a worker.  We have to do this
			// synchronously as we need to lease prior to requeueing the partition pointer. If
			// we don't do this here, the workers may not lease the items before calling Peek
			// to re-enqeueu the pointer, which then increases contention - as we requeue a
			// pointer too early.
			//
			// This is safe:  only one process runs scan(), and we guard the total number of
			// available workers with the above semaphore.
			leaseID, err := q.Lease(ctx, *p, *item, QueueLeaseDuration)
			if err != nil && q.concurrencyService != nil && err != ErrQueueItemAlreadyLeased {
				// NOTE: Always remove the concurrency key if leasing failed.
				//
				// There's a race condition here;  the key may not be found if
				// there's contention on the worker item.
				//
				// w1: add to concurrency
				// w1: lease
				// w2: add to concurrency
				// w1: remove concurrency key
				// w2: attempt to lease; not found error
				// w2: remove concurrency
				// w2: concurrency key not found
				doneErr := q.concurrencyService.Done(ctx, item.WorkflowID, item.Data)
				if doneErr != nil && doneErr != concurrency.ErrKeyNotFound {
					// Return both the lease error and the error for removing
					// the concurrency key.
					q.sem.Release(1)
					return multierror.Append(err, doneErr)
				}
			}

			switch err {
			case ErrPartitionConcurrencyLimit, ErrConcurrencyLimit:
				q.scope.Counter(counterConcurrencyLimit).Inc(1)
				q.sem.Release(1)
				// Since the queue is at capacity, return the error so that we
				// don't keep hammering with "does the queue have room?" logic.
				return err
			case ErrQueueItemNotFound:
				q.scope.Counter(counterQueueItemsGone).Inc(1)
				q.sem.Release(1)
				return nil
			case ErrQueueItemAlreadyLeased:
				q.scope.Counter(counterQueueItemsLeaseConflict).Inc(1)
				q.sem.Release(1)
				q.logger.
					Warn().
					Interface("item", item).
					Msg("worker attempting to claim existing lease")
				return nil
			}
			if err != nil {
				q.sem.Release(1)
				return fmt.Errorf("error leasing in process: %w", err)
			}

			// Assign the lease ID and pass this to be handled by the available worker.
			// There should always be capacity on this queue as we track capacity via
			// a semaphore.
			item.LeaseID = leaseID
			q.workers <- processItem{P: *p, I: *item}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		// The lease for the partition will expire and we will be able to restart
		// work in the future.
		switch err {
		case ErrPartitionConcurrencyLimit, ErrConcurrencyLimit:
			// Requeue this partition as we hit concurrency limits.
			q.scope.Counter(counterConcurrencyLimit).Inc(1)
			return q.PartitionRequeue(ctx, p.Queue(), time.Now().Truncate(time.Second).Add(PartitionConcurrencyLimitRequeueExtension), true)
		default:
			return err
		}
	}

	// XXX: If we haven't been able to lease a single item, ensure we enqueue this
	// for a minimum of 5 seconds.

	// Requeue the partition, which reads the next unleased job or sets a time of
	// 30 seconds.  This is why we have to lease items above, else this may return an item that is
	// about to be leased and processed by the worker.
	err = q.PartitionRequeue(ctx, p.Queue(), time.Now().Add(PartitionRequeueExtension), false)
	if err == ErrPartitionGarbageCollected {
		// Safe;  we're preventing this from wasting cycles in the future.
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

func (q *queue) process(ctx context.Context, p QueuePartition, qi QueueItem, f osqueue.RunFunc) error {
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
				errCh <- osqueue.AlwaysRetry(fmt.Errorf("job panicked: %v", r))
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

		jobCtx = context.WithValue(jobCtx, startedAtKey, time.Now())

		// Track the latency on average globally.  Do this in a goroutine so that it doesn't
		// at all delay the job during concurrenty locking contention.
		latency := time.Since(time.UnixMilli(qi.AtMS))
		go func() {
			// Update the ewma
			latencySem.Lock()
			latencyAvg.Add(float64(latency))
			scope.Gauge(gaugeQueueItemLatencyEWMA).Update(latencyAvg.Value() / 1e6)
			latencySem.Unlock()

			// Set the metrics historgram and gauge, which reports the ewma value.
			scope.Histogram(histogramItemLatency, latencyBuckets).RecordDuration(latency)

			q.logger.Debug().
				Str("job_id", qi.ID).
				Int64("latency_ms", latency.Milliseconds()).
				Msg("processing job")
		}()

		go scope.Counter(counterQueueItemsStarted).Inc(1)
		err := f(jobCtx, qi.Data)
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
			at := backoff.DefaultBackoff(qi.Data.Attempt)
			qi.Data.Attempt += 1
			qi.AtMS = at.UnixMilli()
			q.logger.Warn().Err(err).
				Str("queue", qi.Queue()).
				Int64("at_ms", at.UnixMilli()).
				Msg("requeuing job")
			if err := q.Requeue(ctx, p, qi, at); err != nil {
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
		if err := q.Dequeue(ctx, p, qi); err != nil {
			return err
		}

		if _, ok := err.(osqueue.QuitError); ok {
			q.quit <- err
			return err
		}

	case <-doneCh:
		if err := q.Dequeue(ctx, p, qi); err != nil {
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
	return ulid.Time(l.Time()).After(time.Now())
}

func (q *queue) isScavenger() bool {
	l := q.scavengerLease()
	if l == nil {
		return false
	}
	return ulid.Time(l.Time()).After(time.Now())
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
