package queue

import (
	"context"

	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/oklog/ulid/v2"
)

// claimSequentialLease is a process which continually runs while listening to the queue,
// attempting to claim a lease on sequential processing.  Only one worker is allowed to
// work on partitions sequentially;  this reduces contention.
func (q *queueProcessor) claimSequentialLease(ctx context.Context) {
	proc := q.PrimaryQueueShard.Processor()

	// Workers with an allowlist can never claim sequential queues.
	if len(q.AllowQueues) > 0 {
		return
	}

	// Attempt to claim the lease immediately.
	leaseID, err := proc.ConfigLease(ctx, "sequential", ConfigLeaseDuration, q.sequentialLease())
	if err != ErrConfigAlreadyLeased && err != nil {
		q.quit <- err
		return
	}

	q.seqLeaseLock.Lock()
	q.seqLeaseID = leaseID
	q.seqLeaseLock.Unlock()

	tick := q.Clock.NewTicker(ConfigLeaseDuration / 3)
	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return
		case <-tick.Chan():
			leaseID, err := proc.ConfigLease(ctx, "sequential", ConfigLeaseDuration, q.sequentialLease())
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
				metrics.IncrQueueSequentialLeaseClaimsCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.PrimaryQueueShard.Name()}})
			}
			q.seqLeaseID = leaseID
			q.seqLeaseLock.Unlock()
		}
	}
}

// sequentialLease is a helper method for concurrently reading the sequential
// lease ID.
func (q *queueProcessor) sequentialLease() *ulid.ULID {
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
func (q *queueProcessor) instrumentationLease() *ulid.ULID {
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
func (q *queueProcessor) scavengerLease() *ulid.ULID {
	q.scavengerLeaseLock.RLock()
	defer q.scavengerLeaseLock.RUnlock()
	if q.scavengerLeaseID == nil {
		return nil
	}
	copied := *q.scavengerLeaseID
	return &copied
}

func (q *queueProcessor) activeCheckerLease() *ulid.ULID {
	q.activeCheckerLeaseLock.RLock()
	defer q.activeCheckerLeaseLock.RUnlock()
	if q.activeCheckerLeaseID == nil {
		return nil
	}
	copied := *q.activeCheckerLeaseID
	return &copied
}

func (q *queueProcessor) isScavenger() bool {
	l := q.scavengerLease()
	if l == nil {
		return false
	}
	return ulid.Time(l.Time()).After(q.Clock.Now())
}

func (q *queueProcessor) isActiveChecker() bool {
	l := q.activeCheckerLease()
	if l == nil {
		return false
	}
	return ulid.Time(l.Time()).After(q.Clock.Now())
}

func (q *queueProcessor) isInstrumentator() bool {
	l := q.instrumentationLease()
	if l == nil {
		return false
	}
	return ulid.Time(l.Time()).After(q.Clock.Now())
}

func (q *queueProcessor) isSequential() bool {
	l := q.sequentialLease()
	if l == nil {
		return false
	}
	return ulid.Time(l.Time()).After(q.Clock.Now())
}
