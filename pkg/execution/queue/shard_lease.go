package queue

import (
	"context"
	"math/rand"

	"github.com/oklog/ulid/v2"
)

// shardLease is a helper method for concurrently reading the shard
// lease ID.
func (q *queueProcessor) shardLease() *ulid.ULID {
	q.shardLeaseLock.RLock()
	defer q.shardLeaseLock.RUnlock()
	if q.shardLeaseID == nil {
		return nil
	}
	copied := *q.shardLeaseID
	return &copied
}

// claimShardLease is a process which continually runs while listening to the queue,
// attempting to claim a lease on a shard from the pool. This is a blocking call that
// only returns when a successful shard lease has been assigned or on error.
func (q *queueProcessor) claimShardLease(ctx context.Context) {

	shardGroup := q.runMode.ShardGroup
	if len(shardGroup) == 0 {
		return
	}
	shards := q.shardsByGroupName(shardGroup)
	if len(shards) == 0 {
		q.log.Error("no shards found for group", "group", shardGroup)
		q.quit <- ErrQueueShardNotFound
		return
	}

	tick := q.Clock().NewTicker(ShardLeaseDuration / 3)
	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return
		case <-tick.Chan():
			// Attempt to claim the lease.
			err := q.tryClaimShardLease(ctx, shards)
			if err != nil {
				q.quit <- err
				return
			}

			if q.shardLease() != nil {
				tick.Stop()

				// After getting the lease, renew it indefinitely in a separate goroutine
				go q.renewShardLease(ctx)
				return
			}
		}
	}

}

// tryClaimShardLease attempts to claim a lease on one of the shards in the pool.
func (q *queueProcessor) tryClaimShardLease(ctx context.Context, shards []QueueShard) error {

	// if a shard was already leased, early exit.
	if q.shardLease() != nil {
		q.log.Warn("Calling tryClaimShardLease when already leased")
		return nil
	}

	// Randomize shards to minimize contention
	rand.Shuffle(len(shards), func(i, j int) {
		shards[i], shards[j] = shards[j], shards[i]
	})

	// Try to get a lease on one of them
	for _, shard := range shards {
		leaseID, err := shard.ShardLease(ctx, "shard-group-"+q.runMode.ShardGroup, ShardLeaseDuration, shard.ShardAssignmentConfig().NumExecutors, nil)

		if err == ErrAllShardsAlreadyLeased {
			q.log.Warn("Could not get a shard lease", "shard", shard.Name())
			continue
		}
		if err != nil {
			return err
		}

		// If lease has been gotten, set the primary shard and return it
		if leaseID != nil {
			q.shardLeaseLock.Lock()
			q.shardLeaseID = leaseID
			q.primaryQueueShard = shard
			q.shardLeaseLock.Unlock()
			q.log.Info("claimed shard lease", "shard", shard.Name(), "group", q.runMode.ShardGroup)
			return nil
		}
	}

	// If we couldn't get a lease on any shard, return nil
	return nil
}

// renewShardLease continuously renews the shard lease until the context is cancelled
func (q *queueProcessor) renewShardLease(ctx context.Context) {
	tick := q.Clock().NewTicker(ConfigLeaseDuration / 3)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.Chan():
			q.log.Trace("Renewing Shard Lease")

			leaseID := q.shardLease()
			if leaseID == nil {
				// Lease was lost somehow, stop renewing
				q.log.Error("shard lease lost during renewal")
				q.quit <- ErrShardLeaseNotFound
				return
			}

			shard := q.primaryQueueShard
			if shard == nil {
				q.log.Error("primary shard not set during lease renewal")
				q.quit <- ErrShardLeaseNotFound
				return
			}

			// Renew the lease
			newLeaseID, err := shard.ShardLease(ctx, "shard-group-"+q.runMode.ShardGroup, ShardLeaseDuration, shard.ShardAssignmentConfig().NumExecutors, leaseID)
			if err == ErrShardLeaseExpired || err == ErrShardLeaseNotFound {
				// Another process took the lease
				q.log.Error("shard lease taken by another process", "shard", shard.Name(), "group", q.runMode.ShardGroup)
				q.quit <- err
				return
			}
			if err != nil {
				q.log.Error("failed to renew shard lease", "error", err, "shard", shard.Name(), "group", q.runMode.ShardGroup)
				q.quit <- err
				return
			}

			// Update the lease ID
			if newLeaseID != leaseID {
				q.shardLeaseLock.Lock()
				q.shardLeaseID = newLeaseID
				q.shardLeaseLock.Unlock()
			}
		}
	}
}
