package queue

import (
	"context"
	"math/rand"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

func (q *queueProcessor) runScavenger(ctx context.Context) {
	// Attempt to claim the lease immediately.
	leaseID, err := q.primaryQueueShard.ConfigLease(ctx, "scavenger", ConfigLeaseDuration, q.scavengerLease())
	if err != ErrConfigAlreadyLeased && err != nil {
		q.quit <- err
		return
	}

	q.scavengerLeaseLock.Lock()
	q.scavengerLeaseID = leaseID // no-op if not leased
	q.scavengerLeaseLock.Unlock()

	tick := q.Clock().NewTicker(ConfigLeaseDuration / 3)
	scavenge := q.Clock().NewTicker(30 * time.Second)

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			scavenge.Stop()
			return
		case <-scavenge.Chan():
			// Scavenge the items
			if q.isScavenger() {
				count, err := q.primaryQueueShard.Scavenge(ctx, ScavengePeekSize)
				if err != nil {
					logger.StdlibLogger(ctx).Error("error scavenging", "error", err)
				}
				if count > 0 {
					logger.StdlibLogger(ctx).Info("scavenged lost jobs", "len", count)
				}
			}
		case <-tick.Chan():
			// Attempt to re-lease the lock.
			leaseID, err := q.primaryQueueShard.ConfigLease(ctx, "scavenger", ConfigLeaseDuration, q.scavengerLease())
			if err == ErrConfigAlreadyLeased {
				// This is expected; every time there is > 1 runner listening to the
				// queue there will be contention.
				q.scavengerLeaseLock.Lock()
				q.scavengerLeaseID = nil
				q.scavengerLeaseLock.Unlock()
				continue
			}
			if err != nil {
				logger.StdlibLogger(ctx).Error("error claiming scavenger lease", "error", err)
				q.scavengerLeaseLock.Lock()
				q.scavengerLeaseID = nil
				q.scavengerLeaseLock.Unlock()
				continue
			}

			q.scavengerLeaseLock.Lock()
			if q.scavengerLeaseID == nil {
				// Only track this if we're creating a new lease, not if we're renewing
				// a lease.
				metrics.IncrQueueSequentialLeaseClaimsCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"queue_shard": q.primaryQueueShard.Name()}})
			}
			q.scavengerLeaseID = leaseID
			q.scavengerLeaseLock.Unlock()
		}
	}
}

func RandomScavengeOffset(seed int64, count int64, limit int) int64 {
	// only apply random offset if there are more total items to scavenge than the limit
	if count > int64(limit) {
		r := rand.New(rand.NewSource(seed))

		// the result of count-limit must be greater than 0 as we have already checked count > limit
		// we increase the argument by 1 to make the highest possible index accessible
		// example: for count = 9, limit = 3, we want to access indices 0 through 6, not 0 through 5
		return r.Int63n(count - int64(limit) + 1)
	}

	return 0
}
