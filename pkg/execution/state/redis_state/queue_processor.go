package redis_state

import (
	"context"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"golang.org/x/sync/semaphore"
)

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

func (q *queue) PeekAccountPartitions(
	ctx context.Context,
	accountID uuid.UUID,
	peekLimit int64,
	peekUntil time.Time,
	sequential bool,
) ([]*osqueue.QueuePartition, error) {
	partitionKey := q.RedisClient.kg.AccountPartitionIndex(accountID)

	// Peek 1s into the future to pull jobs off ahead of time, minimizing 0 latency
	partitions, err := osqueue.DurationWithTags(ctx, q.PrimaryQueueShard.Name(), "partition_peek", q.Clock.Now(), func(ctx context.Context) ([]*osqueue.QueuePartition, error) {
		return q.partitionPeek(ctx, partitionKey, sequential, peekUntil, peekLimit, &accountID)
	}, map[string]any{
		"is_global_partition_peek": false,
	})
	if err != nil {
		return nil, err
	}

	return partitions, nil
}

func (q *queue) PeekGlobalPartitions(
	ctx context.Context,
	peekLimit int64,
	peekUntil time.Time,
	sequential bool,
) ([]*osqueue.QueuePartition, error) {
	partitionKey := q.RedisClient.kg.GlobalPartitionIndex()

	// Peek 1s into the future to pull jobs off ahead of time, minimizing 0 latency
	partitions, err := osqueue.DurationWithTags(ctx, q.PrimaryQueueShard.Name(), "partition_peek", q.Clock.Now(), func(ctx context.Context) ([]*osqueue.QueuePartition, error) {
		return q.partitionPeek(ctx, partitionKey, sequential, peekUntil, peekLimit, nil)
	}, map[string]any{
		"is_global_partition_peek": true,
	})
	if err != nil {
		return nil, err
	}

	return partitions, nil
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
