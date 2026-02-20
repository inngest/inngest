package queue

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
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
	l := logger.StdlibLogger(ctx)
	shardGroup := q.runMode.ShardGroup
	if len(shardGroup) == 0 {
		return
	}
	shards := q.shardsByGroupName(shardGroup)
	if len(shards) == 0 {
		l.Error("no shards found for group", "group", shardGroup)
		q.quit <- ErrQueueShardNotFound
		return
	}

	// Attempt to claim the lease immediately before waiting for the ticker.
	claimed, err := q.tryClaimShardLease(ctx, shards)
	if err != nil {
		q.quit <- err
		return
	}
	if claimed {
		return
	}

	tick := q.Clock().NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return
		case <-tick.Chan():
			// Attempt to claim the lease.
			claimed, err := q.tryClaimShardLease(ctx, shards)
			if err != nil {
				q.quit <- err
				return
			}

			if claimed {
				tick.Stop()
				return
			}
		}
	}

}

// tryClaimShardLease attempts to claim a lease on one of the shards in the pool.
func (q *queueProcessor) tryClaimShardLease(ctx context.Context, shards []QueueShard) (bool, error) {
	l := logger.StdlibLogger(ctx)

	// if a shard was already leased, early exit.
	if q.shardLease() != nil {
		l.Warn("Calling tryClaimShardLease when already leased")
		return false, nil
	}

	// Randomize shards to minimize contention
	rand.Shuffle(len(shards), func(i, j int) {
		shards[i], shards[j] = shards[j], shards[i]
	})

	// Try to get a lease on one of them
	for _, shard := range shards {
		maxExecutors := shard.ShardAssignmentConfig().NumExecutors
		if maxExecutors <= 0 {
			l.Debug("no executor capacity requested, skipping shard lease", "shard", shard.Name())
			continue
		}
		leaseID, err := Duration(ctx, shard.Name(), "shard_lease", q.Clock().Now(), func(ctx context.Context) (*ulid.ULID, error) {
			return shard.ShardLease(ctx, shard.Name()+"-"+q.ShardLeaseKeySuffix, ShardLeaseDuration, maxExecutors, nil)
		})

		if err == ErrAllShardsAlreadyLeased {
			l.Warn("Could not get a shard lease", "shard", shard.Name())
			metrics.IncrShardLeaseContentionCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"shard_group": q.runMode.ShardGroup, "queue_shard": shard.Name(), "segment": q.ShardLeaseKeySuffix}})
			continue
		}
		if err != nil {
			return false, err
		}

		// If lease has been gotten, set the primary shard and return it
		if leaseID != nil {
			q.shardLeaseLock.Lock()
			q.shardLeaseID = leaseID
			q.SetPrimaryShard(ctx, shard)
			q.shardLeaseLock.Unlock()

			metrics.GaugeActiveShardLease(ctx, 1, metrics.GaugeOpt{PkgName: pkgName, Tags: map[string]any{"shard_group": q.runMode.ShardGroup, "queue_shard": shard.Name(), "segment": q.ShardLeaseKeySuffix}})
			l.Info("claimed shard lease", "shard", shard.Name(), "group", q.runMode.ShardGroup, "leaseID", leaseID)

			// Renew the lease indefinitely in a separate goroutine
			go q.renewShardLease(ctx)

			if q.OnShardLeaseAcquired != nil {
				go q.OnShardLeaseAcquired(ctx, shard.Name())
			}
			return true, nil
		}
	}

	// If we couldn't get a lease on any shard, return false
	return false, nil
}

// renewShardLease continuously renews the shard lease until the context is cancelled
func (q *queueProcessor) renewShardLease(ctx context.Context) {
	l := logger.StdlibLogger(ctx)

	tick := q.Clock().NewTicker(ShardLeaseDuration / 3)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			l.Debug("Stopping Shard Lease Renewal")
			return
		case <-tick.Chan():
			l.Trace("Renewing Shard Lease")

			leaseID := q.shardLease()

			shard := q.primaryQueueShard
			if shard == nil {
				q.log.ReportError(errors.New("missing primary shard during lease renewal"), fmt.Sprintf("missing primary shard during lease renewal for shard group: %s", q.runMode.ShardGroup))
				q.quit <- ErrShardLeaseNotFound
				return
			}

			if leaseID == nil {
				// Lease was lost somehow, stop renewing
				metrics.GaugeActiveShardLease(ctx, 0, metrics.GaugeOpt{PkgName: pkgName, Tags: map[string]any{"shard_group": q.runMode.ShardGroup, "queue_shard": q.primaryQueueShard.Name(), "segment": q.ShardLeaseKeySuffix}})
				l.Error("shard lease lost during renewal")
				q.quit <- ErrShardLeaseNotFound
				return
			}

			// Renew the lease
			newLeaseID, err := shard.ShardLease(ctx, shard.Name()+"-"+q.ShardLeaseKeySuffix, ShardLeaseDuration, shard.ShardAssignmentConfig().NumExecutors, leaseID)
			if err == ErrShardLeaseExpired || err == ErrShardLeaseNotFound {
				// Another process took the lease
				metrics.GaugeActiveShardLease(ctx, 0, metrics.GaugeOpt{PkgName: pkgName, Tags: map[string]any{"shard_group": q.runMode.ShardGroup, "queue_shard": q.primaryQueueShard.Name(), "segment": q.ShardLeaseKeySuffix}})
				l.Error("failed to renew shard lease", "shard", shard.Name(), "group", q.runMode.ShardGroup, "error", err)
				q.quit <- err
				return
			}
			if err != nil {
				metrics.GaugeActiveShardLease(ctx, 0, metrics.GaugeOpt{PkgName: pkgName, Tags: map[string]any{"shard_group": q.runMode.ShardGroup, "queue_shard": q.primaryQueueShard.Name(), "segment": q.ShardLeaseKeySuffix}})
				l.Error("failed to renew shard lease", "error", err, "shard", shard.Name(), "group", q.runMode.ShardGroup, "leaseID", leaseID)
				q.quit <- err
				return
			}

			// Update the lease ID
			if *newLeaseID != *leaseID {
				q.shardLeaseLock.Lock()
				q.shardLeaseID = newLeaseID
				q.shardLeaseLock.Unlock()
				l.Trace("Successfully renewed lease", "old_lease", leaseID, "new_lease", newLeaseID)
			}
		}
	}
}
