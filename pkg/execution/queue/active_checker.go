package queue

import (
	"context"

	"github.com/inngest/inngest/pkg/logger"
)

func (q *queueProcessor) runActiveChecker(ctx context.Context) {
	// Attempt to claim the lease immediately.
	leaseID, err := q.primaryQueueShard.ConfigLease(ctx, "active-checker", ConfigLeaseDuration, q.activeCheckerLease())
	if err != ErrConfigAlreadyLeased && err != nil {
		q.quit <- err
		return
	}

	q.activeCheckerLeaseLock.Lock()
	q.activeCheckerLeaseID = leaseID // no-op if not leased
	q.activeCheckerLeaseLock.Unlock()

	tick := q.Clock().NewTicker(ConfigLeaseDuration / 3)
	checkTick := q.Clock().NewTicker(q.ActiveCheckTick)

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			checkTick.Stop()
			return
		case <-checkTick.Chan():
			// Active check backlogs
			if q.isActiveChecker() {
				count, err := q.primaryQueueShard.ActiveCheck(ctx)
				if err != nil {
					logger.StdlibLogger(ctx).Error("error checking active jobs", "error", err)
				}
				if count > 0 {
					logger.StdlibLogger(ctx).Trace("checked active jobs", "len", count)
				}
			}
		case <-tick.Chan():
			// Attempt to re-lease the lock.
			leaseID, err := q.primaryQueueShard.ConfigLease(ctx, "active-checker", ConfigLeaseDuration, q.activeCheckerLease())
			if err == ErrConfigAlreadyLeased {
				// This is expected; every time there is > 1 runner listening to the
				// queue there will be contention.
				q.activeCheckerLeaseLock.Lock()
				q.activeCheckerLeaseID = nil
				q.activeCheckerLeaseLock.Unlock()
				continue
			}
			if err != nil {
				logger.StdlibLogger(ctx).Error("error claiming active checker lease", "error", err)
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
