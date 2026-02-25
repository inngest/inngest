package queue

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/util"
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
	_, _ = q.tryClaimShardLease(ctx, shards)

	if q.shardLease() != nil {
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
			_, err := q.tryClaimShardLease(ctx, shards)
			if err != nil {
				q.quit <- err
				return
			}

			if q.shardLease() != nil {
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
		start := time.Now()
		maxExecutors := shard.ShardAssignmentConfig().NumExecutors
		if maxExecutors <= 0 {
			l.Debug("no executor capacity requested, skipping shard lease", "shard", shard.Name())
			continue
		}
		leaseID, err := DurationWithTags(ctx, shard.Name(), "shard_lease", q.Clock().Now(), func(ctx context.Context) (*ulid.ULID, error) {
			return shard.ShardLease(ctx, shard.Name()+"-"+q.ShardLeaseKeySuffix, ShardLeaseDuration, maxExecutors, nil)
		}, map[string]any{
			"action": "new",
		})

		if err == ErrAllShardsAlreadyLeased {
			l.Warn("could not get a shard lease", "shard", shard.Name(), "err", err, "duration", time.Since(start))
			metrics.IncrShardLeaseContentionCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"shard_group": q.runMode.ShardGroup, "queue_shard": shard.Name(), "segment": q.ShardLeaseKeySuffix}})
			continue
		}
		if err != nil {
			l.Warn("could not get a shard lease", "shard", shard.Name(), "err", err, "duration", time.Since(start))
			return false, err
		}

		// If lease has been gotten, set the primary shard and return it
		if leaseID != nil {
			q.shardLeaseLock.Lock()
			q.shardLeaseID = leaseID
			q.SetPrimaryShard(ctx, shard)
			q.shardLeaseLock.Unlock()

			metrics.GaugeActiveShardLease(ctx, 1, metrics.GaugeOpt{PkgName: pkgName, Tags: map[string]any{"shard_group": q.runMode.ShardGroup, "queue_shard": shard.Name(), "segment": q.ShardLeaseKeySuffix}})
			l.Info("claimed shard lease", "shard", shard.Name(), "group", q.runMode.ShardGroup, "leaseID", leaseID, "duration", time.Since(start))

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

// releaseShardLease attempts to release the current shard lease. This is called
// whenever we stop renewing the lease to free the slot for other workers.
func (q *queueProcessor) releaseShardLease() {
	l := logger.StdlibLogger(context.Background())

	shard := q.primaryQueueShard
	if shard == nil {
		l.Warn("could not release shard lease, no primary shard set")
		return
	}

	leaseID := q.shardLease()
	if leaseID == nil {
		l.Error("could not release shard lease, no leaseID set", "shard", shard.Name())
		return
	}

	if err := shard.ReleaseShardLease(context.Background(), shard.Name()+"-"+q.ShardLeaseKeySuffix, *leaseID); err != nil {
		l.Error("failed to release shard lease", "shard", shard.Name(), "error", err)
	} else {
		l.Debug("released shard lease", "shard", shard.Name())
	}
}

// renewShardLease continuously renews the shard lease until the context is cancelled
func (q *queueProcessor) renewShardLease(ctx context.Context) {
	l := logger.StdlibLogger(ctx)

	tick := q.Clock().NewTicker(ShardLeaseDuration / 3)
	defer tick.Stop()
	defer q.releaseShardLease()

	for {
		select {
		case <-ctx.Done():
			shardName := ""
			if shard := q.primaryQueueShard; shard != nil {
				shardName = shard.Name()
			}
			l.Debug("stopping shard lease renewal", "shard", shardName)
			return
		case <-tick.Chan():

			leaseID := q.shardLease()

			shard := q.primaryQueueShard
			if shard == nil {
				q.log.ReportError(errors.New("missing primary shard during lease renewal"), fmt.Sprintf("stopping shard lease renewal, missing primary shard during lease renewal for shard group: %s", q.runMode.ShardGroup))
				q.quit <- ErrShardLeaseNotFound
				return
			}

			if leaseID == nil {
				// Lease was lost somehow, stop renewing
				metrics.GaugeActiveShardLease(ctx, 0, metrics.GaugeOpt{PkgName: pkgName, Tags: map[string]any{"shard_group": q.runMode.ShardGroup, "queue_shard": q.primaryQueueShard.Name(), "segment": q.ShardLeaseKeySuffix}})
				l.Error("stopping shard lease renewal, shard lease lost during renewal")
				q.quit <- ErrShardLeaseNotFound
				return
			}
			l.Trace("renewing shard lease", "shard", shard.Name())
			start := time.Now()
			// Renew the lease
			newLeaseID, err := util.WithRetry(ctx, "queue.ShardLeaseRenewal", func(ctx context.Context) (*ulid.ULID, error) {
				return DurationWithTags(ctx, shard.Name(), "shard_lease", q.Clock().Now(), func(ctx context.Context) (*ulid.ULID, error) {
					return shard.ShardLease(ctx, shard.Name()+"-"+q.ShardLeaseKeySuffix, ShardLeaseDuration, shard.ShardAssignmentConfig().NumExecutors, leaseID)
				}, map[string]any{
					"action": "renew",
				})
			}, util.NewRetryConf(util.WithRetryConfRetryableErrors(ShardLeaseRenewalRetryableError)))
			if err != nil {
				metrics.GaugeActiveShardLease(ctx, 0, metrics.GaugeOpt{PkgName: pkgName, Tags: map[string]any{"shard_group": q.runMode.ShardGroup, "queue_shard": q.primaryQueueShard.Name(), "segment": q.ShardLeaseKeySuffix}})
				l.Error("stopping shard lease renewal, failed to renew shard lease", "shard", shard.Name(), "group", q.runMode.ShardGroup, "error", err, "duration", time.Since(start), "leaseID", *leaseID)
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
