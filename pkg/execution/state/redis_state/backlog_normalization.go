package redis_state

import (
	"context"
	"crypto/rand"
	"fmt"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/oklog/ulid/v2"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"math"
	"time"

	"github.com/inngest/inngest/pkg/logger"
)

// backlogNormalizationWorker runs a blocking process that listens to item being pushed into the normalization partition. This allows us to process individual
// backlogs that need to be normalized
func (q *queue) backlogNormalizationWorker(ctx context.Context, nc chan *QueueBacklog) {
	l := logger.StdlibLogger(ctx)

	for {
		select {
		case <-ctx.Done():
			return

		case backlog := <-nc:
			err := q.normalizeBacklog(ctx, backlog)
			if err != nil {
				l.Error("could not normalize backlog", "error", err, "backlog", backlog)
			}
		}
	}
}

// backlogNormalizationScan iterates through a partition of backlogs and reenqueue
// the items to the appropriate backlogs
func (q *queue) backlogNormalizationScan(ctx context.Context) error {
	l := logger.StdlibLogger(ctx)
	bc := make(chan *QueueBacklog)

	for i := int32(0); i < q.numBacklogNormalizationWorkers; i++ {
		go q.backlogNormalizationWorker(ctx, bc)
	}

	tick := q.clock.NewTicker(q.pollTick)
	l.Debug("starting normalization scanner", "poll", q.pollTick.String())

	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return nil

		case <-tick.Chan():
			until := q.clock.Now()

			if err := q.iterateNormalizationPartition(ctx, until, bc); err != nil {
				// TODO: check errors

				l.Error("error scanning global normalization partition", "error", err)

				// TODO: return if error is not acceptable
			}
		}
	}
}

// iterateNormalizationPartition scans and iterate through the global normalization partition to process backlogs needing to be normalized
func (q *queue) iterateNormalizationPartition(ctx context.Context, until time.Time, bc chan *QueueBacklog) error {
	l := logger.StdlibLogger(ctx)

	// TODO: check capacity

	// TODO introduce weight probability to blend account/global scanning
	peekedAccounts, err := q.peekGlobalNormalizeAccounts(ctx, until, NormalizeAccountPeekMax)
	if err != nil {
		return fmt.Errorf("could not peek global normalize accounts: %w", err)
	}

	if len(peekedAccounts) == 0 {
		return nil
	}

	// Reduce number of peeked partitions as we're processing multiple accounts in parallel
	// Note: This is not optimal as some accounts may have fewer partitions than others and
	// we're leaving capacity on the table. We'll need to find a better way to determine the
	// optimal peek size in this case.
	accountShadowPartitionPeekMax := int64(math.Round(float64(ShadowPartitionPeekMax / int64(len(peekedAccounts)))))

	// Scan and process account shadow partitions in parallel
	eg := errgroup.Group{}
	for _, account := range peekedAccounts {
		partitionKey := q.primaryQueueShard.RedisClient.kg.AccountNormalizeSet(account)

		eg.Go(func() error {
			return q.iterateNormalizationShadowPartition(ctx, partitionKey, accountShadowPartitionPeekMax, until, bc, l)
		})
	}

	err = eg.Wait()
	if err != nil {
		return fmt.Errorf("failed to scan and normalize backlogs for accounts: %w", err)
	}

	// TODO: counter metric for scanned backlogs in normalization partition

	return nil
}

func (q *queue) iterateNormalizationShadowPartition(ctx context.Context, shadowPartitionIndexKey string, peekLimit int64, until time.Time, bc chan *QueueBacklog, l *slog.Logger) error {
	// Find partitions in account or globally with backlogs to normalize
	shadowPartitions, err := q.peekShadowPartitions(ctx, shadowPartitionIndexKey, peekLimit, until)
	if err != nil {
		return fmt.Errorf("could not peek shadow partitions to normalize: %w", err)
	}

	// For each partition, attempt to normalize backlogs
	for _, partition := range shadowPartitions {
		backlogs, err := duration(ctx, q.primaryQueueShard.Name, "normalize_peek", until, func(ctx context.Context) ([]*QueueBacklog, error) {
			return q.ShadowPartitionPeekNormalizeBacklogs(ctx, partition, NormalizePartitionPeekMax)
		})
		if err != nil {
			return err
		}

		for _, bl := range backlogs {
			// lease the backlog
			if err := q.leaseBacklogForNormalization(ctx, bl); err != nil {
				l.Error("error leasing backlog for normalization", "error", err, "backlog", bl)
				continue
			}

			// dump it into the channel for the workers to do their thing
			bc <- bl
		}
	}

	return nil
}

func (q *queue) leaseBacklogForNormalization(ctx context.Context, bl *QueueBacklog) error {
	leaseExpiry := q.clock.Now().Add(BacklogNormalizeLeaseDuration)
	leaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return fmt.Errorf("could not generate leaseID: %w", err)
	}

	// TODO Run script

	bl.NormalizationLease = &leaseID

	return fmt.Errorf("not implemented")
}

var (
	errBacklogNormalizationLeaseExpired     = fmt.Errorf("backlog normalization lease expired")
	errBacklogAlreadyLeasedForNormalization = fmt.Errorf("backlog already leased for normalization")
)

func (q *queue) extendBacklogNormalizationLease(ctx context.Context, now time.Time, bl *QueueBacklog) error {
	leaseExpiry := now.Add(BacklogNormalizeLeaseDuration)
	newLeaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	if err != nil {
		return fmt.Errorf("could not generate newLeaseID: %w", err)
	}

	if bl.NormalizationLease == nil || ulid.Time(bl.NormalizationLease.Time()).Before(now) {
		return errBacklogNormalizationLeaseExpired
	}

	// TODO Run script

	bl.NormalizationLease = &newLeaseID

	return nil
}

// normalizeBacklog must be called with exclusive access to the shadow partition
// NOTE: ideally this is one transaction in a lua script but enqueue_to_backlog is way too much work to
// utilize
func (q *queue) normalizeBacklog(ctx context.Context, backlog *QueueBacklog) error {
	l := logger.StdlibLogger(ctx).With("backlog", backlog)

	// TODO: extend the lease

	shard := q.primaryQueueShard
	var processed int64
	for {
		res, err := q.BacklogNormalizePeek(ctx, backlog, NormalizePartitionPeekMax)
		if err != nil {
			return fmt.Errorf("could not peek backlog items for normalization: %w", err)
		}

		for _, item := range res.Items {
			if _, err := q.EnqueueItem(ctx, shard, *item, time.UnixMilli(item.AtMS), osqueue.EnqueueOpts{
				PassthroughJobId:       true,
				NormalizeFromBacklogID: backlog.BacklogID,
			}); err != nil {
				return fmt.Errorf("could not re-enqueue backlog item: %w", err)
			}

			processed += 1
		}

		l.Info("processed normalization for backlog",
			"processed", processed,
			"removed", res.RemovedCount,
		)
	}
}
