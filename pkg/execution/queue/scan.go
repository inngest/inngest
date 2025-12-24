package queue

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"golang.org/x/sync/errgroup"
)

func (q *queueProcessor) executionScan(ctx context.Context, f RunFunc) error {
	l := q.log.With(
		"queue_shard", q.PrimaryQueueShard.Name(),
	)

	for i := int32(0); i < q.numWorkers; i++ {
		go q.worker(ctx, f)
	}

	if !q.runMode.Partition && !q.runMode.Account {
		return fmt.Errorf("need to specify either partition, account, or both in queue run mode")
	}

	tick := q.Clock.NewTicker(q.pollTick)
	l.Debug("starting queue worker", "poll", q.pollTick.String())

	backoff := time.Millisecond * 250

	var err error
LOOP:
	for {
		select {
		case <-ctx.Done():
			// Kill signal
			tick.Stop()
			break LOOP
		case err = <-q.quit:
			// An inner function received an error which was deemed irrecoverable, so
			// we're quitting the queue.
			q.log.ReportError(err, "quitting runner internally")
			tick.Stop()
			break LOOP

		case <-tick.Chan():
			if q.capacity() < minWorkersFree {
				// Wait until we have more workers free.  This stops us from
				// claiming a partition to work on a single job, ensuring we
				// have capacity to run at least MinWorkersFree concurrent
				// QueueItems.  This reduces latency of enqueued items when
				// there are lots of enqueued and available jobs.
				l.Trace("all workers busy, early exiting scan", "worker_capacity", q.capacity())
				continue
			}

			if err = q.scan(ctx); err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					q.log.Warn("deadline exceeded scanning partition pointers")
					<-time.After(backoff)

					// Backoff doubles up to 3 seconds.
					backoff = time.Duration(math.Min(float64(backoff*2), float64(time.Second*5)))
					continue
				}

				// On scan errors, halt the worker entirely.
				if !errors.Is(err, context.Canceled) {
					q.log.ReportError(err, "error scanning partition pointers")
				}
				break LOOP
			}

			backoff = time.Millisecond * 250
		}
	}

	// Wait for all in-progress items to complete.
	q.log.Info("queue waiting to quit")
	q.wg.Wait()

	return err
}

func (q *queueProcessor) scan(ctx context.Context) error {
	if q.capacity() == 0 {
		return nil
	}

	// If there are continuations, process those immediately.
	if err := q.scanContinuations(ctx); err != nil {
		return fmt.Errorf("error scanning continuations: %w", err)
	}

	// Store the shard that we processed, allowing us to eventually pass this
	// down to the job for stat tracking.
	metricShardName := "<global>" // default global name for metrics in this function

	peekUntil := q.Clock.Now().Add(PartitionLookahead)

	processAccount := false
	if q.runMode.Account && (!q.runMode.Partition || rand.Intn(100) <= q.runMode.AccountWeight) {
		processAccount = true
	}

	if len(q.runMode.ExclusiveAccounts) > 0 {
		processAccount = true
	}

	if processAccount {
		metrics.IncrQueueScanCounter(ctx,
			metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"kind":        "accounts",
					"queue_shard": q.PrimaryQueueShard.Name(),
				},
			},
		)

		var peekedAccounts []uuid.UUID
		if len(q.runMode.ExclusiveAccounts) > 0 {
			peekedAccounts = q.runMode.ExclusiveAccounts
		} else {
			peeked, err := Duration(ctx, q.PrimaryQueueShard.Name(), "account_peek", q.Clock.Now(), func(ctx context.Context) ([]uuid.UUID, error) {
				return q.PrimaryQueueShard.AccountPeek(ctx, q.isSequential(), peekUntil, AccountPeekMax)
			})
			if err != nil {
				return fmt.Errorf("could not peek accounts: %w", err)
			}
			peekedAccounts = peeked
		}

		if len(peekedAccounts) == 0 {
			q.log.Trace("account_peek yielded no accounts")
			return nil
		}

		// Reduce number of peeked partitions as we're processing multiple accounts in parallel
		// Note: This is not optimal as some accounts may have fewer partitions than others and
		// we're leaving capacity on the table. We'll need to find a better way to determine the
		// optimal peek size in this case.
		accountPartitionPeekMax := int64(math.Round(float64(PartitionPeekMax / int64(len(peekedAccounts)))))

		var actualScannedPartitions int64

		// Scan and process account partitions in parallel
		wg := sync.WaitGroup{}
		for _, account := range peekedAccounts {
			account := account

			wg.Add(1)
			go func(account uuid.UUID) {
				defer wg.Done()
				if err := q.ScanAccountPartitions(ctx, account, accountPartitionPeekMax, peekUntil, metricShardName, &actualScannedPartitions); err != nil {
					q.log.Error("error processing account partitions", "error", err)
				}
			}(account)
		}

		wg.Wait()

		metrics.IncrQueuePartitionScannedCounter(ctx,
			actualScannedPartitions,
			metrics.CounterOpt{
				PkgName: pkgName,
				Tags: map[string]any{
					"kind":        "accounts",
					"queue_shard": q.PrimaryQueueShard.Name(),
				},
			},
		)

		return nil
	}

	metrics.IncrQueueScanCounter(ctx,
		metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"kind":        "partitions",
				"queue_shard": q.PrimaryQueueShard.Name(),
			},
		},
	)

	var actualScannedPartitions int64
	err := q.ScanGlobalPartitions(ctx, PartitionPeekMax, peekUntil, metricShardName, &actualScannedPartitions)
	if err != nil {
		return fmt.Errorf("error scanning partition: %w", err)
	}

	metrics.IncrQueuePartitionScannedCounter(ctx,
		actualScannedPartitions,
		metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"kind":        "partitions",
				"queue_shard": q.PrimaryQueueShard.Name(),
			},
		},
	)

	return nil
}

func (q *queueProcessor) ScanAccountPartitions(ctx context.Context, accountID uuid.UUID, peekLimit int64, peekUntil time.Time, metricShardName string, reportPeekedPartitions *int64) error {
	partitions, err := q.PrimaryQueueShard.PeekAccountPartitions(ctx, accountID, peekLimit, peekUntil, q.isSequential())
	if err != nil {
		return fmt.Errorf("could not peek account partitions: %w", err)
	}

	return q.processScannedPartitions(ctx, partitions, peekUntil, metricShardName, reportPeekedPartitions)
}

func (q *queueProcessor) ScanGlobalPartitions(ctx context.Context, peekLimit int64, peekUntil time.Time, metricShardName string, reportPeekedPartitions *int64) error {
	partitions, err := q.PrimaryQueueShard.PeekGlobalPartitions(ctx, peekLimit, peekUntil, q.isSequential())
	if err != nil {
		return fmt.Errorf("could not peek global partitions: %w", err)
	}

	return q.processScannedPartitions(ctx, partitions, peekUntil, metricShardName, reportPeekedPartitions)
}

func (q *queueProcessor) processScannedPartitions(
	ctx context.Context,
	partitions []*QueuePartition,
	peekUntil time.Time,
	metricShardName string,
	reportPeekedPartitions *int64,
) error {
	if reportPeekedPartitions != nil {
		atomic.AddInt64(reportPeekedPartitions, int64(len(partitions)))
	}

	q.log.Trace("processing partitions",
		"peek_until", peekUntil.Format(time.StampMilli),
		"partition", len(partitions),
	)

	eg := errgroup.Group{}

	for _, ptr := range partitions {
		p := *ptr
		eg.Go(func() error {
			if q.capacity() == 0 {
				// no longer any available workers for partition, so we can skip
				// work
				metrics.IncrQueueScanNoCapacityCounter(ctx, metrics.CounterOpt{PkgName: pkgName, Tags: map[string]any{"shard": metricShardName, "queue_shard": q.PrimaryQueueShard.Name()}})
				return nil
			}
			if err := q.processPartition(ctx, &p, 0, false); err != nil {
				if err == ErrPartitionNotFound || err == ErrPartitionGarbageCollected {
					// Another worker grabbed the partition, or the partition was deleted
					// during the scan by an another worker.
					// TODO: Increase internal metrics
					return nil
				}
				if !errors.Is(err, context.Canceled) {
					q.log.Error("error processing partition", "error", err)
				}
				return err
			}

			metrics.IncrQueuePartitionProcessedCounter(ctx, metrics.CounterOpt{
				PkgName: pkgName,
				Tags:    map[string]any{"shard": metricShardName, "queue_shard": q.PrimaryQueueShard.Name()},
			})
			return nil
		})
	}

	return eg.Wait()
}
