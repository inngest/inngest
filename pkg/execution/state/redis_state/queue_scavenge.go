package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	mrand "math/rand"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/redis/rueidis"
)

const (
	ScavengePeekSize                 = 100
	ScavengeConcurrencyQueuePeekSize = 100
)

func (q *queue) randomScavengeOffset(seed int64, count int64, limit int) int64 {
	// only apply random offset if there are more total items to scavenge than the limit
	if count > int64(limit) {
		r := mrand.New(mrand.NewSource(seed))

		// the result of count-limit must be greater than 0 as we have already checked count > limit
		// we increase the argument by 1 to make the highest possible index accessible
		// example: for count = 9, limit = 3, we want to access indices 0 through 6, not 0 through 5
		return r.Int63n(count - int64(limit) + 1)
	}

	return 0
}

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
		Limit(q.randomScavengeOffset(q.Clock.Now().UnixMilli(), count, limit), int64(limit)).
		Build()

	// NOTE: Received keys can be legacy (workflow IDs or system/internal queue names) or new (full Redis keys)
	pKeys, err := client.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return 0, fmt.Errorf("error scavenging for lost items: %w", err)
	}

	counter := 0

	// Each of the items is a concurrency queue with lost items.
	var resultErr error
	for _, partition := range pKeys {
		// NOTE: If this is not a fully-qualified Redis key to a concurrency queue,
		// assume that this is an old queueName or function ID
		// This is for backwards compatibility with the previous concurrency index item format
		queueKey := partition
		if !isKeyConcurrencyPointerItem(partition) {
			queueKey = kg.Concurrency("p", partition)
		}

		// Drop key queues from concurrency pointer - these should not be in here
		if strings.HasPrefix(queueKey, "{q:v1}:concurrency:custom:") {
			err := client.Do(ctx, client.B().Zrem().Key(kg.ConcurrencyIndex()).Member(partition).Build()).Error()
			if err != nil {
				resultErr = multierror.Append(resultErr, fmt.Errorf("error removing key queue '%s' from concurrency pointer: %w", partition, err))
			}
			continue
		}

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
				Limit(0, ScavengeConcurrencyQueuePeekSize).
				Build()
			itemIDs, err := client.Do(ctx, cmd).AsStrSlice()
			if err != nil && err != rueidis.Nil {
				return 0, 0, fmt.Errorf("error querying partition concurrency queue '%s' during scavenge: %w", partition, err)
			}
			if len(itemIDs) == 0 {
				return 0, 0, nil
			}

			// Fetch the queue item, then requeue.
			cmd = client.B().Hmget().Key(kg.QueueItem()).Field(itemIDs...).Build()
			jobs, err := client.Do(ctx, cmd).AsStrSlice()
			if err != nil && err != rueidis.Nil {
				return 0, 0, fmt.Errorf("error fetching jobs for concurrency queue '%s' during scavenge: %w", partition, err)
			}

			var counter int
			for i, item := range jobs {
				itemID := itemIDs[i]
				if item == "" {
					l.Error("missing queue item in concurrency queue",
						"index_partition", partition,
						"concurrency_queue_key", queueKey,
						"item_id", itemID,
					)

					// Drop item reference to prevent spinning on this item
					err := client.Do(ctx, client.B().Zrem().Key(queueKey).Member(itemID).Build()).Error()
					if err != nil {
						resultErr = multierror.Append(resultErr, fmt.Errorf("error removing missing item '%s' from concurrency queue '%s': %w", itemID, partition, err))
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
				counter++
			}

			return len(itemIDs), counter, nil
		}

		peekedFromIndex, scavengedFromIndex, err := scavengePartition(kg.PartitionScavengerIndex(partition), "partition_index")
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

		if peekedFromIndex < ScavengeConcurrencyQueuePeekSize {
			// Atomically attempt to drop empty pointer if we've processed all items
			err := q.dropPartitionPointerIfEmpty(
				ctx,
				kg.ConcurrencyIndex(),
				queueKey,
				partition,
			)
			if err != nil {
				resultErr = multierror.Append(resultErr, fmt.Errorf("error dropping potentially empty pointer %q for partition %q: %w", partition, queueKey, err))
			}
			continue
		}
	}

	return counter, resultErr
}
