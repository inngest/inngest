package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/redis/rueidis"
)

// Scavenge attempts to find jobs that may have been lost due to killed workers.  Workers are shared
// nothing, and each item in a queue has a lease.  If a worker dies, it will not finish the job and
// cannot renew the item's lease.
//
// We scan all partition concurrency queues - queues of leases - to find leases that have expired.
func (q *queue) Scavenge(ctx context.Context, limit int) (int, error) {
	l := logger.StdlibLogger(ctx)

	client := q.RedisClient.unshardedRc
	kg := q.RedisClient.KeyGenerator()

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "Scavenge"), redis_telemetry.ScopeQueue)

	// Find all items that have an expired lease - eg. where the min time for a lease is between
	// (0-now] in unix milliseconds.
	now := fmt.Sprintf("%d", q.Clock.Now().UnixMilli())

	count, err := client.Do(ctx, client.B().Zcount().Key(kg.ConcurrencyIndex()).Min("-inf").Max(now).Build()).AsInt64()
	if err != nil {
		return 0, fmt.Errorf("error counting concurrency index: %w", err)
	}

	cmd := client.B().Zrange().
		Key(kg.ConcurrencyIndex()).
		Min("-inf").
		Max(now).
		Byscore().
		Limit(osqueue.RandomScavengeOffset(q.Clock.Now().UnixMilli(), count, limit), int64(limit)).
		Build()

	// NOTE: Received keys can be legacy (workflow IDs or system/internal queue names) or new (full Redis keys)
	pKeys, err := client.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return 0, fmt.Errorf("error scavenging for lost items: %w", err)
	}

	counter := 0

	// Each of the items is a concurrency queue with lost items.
	var resultErr error
	for _, partitionID := range pKeys {
		scavengePartition := func(queueKey string, kind string) (int, int, error) {
			start := q.Clock.Now()
			defer func() {
				dur := q.Clock.Now().Sub(start)
				metrics.HistogramQueueScavengerPartitionScavengeDuration(ctx, time.Duration(dur.Milliseconds()), metrics.HistogramOpt{
					PkgName: pkgName,
					Tags: map[string]any{
						"kind": kind,
					},
				})
			}()

			cmd := client.B().Zrange().
				Key(queueKey).
				Min("-inf").
				Max(now).
				Byscore().
				Limit(0, osqueue.ScavengeConcurrencyQueuePeekSize).
				Build()
			itemIDs, err := client.Do(ctx, cmd).AsStrSlice()
			if err != nil && err != rueidis.Nil {
				return 0, 0, fmt.Errorf("error querying partition concurrency queue '%s' during scavenge: %w", partitionID, err)
			}
			if len(itemIDs) == 0 {
				return 0, 0, nil
			}

			// Fetch the queue item, then requeue.
			cmd = client.B().Hmget().Key(kg.QueueItem()).Field(itemIDs...).Build()
			jobs, err := client.Do(ctx, cmd).AsStrSlice()
			if err != nil && err != rueidis.Nil {
				return 0, 0, fmt.Errorf("error fetching jobs for concurrency queue '%s' during scavenge: %w", partitionID, err)
			}

			var counter int
			for i, item := range jobs {
				itemID := itemIDs[i]
				if item == "" {
					l.Error("missing queue item in concurrency queue",
						"index_partition", partitionID,
						"concurrency_queue_key", queueKey,
						"item_id", itemID,
					)

					// Drop item reference to prevent spinning on this item
					err := client.Do(ctx, client.B().Zrem().Key(queueKey).Member(itemID).Build()).Error()
					if err != nil {
						resultErr = multierror.Append(resultErr, fmt.Errorf("error removing missing item '%s' from concurrency queue '%s': %w", itemID, partitionID, err))
					}
					continue
				}

				qi := osqueue.QueueItem{}
				if err := json.Unmarshal([]byte(item), &qi); err != nil {
					resultErr = multierror.Append(resultErr, fmt.Errorf("error unmarshalling job '%s': %w", item, err))
					continue
				}
				if err := q.Requeue(ctx, qi, q.Clock.Now()); err != nil {
					resultErr = multierror.Append(resultErr, fmt.Errorf("error requeueing job '%s': %w", item, err))
					continue
				}
				l.Debug("scavenger requeued queue item",
					"id", qi.ID,
					"kind", qi.Data.Kind,
					"run_id", qi.Data.Identifier.RunID,
				)
				counter++
			}

			return len(itemIDs), counter, nil
		}

		keyPartitionScavengerIndex := kg.PartitionScavengerIndex(partitionID)

		peekedFromIndex, scavengedFromIndex, err := scavengePartition(keyPartitionScavengerIndex, "partition_index")
		if err != nil {
			resultErr = multierror.Append(resultErr, fmt.Errorf("could not scavenge from scavenger index: %w", err))
			continue
		}
		counter += scavengedFromIndex
		metrics.IncrQueueScavengerRequeuedItemsCounter(ctx, int64(peekedFromIndex), metrics.CounterOpt{
			PkgName: pkgName,
			Tags: map[string]any{
				"kind": "partition_index",
			},
		})

		if peekedFromIndex < osqueue.ScavengeConcurrencyQueuePeekSize {
			// Atomically attempt to drop empty pointer if we've processed all items
			err := q.dropPartitionPointerIfEmpty(
				ctx,
				kg.ConcurrencyIndex(),
				keyPartitionScavengerIndex,
				partitionID,
			)
			if err != nil {
				resultErr = multierror.Append(resultErr, fmt.Errorf("error dropping potentially empty pointer for partition %q: %w", partitionID, err))
			}
			continue
		}
	}

	return counter, resultErr
}
