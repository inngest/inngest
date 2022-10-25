package redisqueue

import (
	"context"
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
)

const (
	// SequentialPollTime indicates the duration at which the worker checks whether
	// it can become the sequential processor.
	SequentialPollTime = 5 * time.Second
	// SequentialLeaseTime is the time for each lease when becoming the sequential
	// processor.
	SequentialLeaseTime = 5 * SequentialPollTime

	// OverfetchMultiple represents the multiple number of items to overfetch when
	// reading from a queue to prevent backlogging and contention.
	//
	// TODO: Move into a bucketed approach in which we partition by workflow ID
	// and fetch workflow pointers, removing this.  Each worker claims a workflow
	// instead of individual tasks, leading to less contention.
	OverfetchMultiple = 2

	// LeaseDuration represents the length of time that an individual queue item
	// is leased for.
	LeaseDuration = 30 * time.Second
)

type Config struct{}

type item struct {
	ID    ulid.ULID  `json:"id"`
	Item  queue.Item `json:"item"`
	Lease *time.Time `json:"lease"`
}

type impl struct {
	r *redis.Client

	// handler represents the external function passed to Run() which processes
	// an individual task.
	handler func(context.Context, queue.Item) error

	// concurrency represents the number of concurrent items to be processed.
	concurrency int

	// buffer represents a shared communication channel for each concurrent processor
	// of an item.
	buffer chan string

	// sequential represents whether this worker reads items sequentially,
	// in the order to be processed, or randomly.  When running a cluster
	// of workers, one worker should read sequentially while the others
	// read randomly.  This minimizes the impact of long-running jobs causing
	// backlogs.
	//
	// As only one worker should run sequentially, this is leased by workers
	// via a key in Redis.
	//
	// This is the unix timestamp at which the worker became the sequential
	// processor, or 0 if the worker is not the sequential processor.
	sequential int64
}

func (q *impl) Enqueue(ctx context.Context, i queue.Item, at time.Time) error {
	// In order to maintain a concurrent redis queue, we need to decouple the
	// queue items from the zset.  This allows us to update items in the queue
	// without updating the zset, and to individually claim items from a the
	// queue transactionally.
	//
	// We may also enqueue and run many steps of a workflow at once.  This means
	// that each item needs its own ID, and we cannot rely on
	// identifier.IdempotencyKey to uniquely represent this queue item.
	item := item{
		// Create a new unique ID based off of the time that the item
		// should be processed at to represent this queue item.
		ID:   ulid.MustNew(uint64(at.UnixMilli()), crand.Reader),
		Item: i,
	}

	byt, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("error marshalling queue item: %w", err)
	}

	// TODO: Move this into a lua script as a single redis transaction.
	if err := q.r.Set(ctx, itemKey(i.Kind, item.ID.String()), byt, 0).Err(); err != nil {
		return fmt.Errorf("error setting queue item: %w", err)
	}
	if err := q.r.ZAdd(ctx, zsetKey(i.Kind), &redis.Z{
		Score:  float64(at.UnixMilli()),
		Member: item.ID.String(),
	}).Err(); err != nil {
		return fmt.Errorf("error adding item to queue set: %w", err)
	}
	return nil
}

func (q *impl) Run(ctx context.Context, f func(context.Context, queue.Item) error) error {
	if q.concurrency == 0 {
		q.concurrency = 1
	}

	for n := 0; n < q.concurrency; n++ {
		go q.concurrentlyProcess(ctx)
	}

	go q.claimSequentialProcessing(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			at := time.Now()

			keys, err := q.scan(ctx, queue.KindEdge)
			if err != nil {
				return err
			}

			// Push the item onto the queue.  The processor will check to see if
			// the item is claimed, and if not will claim and process.
			for _, k := range keys {
				q.buffer <- k
			}

			at.Add(10)
		}
	}

}

func (q *impl) concurrentlyProcess(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			logger.From(ctx).Debug().Msg("concurrent processor terminating")
			return
		case key := <-q.buffer:
			logger.From(ctx).Debug().Str("keyy", key).Msg("processing queue key")
			// TODO: Atomically read the item from Redis by key and check the lease:
			// - If the lease is nil or has expired (<= now()), claim the item
			//   by updating the lease to time.Now().Add(LeaseDuration)
			// - If the item is leased continue.
			//
			// Add a lua script to do this atomically.
			var i item
			_ = key

			done := false
			// renew the lease every LeaseDuration / 2 seconds, ensuring that the
			// job doesn't get retried if it takes longer than the original duration.
			go func() {
				for range time.Tick(LeaseDuration / 2) {
					if done {
						return
					}
				}
			}()

			if err := q.handler(ctx, i.Item); err != nil {
				// TODO: Re-enqueue the item as a retry.  Do this atomically
				// as a lua script.
			}

			// TODO: Remove the item from the queue entirely, including the zset
			// and the key.
			done = true
		}
	}
}

// claimSequentialProcessing attempts to claim sequential processing
func (q *impl) claimSequentialProcessing(ctx context.Context) {
	var t *time.Time

	for {
		t = nil
		if val := atomic.LoadInt64(&q.sequential); val != 0 {
			at := time.Unix(val, 0)
			t = &at
		}

		select {
		case <-ctx.Done():
			// Revoke the lease if we're the lease holder for being
			// the sequential worker.
			if t != nil {
				// TODO: Do this atomically in a lua script by
				// checking to see if we are still the lease
				// holder and destroying the lease if so.
			}
			return
		case <-time.After(SequentialPollTime):
			// Check to see if the lease for the sequential worker has expired
			// or is unset, and if so claim it.
			// TODO: Do this atomically in a lua script.
		}
	}
}

func (q *impl) scan(ctx context.Context, kind string) ([]string, error) {
	keys, err := q.r.ZRangeByScore(ctx, zsetKey(kind), &redis.ZRangeBy{
		Min:    "-inf",
		Max:    strconv.FormatInt(time.Now().UnixMilli(), 10),
		Offset: 0,
		// Overfetch, allowing us to skip any items previously claimed.
		Count: int64(q.concurrency) * OverfetchMultiple,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("error scanning queues: %w", err)
	}

	if val := atomic.LoadInt64(&q.sequential); val == 0 {
		// Shuffle the keys.
		rand.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
	}

	return nil, nil
}

func itemKey(kind, id string) string {
	return fmt.Sprintf("queue:item:%s:%s", kind, id)
}

func zsetKey(kind string) string {
	return fmt.Sprintf("queue:zset:%s", kind)
}
