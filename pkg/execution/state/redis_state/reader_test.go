package redis_state

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestItemsByPartitionOnEmptyPartition(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	t.Run("test empty partition", func(t *testing.T) {
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return true
			}),
			osqueue.WithClock(clock),
		)

		_, err := q.ItemsByPartition(ctx, shard, "i-dont-exist", time.Time{}, clock.Now().Add(time.Minute),
			osqueue.WithQueueItemIterBatchSize(150),
		)
		require.Error(t, err)
		require.True(t, errors.Is(err, rueidis.Nil))
	})
}

func TestItemsByPartition(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	testcases := []struct {
		name             string
		num              int
		interval         time.Duration
		from             time.Time
		until            time.Time
		expectedItems    int
		keyQueuesEnabled bool
		batchSize        int64
	}{
		{
			name:          "retrieve items in one fetch",
			num:           500,
			from:          time.Time{},
			until:         clock.Now().Add(time.Minute),
			expectedItems: 500,
		},
		{
			name:          "with interval",
			num:           500,
			from:          time.Time{},
			until:         clock.Now().Add(time.Minute),
			interval:      -1 * time.Second,
			expectedItems: 500,
		},
		{
			name:          "with out of range interval",
			num:           10,
			from:          clock.Now(),
			until:         clock.Now().Add(7 * time.Second).Truncate(time.Second),
			interval:      time.Second,
			expectedItems: 7,
		},
		{
			name:          "with batch size",
			num:           500,
			from:          time.Time{},
			until:         clock.Now().Add(10 * time.Second).Truncate(time.Second),
			interval:      10 * time.Millisecond,
			expectedItems: 500,
			batchSize:     150,
		},
		// With key queues
		{
			name:             "kq - retrieve items in one fetch",
			num:              500,
			from:             clock.Now(),
			until:            clock.Now().Add(time.Minute),
			expectedItems:    500,
			keyQueuesEnabled: true,
		},
		{
			name:             "kq - with interval",
			num:              500,
			from:             time.Time{},
			until:            clock.Now().Add(time.Minute),
			interval:         10 * time.Millisecond,
			expectedItems:    500,
			keyQueuesEnabled: true,
		},
		{
			name:             "kq - with out of range interval",
			num:              10,
			from:             clock.Now(),
			until:            clock.Now().Add(7 * time.Second).Truncate(time.Second),
			interval:         time.Second,
			expectedItems:    7,
			keyQueuesEnabled: true,
		},
		{
			name:             "kq - with batch size",
			num:              500,
			from:             time.Time{},
			until:            clock.Now().Add(10 * time.Second).Truncate(time.Second),
			interval:         -10 * time.Millisecond,
			expectedItems:    500,
			batchSize:        200,
			keyQueuesEnabled: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r.FlushAll()

			q, shard := newQueue(
				t, rc,
				osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
					return tc.keyQueuesEnabled
				}),
				osqueue.WithClock(clock),
			)

			for i := range tc.num {
				at := clock.Now()
				if !tc.from.IsZero() {
					at = tc.from
				}
				at = at.Add(time.Duration(i) * tc.interval)

				item := osqueue.QueueItem{
					ID:          fmt.Sprintf("test%d", i),
					FunctionID:  fnID,
					WorkspaceID: wsID,
					Data: osqueue.Item{
						WorkspaceID: wsID,
						Kind:        osqueue.KindEdge,
						Identifier: state.Identifier{
							AccountID:       acctId,
							WorkspaceID:     wsID,
							WorkflowID:      fnID,
							WorkflowVersion: 1,
						},
					},
				}

				_, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
				require.NoError(t, err)
			}

			items, err := q.ItemsByPartition(ctx, shard, fnID.String(), tc.from, tc.until,
				osqueue.WithQueueItemIterBatchSize(tc.batchSize),
			)
			require.NoError(t, err)

			var count int
			for range items {
				count++
			}

			require.Equal(t, tc.expectedItems, count)
		})
	}
}

func TestItemsByPartitionWithSystemQueue(t *testing.T) {
	_, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	acctID, wsID := uuid.New(), uuid.New()

	q, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return false
		}),
		osqueue.WithClock(clock),
	)

	num := 5000

	systemQueueName := "a-system-queue"

	for i := range num {
		at := clock.Now().Add(time.Duration(i) * time.Millisecond)

		item := osqueue.QueueItem{
			ID:          fmt.Sprintf("test%d", i),
			QueueName:   &systemQueueName,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				QueueName:   &systemQueueName,
				Identifier: state.Identifier{
					AccountID:       acctID,
					WorkspaceID:     wsID,
					WorkflowVersion: 1,
				},
			},
		}

		_, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
		require.NoError(t, err)
	}

	items, err := q.ItemsByPartition(ctx, shard, systemQueueName, time.Time{}, clock.Now().Add(1*time.Hour),
		osqueue.WithQueueItemIterBatchSize(100),
		osqueue.WithQueueItemIterEnableBacklog(false),
	)
	require.NoError(t, err)

	var count int
	for range items {
		count++
	}

	require.Equal(t, num, count)
}

// leaseQueueItem modifies a queue item in Redis to simulate it being leased.
// It reads the item from the hash, sets LeaseID to a future ULID, and writes it back.
func leaseQueueItem(t *testing.T, rc rueidis.Client, kg QueueKeyGenerator, itemID string, leaseExpiry time.Time) {
	t.Helper()
	ctx := context.Background()

	hashKey := kg.QueueItem()

	// Read the current item
	cmd := rc.B().Hget().Key(hashKey).Field(itemID).Build()
	byt, err := rc.Do(ctx, cmd).AsBytes()
	require.NoError(t, err, "failed to read queue item %s", itemID)

	var qi osqueue.QueueItem
	err = json.Unmarshal(byt, &qi)
	require.NoError(t, err, "failed to unmarshal queue item %s", itemID)

	// Set LeaseID to a ULID with a timestamp in the future
	leaseID, err := ulid.New(ulid.Timestamp(leaseExpiry), rand.Reader)
	require.NoError(t, err)
	qi.LeaseID = &leaseID

	// Write back
	updated, err := json.Marshal(qi)
	require.NoError(t, err)

	setCmd := rc.B().Hset().Key(hashKey).FieldValue().FieldValue(itemID, string(updated)).Build()
	err = rc.Do(ctx, setCmd).Error()
	require.NoError(t, err, "failed to update queue item %s", itemID)
}

func TestItemsByPartitionLeasedItems(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	enqueueItems := func(t *testing.T, shard RedisQueueShard, n int, prefix string, atFn func(i int) time.Time) []string {
		t.Helper()
		ids := make([]string, 0, n)
		for i := range n {
			item := osqueue.QueueItem{
				ID:          fmt.Sprintf("%s-%d", prefix, i),
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindEdge,
					Identifier: state.Identifier{
						AccountID:       acctId,
						WorkspaceID:     wsID,
						WorkflowID:      fnID,
						WorkflowVersion: 1,
					},
				},
			}
			enqueued, err := shard.EnqueueItem(ctx, item, atFn(i), osqueue.EnqueueOpts{})
			require.NoError(t, err)
			ids = append(ids, enqueued.ID)
		}
		return ids
	}

	countIter := func(items func(yield func(*osqueue.QueueItem) bool)) int {
		var count int
		for range items {
			count++
		}
		return count
	}

	t.Run("should skip leased items but continue iterating remaining items", func(t *testing.T) {
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg

		// Enqueue 10 items spread across 10ms
		ids := enqueueItems(t, shard, 10, "leased", func(i int) time.Time {
			return clock.Now().Add(time.Duration(i) * time.Millisecond)
		})

		// Lease ALL items
		leaseExpiry := clock.Now().Add(10 * time.Minute)
		for _, id := range ids {
			leaseQueueItem(t, rc, kg, id, leaseExpiry)
		}

		items, err := q.ItemsByPartition(ctx, shard, fnID.String(), time.Time{}, clock.Now().Add(time.Minute),
			osqueue.WithQueueItemIterBatchSize(100),
			osqueue.WithQueueItemIterEnableBacklog(false),
		)
		require.NoError(t, err)

		// All items are leased so none should be yielded, but the iterator should
		// NOT exit early — it should recognize that peek returned data (all leased)
		// and that there may be more items beyond this batch. With all items fitting
		// in a single batch and all leased, returning 0 is acceptable.
		count := countIter(items)
		require.Equal(t, 0, count, "all leased items should be skipped")
	})

	t.Run("should iterate past a fully-leased first batch to reach unleased items", func(t *testing.T) {
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg

		batchSize := int64(10)
		leasedCount := int(batchSize) // first batch entirely leased
		unleasedCount := 20           // items beyond the first batch

		ids := enqueueItems(t, shard, leasedCount+unleasedCount, "mixed", func(i int) time.Time {
			return clock.Now().Add(time.Duration(i) * time.Millisecond)
		})

		// Lease only the first batch worth of items
		leaseExpiry := clock.Now().Add(10 * time.Minute)
		for _, id := range ids[:leasedCount] {
			leaseQueueItem(t, rc, kg, id, leaseExpiry)
		}

		items, err := q.ItemsByPartition(ctx, shard, fnID.String(), time.Time{}, clock.Now().Add(time.Minute),
			osqueue.WithQueueItemIterBatchSize(batchSize),
			osqueue.WithQueueItemIterEnableBacklog(false),
		)
		require.NoError(t, err)

		// The iterator must continue past the fully-leased first batch and
		// return all 20 unleased items from subsequent batches.
		count := countIter(items)
		require.Equal(t, unleasedCount, count,
			"iterator should continue past fully-leased batch and return all unleased items")
	})

	t.Run("items at different milliseconds with leased entries across batch boundary", func(t *testing.T) {
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg

		batchSize := int64(5)

		// Enqueue 10 items at distinct milliseconds
		ids := enqueueItems(t, shard, 10, "diff-ms", func(i int) time.Time {
			return clock.Now().Add(time.Duration(i) * time.Millisecond)
		})

		// Lease items 2 and 3 (in the middle of the first batch)
		leaseExpiry := clock.Now().Add(10 * time.Minute)
		leaseQueueItem(t, rc, kg, ids[2], leaseExpiry)
		leaseQueueItem(t, rc, kg, ids[3], leaseExpiry)

		items, err := q.ItemsByPartition(ctx, shard, fnID.String(), time.Time{}, clock.Now().Add(time.Minute),
			osqueue.WithQueueItemIterBatchSize(batchSize),
			osqueue.WithQueueItemIterEnableBacklog(false),
		)
		require.NoError(t, err)

		// 10 items total, 2 leased → 8 should be returned.
		// The iterator should advance past the first batch and pick up items from
		// the second batch even though some items in the first batch were leased.
		count := countIter(items)
		require.Equal(t, 8, count,
			"all unleased items across batches should be returned")
	})
}

// deleteQueueItemFromHash removes a queue item from the hash map only,
// leaving its entry in the sorted set (simulating a race where the item
// was dequeued between ZRANGEBYSCORE and HMGET).
func deleteQueueItemFromHash(t *testing.T, rc rueidis.Client, kg QueueKeyGenerator, itemID string) {
	t.Helper()
	ctx := context.Background()
	cmd := rc.B().Hdel().Key(kg.QueueItem()).Field(itemID).Build()
	err := rc.Do(ctx, cmd).Error()
	require.NoError(t, err, "failed to delete queue item %s from hash", itemID)
}

func TestItemsByPartitionMissingHashItems(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	enqueueItems := func(t *testing.T, shard RedisQueueShard, n int, prefix string, atFn func(i int) time.Time) []string {
		t.Helper()
		ids := make([]string, 0, n)
		for i := range n {
			item := osqueue.QueueItem{
				ID:          fmt.Sprintf("%s-%d", prefix, i),
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindEdge,
					Identifier: state.Identifier{
						AccountID:       acctId,
						WorkspaceID:     wsID,
						WorkflowID:      fnID,
						WorkflowVersion: 1,
					},
				},
			}
			enqueued, err := shard.EnqueueItem(ctx, item, atFn(i), osqueue.EnqueueOpts{})
			require.NoError(t, err)
			ids = append(ids, enqueued.ID)
		}
		return ids
	}

	countIter := func(items func(yield func(*osqueue.QueueItem) bool)) int {
		var count int
		for range items {
			count++
		}
		return count
	}

	t.Run("should iterate past items missing from hash to reach remaining items", func(t *testing.T) {
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg

		batchSize := int64(10)
		deletedCount := int(batchSize)
		remainingCount := 15

		ids := enqueueItems(t, shard, deletedCount+remainingCount, "missing-hash", func(i int) time.Time {
			return clock.Now().Add(time.Duration(i) * time.Millisecond)
		})

		// Delete the first batch of items from the hash only, leaving them in the sorted set.
		// This simulates items being dequeued/completed between the ZRANGEBYSCORE and HMGET
		// calls inside peek's Lua script, or items that were cleaned up externally.
		for _, id := range ids[:deletedCount] {
			deleteQueueItemFromHash(t, rc, kg, id)
		}

		items, err := q.ItemsByPartition(ctx, shard, fnID.String(), time.Time{}, clock.Now().Add(time.Minute),
			osqueue.WithQueueItemIterBatchSize(batchSize),
			osqueue.WithQueueItemIterEnableBacklog(false),
		)
		require.NoError(t, err)

		// The first peek returns `batchSize` items from the sorted set, but all are
		// missing from the hash. peek cleans them up (removes from zset) and returns
		// an empty slice. The iterator must NOT exit early — it should recognize that
		// peek found items (even though they were missing) and continue to the next batch.
		count := countIter(items)
		require.Equal(t, remainingCount, count,
			"iterator should continue past missing-hash items and return all remaining items")
	})

	t.Run("should handle all items missing from hash without panicking", func(t *testing.T) {
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg

		ids := enqueueItems(t, shard, 10, "all-missing", func(i int) time.Time {
			return clock.Now().Add(time.Duration(i) * time.Millisecond)
		})

		// Delete ALL items from the hash
		for _, id := range ids {
			deleteQueueItemFromHash(t, rc, kg, id)
		}

		items, err := q.ItemsByPartition(ctx, shard, fnID.String(), time.Time{}, clock.Now().Add(time.Minute),
			osqueue.WithQueueItemIterBatchSize(100),
			osqueue.WithQueueItemIterEnableBacklog(false),
		)
		require.NoError(t, err)

		// All items are missing from hash. peek cleans them up from the zset and
		// returns nothing. After cleanup, the zset is empty, so the next peek also
		// returns nothing. The iterator should gracefully return 0 items.
		count := countIter(items)
		require.Equal(t, 0, count, "all items missing from hash should result in 0 yielded items")
	})
}

func TestItemsByPartitionScoreParsing(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	enqueueItems := func(t *testing.T, shard RedisQueueShard, n int, prefix string, atFn func(i int) time.Time) []string {
		t.Helper()
		ids := make([]string, 0, n)
		for i := range n {
			item := osqueue.QueueItem{
				ID:          fmt.Sprintf("%s-%d", prefix, i),
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindEdge,
					Identifier: state.Identifier{
						AccountID:       acctId,
						WorkspaceID:     wsID,
						WorkflowID:      fnID,
						WorkflowVersion: 1,
					},
				},
			}
			enqueued, err := shard.EnqueueItem(ctx, item, atFn(i), osqueue.EnqueueOpts{})
			require.NoError(t, err)
			ids = append(ids, enqueued.ID)
		}
		return ids
	}

	countIter := func(items func(yield func(*osqueue.QueueItem) bool)) int {
		var count int
		for range items {
			count++
		}
		return count
	}

	t.Run("iterator terminates when leased items share the same millisecond", func(t *testing.T) {
		// This test verifies that the LastScore from peek is correctly parsed and
		// used to advance the cursor. If the score were silently parsed as 0
		// (e.g. due to float-string format like "1711252800000.0"), the cursor
		// would regress to epoch+1ms and the iterator would loop forever.
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg

		batchSize := int64(5)
		leasedCount := int(batchSize)
		unleasedCount := 10

		// Enqueue leased items all at the SAME millisecond, followed by unleased
		// items at later milliseconds. The leased batch fills an entire peek, so
		// the iterator must parse LastScore correctly to advance past them.
		leasedTime := clock.Now().Add(time.Second)
		ids := enqueueItems(t, shard, leasedCount+unleasedCount, "score-parse", func(i int) time.Time {
			if i < leasedCount {
				return leasedTime
			}
			return leasedTime.Add(time.Duration(i-leasedCount+1) * time.Millisecond)
		})

		leaseExpiry := clock.Now().Add(10 * time.Minute)
		for _, id := range ids[:leasedCount] {
			leaseQueueItem(t, rc, kg, id, leaseExpiry)
		}

		// Use a channel + timeout to detect an infinite loop.
		done := make(chan int, 1)
		go func() {
			items, err := q.ItemsByPartition(ctx, shard, fnID.String(), time.Time{}, clock.Now().Add(time.Hour),
				osqueue.WithQueueItemIterBatchSize(batchSize),
				osqueue.WithQueueItemIterEnableBacklog(false),
			)
			if err != nil {
				done <- -1
				return
			}
			done <- countIter(items)
		}()

		select {
		case count := <-done:
			require.Equal(t, unleasedCount, count,
				"should return all unleased items after advancing past leased batch")
		case <-time.After(10 * time.Second):
			t.Fatal("iterator did not terminate — likely infinite loop due to score parsing failure")
		}
	})

	t.Run("iterator terminates when all leased items have score zero", func(t *testing.T) {
		// If items in the sorted set have score 0 and are all leased, LastScore
		// will be 0. The iterator must break instead of regressing to epoch+1ms.
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg

		// Enqueue items at a normal time, then overwrite their sorted set scores to 0.
		ids := enqueueItems(t, shard, 5, "zero-score", func(i int) time.Time {
			return clock.Now().Add(time.Duration(i) * time.Millisecond)
		})

		// Lease all items
		leaseExpiry := clock.Now().Add(10 * time.Minute)
		for _, id := range ids {
			leaseQueueItem(t, rc, kg, id, leaseExpiry)
		}

		// Overwrite sorted set scores to 0, simulating a degenerate case where
		// LastScore would be parsed as 0.
		zsetKey := kg.PartitionQueueSet(enums.PartitionTypeDefault, fnID.String(), "")
		for _, id := range ids {
			_, err := r.ZAdd(zsetKey, 0, id)
			require.NoError(t, err)
		}

		done := make(chan int, 1)
		go func() {
			items, err := q.ItemsByPartition(ctx, shard, fnID.String(), time.Time{}, clock.Now().Add(time.Hour),
				osqueue.WithQueueItemIterBatchSize(100),
				osqueue.WithQueueItemIterEnableBacklog(false),
			)
			if err != nil {
				done <- -1
				return
			}
			done <- countIter(items)
		}()

		select {
		case count := <-done:
			// All items are leased and have score 0 — the iterator should break
			// gracefully via the lastScore <= 0 guard.
			require.Equal(t, 0, count,
				"should return 0 items and terminate when all leased items have score 0")
		case <-time.After(10 * time.Second):
			t.Fatal("iterator did not terminate — likely infinite loop due to lastScore == 0 regression")
		}
	})
}

// setQueueItemAtMS modifies a queue item's AtMS in the hash without touching
// its sorted set score.  This simulates the production scenario where items
// are retried/rescheduled and AtMS drifts far ahead of the original score.
func setQueueItemAtMS(t *testing.T, rc rueidis.Client, kg QueueKeyGenerator, itemID string, newAtMS int64) {
	t.Helper()
	ctx := context.Background()

	hashKey := kg.QueueItem()

	cmd := rc.B().Hget().Key(hashKey).Field(itemID).Build()
	byt, err := rc.Do(ctx, cmd).AsBytes()
	require.NoError(t, err, "failed to read queue item %s", itemID)

	var qi osqueue.QueueItem
	err = json.Unmarshal(byt, &qi)
	require.NoError(t, err, "failed to unmarshal queue item %s", itemID)

	qi.AtMS = newAtMS

	updated, err := json.Marshal(qi)
	require.NoError(t, err)

	setCmd := rc.B().Hset().Key(hashKey).FieldValue().FieldValue(itemID, string(updated)).Build()
	err = rc.Do(ctx, setCmd).Error()
	require.NoError(t, err, "failed to update queue item %s", itemID)
}

func TestItemsByPartitionAtMSDivergence(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	enqueueItems := func(t *testing.T, shard RedisQueueShard, n int, prefix string, atFn func(i int) time.Time) []string {
		t.Helper()
		ids := make([]string, 0, n)
		for i := range n {
			item := osqueue.QueueItem{
				ID:          fmt.Sprintf("%s-%d", prefix, i),
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindEdge,
					Identifier: state.Identifier{
						AccountID:       acctId,
						WorkspaceID:     wsID,
						WorkflowID:      fnID,
						WorkflowVersion: 1,
					},
				},
			}
			enqueued, err := shard.EnqueueItem(ctx, item, atFn(i), osqueue.EnqueueOpts{})
			require.NoError(t, err)
			ids = append(ids, enqueued.ID)
		}
		return ids
	}

	countIter := func(items func(yield func(*osqueue.QueueItem) bool)) int {
		var count int
		for range items {
			count++
		}
		return count
	}

	t.Run("should iterate all items when AtMS diverges ahead of sorted set score", func(t *testing.T) {
		// This reproduces the production bug where items' AtMS was ~184 days
		// ahead of their sorted set scores (due to retries/rescheduling),
		// causing the iterator cursor to jump past all remaining items.
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg

		totalItems := 30
		batchSize := int64(10)

		// Enqueue items 1ms apart — sorted set scores will be
		// clock.Now(), clock.Now()+1ms, clock.Now()+2ms, ...
		ids := enqueueItems(t, shard, totalItems, "diverge", func(i int) time.Time {
			return clock.Now().Add(time.Duration(i) * time.Millisecond)
		})

		// Now update every item's AtMS to be far in the future (simulating
		// retries that bumped AtMS but didn't change the sorted set score).
		futureAtMS := clock.Now().Add(180 * 24 * time.Hour).UnixMilli()
		for _, id := range ids {
			setQueueItemAtMS(t, rc, kg, id, futureAtMS)
		}

		done := make(chan int, 1)
		go func() {
			items, err := q.ItemsByPartition(ctx, shard, fnID.String(), time.Time{}, clock.Now().Add(time.Hour),
				osqueue.WithQueueItemIterBatchSize(batchSize),
				osqueue.WithQueueItemIterEnableBacklog(false),
			)
			if err != nil {
				done <- -1
				return
			}
			done <- countIter(items)
		}()

		select {
		case count := <-done:
			require.Equal(t, totalItems, count,
				"iterator must return all items even when AtMS >> sorted set score")
		case <-time.After(30 * time.Second):
			t.Fatal("iterator did not terminate — likely stuck due to AtMS/score divergence")
		}
	})

	t.Run("should iterate all items across multiple batches with varying AtMS", func(t *testing.T) {
		// Items in early batches have AtMS far in the future, items in
		// later batches have normal AtMS.  The old code would skip the
		// later batches because the cursor jumped ahead.
		r.FlushAll()

		q, shard := newQueue(
			t, rc,
			osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
				return false
			}),
			osqueue.WithClock(clock),
		)
		kg := shard.Client().kg

		totalItems := 25
		batchSize := int64(5)

		ids := enqueueItems(t, shard, totalItems, "mixed", func(i int) time.Time {
			return clock.Now().Add(time.Duration(i) * time.Millisecond)
		})

		// Set first batch's AtMS far in the future; leave the rest alone.
		futureAtMS := clock.Now().Add(365 * 24 * time.Hour).UnixMilli()
		for _, id := range ids[:5] {
			setQueueItemAtMS(t, rc, kg, id, futureAtMS)
		}

		done := make(chan int, 1)
		go func() {
			items, err := q.ItemsByPartition(ctx, shard, fnID.String(), time.Time{}, clock.Now().Add(time.Hour),
				osqueue.WithQueueItemIterBatchSize(batchSize),
				osqueue.WithQueueItemIterEnableBacklog(false),
			)
			if err != nil {
				done <- -1
				return
			}
			done <- countIter(items)
		}()

		select {
		case count := <-done:
			require.Equal(t, totalItems, count,
				"iterator must return all items even when first batch has divergent AtMS")
		case <-time.After(30 * time.Second):
			t.Fatal("iterator did not terminate — likely stuck due to AtMS/score divergence")
		}
	})
}

func TestItemsByBacklog(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Minute))

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	q, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return true
		}),
		osqueue.WithClock(clock),
	)
	kg := shard.Client().kg

	testcases := []struct {
		name          string
		num           int
		expectedItems int
		interval      time.Duration
		from          time.Time
		until         time.Time
		batchSize     int64
	}{
		{
			name:          "retrieve items in one fetch",
			num:           10,
			from:          clock.Now(),
			until:         clock.Now().Add(time.Minute),
			expectedItems: 10,
		},
		{
			name:          "with interval",
			num:           10,
			from:          clock.Now(),
			until:         clock.Now().Add(time.Minute),
			interval:      time.Second,
			expectedItems: 10,
		},
		{
			name:          "with out of range interval",
			num:           10,
			from:          clock.Now(),
			until:         clock.Now().Add(7 * time.Second).Truncate(time.Second),
			interval:      time.Second,
			// 10 items enqueued at 0s–9s; the inclusive [from, until] window
			// covers 0s–7s, so 8 items (at offsets 0–7) are returned.
			expectedItems: 8,
		},
		{
			name:          "with batch size",
			num:           10,
			from:          clock.Now(),
			until:         clock.Now().Add(10 * time.Second).Truncate(time.Second),
			interval:      10 * time.Millisecond,
			expectedItems: 10,
			batchSize:     2,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r.FlushAll()

			for i := range tc.num {
				at := tc.from.Add(time.Duration(i) * tc.interval)

				item := osqueue.QueueItem{
					ID:          fmt.Sprintf("test%d", i),
					FunctionID:  fnID,
					WorkspaceID: wsID,
					Data: osqueue.Item{
						WorkspaceID: wsID,
						Kind:        osqueue.KindEdge,
						Identifier: state.Identifier{
							AccountID:       acctId,
							WorkspaceID:     wsID,
							WorkflowID:      fnID,
							WorkflowVersion: 1,
						},
					},
				}

				_, err := shard.EnqueueItem(ctx, item, at, osqueue.EnqueueOpts{})
				require.NoError(t, err)
			}

			var backlogID string
			{
				mem, err := r.ZMembers(kg.ShadowPartitionSet(fnID.String()))
				require.NoError(t, err)
				require.Len(t, mem, 1)
				backlogID = mem[0]
			}
			require.NotEmpty(t, backlogID)

			items, err := q.ItemsByBacklog(ctx, shard, backlogID, tc.from, tc.until,
				osqueue.WithQueueItemIterBatchSize(tc.batchSize),
			)
			require.NoError(t, err)

			var count int
			for range items {
				count++
			}

			require.Equal(t, tc.expectedItems, count)
		})
	}
}

func TestItemsByBacklogZeroCursor(t *testing.T) {
	// This test verifies that ItemsByBacklog terminates correctly when
	// all items in the backlog sorted set have a score of 0 (epoch).
	// Before the fix, peekRes.Cursor would be 0 and backlogFrom would
	// never advance, causing an infinite loop.
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	q, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return true
		}),
		osqueue.WithClock(clock),
	)
	kg := shard.Client().kg

	t.Run("terminates when all backlog items have score zero", func(t *testing.T) {
		r.FlushAll()

		totalItems := 5
		for i := range totalItems {
			item := osqueue.QueueItem{
				ID:          fmt.Sprintf("zero-score-%d", i),
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindEdge,
					Identifier: state.Identifier{
						AccountID:       acctId,
						WorkspaceID:     wsID,
						WorkflowID:      fnID,
						WorkflowVersion: 1,
					},
				},
			}
			_, err := shard.EnqueueItem(ctx, item, clock.Now().Add(time.Duration(i)*time.Millisecond), osqueue.EnqueueOpts{})
			require.NoError(t, err)
		}

		// Find the backlog ID
		var backlogID string
		{
			mem, err := r.ZMembers(kg.ShadowPartitionSet(fnID.String()))
			require.NoError(t, err)
			require.Len(t, mem, 1)
			backlogID = mem[0]
		}
		require.NotEmpty(t, backlogID)

		// Override all sorted set scores to 0 (epoch) to trigger the bug
		backlogSetKey := kg.BacklogSet(backlogID)
		members, err := r.ZMembers(backlogSetKey)
		require.NoError(t, err)
		for _, m := range members {
			_, err := r.ZAdd(backlogSetKey, 0, m)
			require.NoError(t, err)
		}

		done := make(chan int, 1)
		go func() {
			items, err := q.ItemsByBacklog(ctx, shard, backlogID, time.Time{}, clock.Now().Add(time.Hour),
				osqueue.WithQueueItemIterBatchSize(100),
			)
			if err != nil {
				done <- -1
				return
			}
			var count int
			for range items {
				count++
			}
			done <- count
		}()

		select {
		case count := <-done:
			require.Equal(t, totalItems, count,
				"should return all items and terminate even when backlog scores are zero")
		case <-time.After(10 * time.Second):
			t.Fatal("ItemsByBacklog did not terminate — infinite loop when Cursor == 0 with items yielded")
		}
	})

	t.Run("terminates across multiple batches when backlog scores are zero", func(t *testing.T) {
		r.FlushAll()

		totalItems := 10
		batchSize := int64(3)
		for i := range totalItems {
			item := osqueue.QueueItem{
				ID:          fmt.Sprintf("zero-batch-%d", i),
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindEdge,
					Identifier: state.Identifier{
						AccountID:       acctId,
						WorkspaceID:     wsID,
						WorkflowID:      fnID,
						WorkflowVersion: 1,
					},
				},
			}
			_, err := shard.EnqueueItem(ctx, item, clock.Now().Add(time.Duration(i)*time.Millisecond), osqueue.EnqueueOpts{})
			require.NoError(t, err)
		}

		var backlogID string
		{
			mem, err := r.ZMembers(kg.ShadowPartitionSet(fnID.String()))
			require.NoError(t, err)
			require.Len(t, mem, 1)
			backlogID = mem[0]
		}
		require.NotEmpty(t, backlogID)

		// Override all sorted set scores to 0
		backlogSetKey := kg.BacklogSet(backlogID)
		members, err := r.ZMembers(backlogSetKey)
		require.NoError(t, err)
		for _, m := range members {
			_, err := r.ZAdd(backlogSetKey, 0, m)
			require.NoError(t, err)
		}

		done := make(chan int, 1)
		go func() {
			items, err := q.ItemsByBacklog(ctx, shard, backlogID, time.Time{}, clock.Now().Add(time.Hour),
				osqueue.WithQueueItemIterBatchSize(batchSize),
			)
			if err != nil {
				done <- -1
				return
			}
			var count int
			for range items {
				count++
			}
			done <- count
		}()

		select {
		case count := <-done:
			// With all scores at 0 and batching, the iterator must still terminate.
			// It should return items from the first batch and then stop (since cursor
			// can't advance past 0).
			require.Greater(t, count, 0,
				"should return at least some items when backlog scores are zero")
		case <-time.After(10 * time.Second):
			t.Fatal("ItemsByBacklog did not terminate — infinite loop when Cursor == 0 across batches")
		}
	})
}

func TestItemsByPartitionBacklogZeroCursor(t *testing.T) {
	// This test verifies that the backlog phase of ItemsByPartition terminates
	// correctly when all backlogs return Cursor == 0. Before the fix,
	// earliestCursor stayed 0, backlogFrom was never updated, and the
	// outer loop re-fetched the same items forever.
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	q, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return true
		}),
		osqueue.WithClock(clock),
	)
	kg := shard.Client().kg

	t.Run("terminates when all backlog cursors are zero", func(t *testing.T) {
		r.FlushAll()

		totalItems := 5
		for i := range totalItems {
			item := osqueue.QueueItem{
				ID:          fmt.Sprintf("pt-zero-%d", i),
				FunctionID:  fnID,
				WorkspaceID: wsID,
				Data: osqueue.Item{
					WorkspaceID: wsID,
					Kind:        osqueue.KindEdge,
					Identifier: state.Identifier{
						AccountID:       acctId,
						WorkspaceID:     wsID,
						WorkflowID:      fnID,
						WorkflowVersion: 1,
					},
				},
			}
			_, err := shard.EnqueueItem(ctx, item, clock.Now().Add(time.Duration(i)*time.Millisecond), osqueue.EnqueueOpts{})
			require.NoError(t, err)
		}

		// Find the backlog and override scores to 0
		mem, err := r.ZMembers(kg.ShadowPartitionSet(fnID.String()))
		require.NoError(t, err)
		require.NotEmpty(t, mem)

		for _, backlogID := range mem {
			backlogSetKey := kg.BacklogSet(backlogID)
			members, err := r.ZMembers(backlogSetKey)
			require.NoError(t, err)
			for _, m := range members {
				_, err := r.ZAdd(backlogSetKey, 0, m)
				require.NoError(t, err)
			}
		}

		done := make(chan int, 1)
		go func() {
			items, err := q.ItemsByPartition(ctx, shard, fnID.String(), time.Time{}, clock.Now().Add(time.Hour),
				osqueue.WithQueueItemIterBatchSize(100),
				osqueue.WithQueueItemIterEnableBacklog(true),
			)
			if err != nil {
				done <- -1
				return
			}
			var count int
			for range items {
				count++
			}
			done <- count
		}()

		select {
		case count := <-done:
			// Items should be yielded from the backlog phase (scores are 0 which is <= until).
			// The important thing is that the iterator terminates.
			require.Greater(t, count, 0,
				"should return items from backlogs and terminate even when all cursors are zero")
		case <-time.After(10 * time.Second):
			t.Fatal("ItemsByPartition backlog phase did not terminate — infinite loop when all Cursors == 0")
		}
	})
}

func TestQueueIterator(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()

	_, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return false // TODO need to add support for key queues
		}),
		osqueue.WithClock(clock),
	)

	acctId, wsID := uuid.New(), uuid.New()

	testcases := []struct {
		name       string
		partitions int
		items      int
	}{
		{
			name:       "one partition",
			partitions: 1,
			items:      100,
		},
		{
			name:       "multiple partitions",
			partitions: 100,
			items:      500,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r.FlushAll()

			// construct partition IDs
			partitions := make([]uuid.UUID, tc.partitions)
			for i := range tc.partitions {
				partitions[i] = uuid.New()
			}

			for i := range tc.items {
				size := len(partitions)
				fnID := partitions[i%size]

				item := osqueue.QueueItem{
					ID:          fmt.Sprintf("test%d", i),
					FunctionID:  fnID,
					WorkspaceID: wsID,
					Data: osqueue.Item{
						WorkspaceID: wsID,
						Kind:        osqueue.KindEdge,
						Identifier: state.Identifier{
							AccountID:       acctId,
							WorkspaceID:     wsID,
							WorkflowID:      fnID,
							WorkflowVersion: 1,
						},
					},
				}

				_, err := shard.EnqueueItem(ctx, item, clock.Now(), osqueue.EnqueueOpts{})
				require.NoError(t, err)
			}

			ptCnt, piCnt, err := shard.QueueIterator(ctx, QueueIteratorOpts{})
			require.NoError(t, err)

			require.EqualValues(t, tc.partitions, ptCnt)
			require.EqualValues(t, tc.items, piCnt)
		})
	}
}

func TestItemByID(t *testing.T) {
	_, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	q1, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return false
		}),
		osqueue.WithClock(clock),
	)

	enqueue := func(ctx context.Context, shard RedisQueueShard) (osqueue.QueueItem, error) {
		item := osqueue.QueueItem{
			ID:          ulid.MustNew(ulid.Now(), rand.Reader).String(),
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					AccountID:   acctId,
					WorkspaceID: wsID,
					WorkflowID:  fnID,
				},
			},
		}

		return shard.EnqueueItem(ctx, item, clock.Now(), osqueue.EnqueueOpts{})
	}

	t.Run("should be able to find the queue item", func(t *testing.T) {
		enqueued, err := enqueue(ctx, shard)
		require.NoError(t, err)

		res, err := q1.ItemByID(ctx, shard, enqueued.ID)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, enqueued.ID, res.ID)
	})

	t.Run("should return not found error if absent", func(t *testing.T) {
		_, err := enqueue(ctx, shard)
		require.NoError(t, err)

		res, err := q1.ItemByID(ctx, shard, "random")
		require.ErrorIs(t, err, osqueue.ErrQueueItemNotFound)
		require.Nil(t, res)
	})
}

func TestItemExists(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()
	clock := clockwork.NewFakeClock()
	acctId, fnID, wsID := uuid.New(), uuid.New(), uuid.New()

	q, shard := newQueue(
		t, rc,
		osqueue.WithAllowKeyQueues(func(ctx context.Context, acctID uuid.UUID, envID, fnID uuid.UUID) bool {
			return false
		}),
		osqueue.WithClock(clock),
	)

	enqueue := func(ctx context.Context, shard RedisQueueShard, jobID string) (osqueue.QueueItem, error) {
		item := osqueue.QueueItem{
			ID:          jobID,
			FunctionID:  fnID,
			WorkspaceID: wsID,
			Data: osqueue.Item{
				WorkspaceID: wsID,
				Kind:        osqueue.KindEdge,
				Identifier: state.Identifier{
					AccountID:   acctId,
					WorkspaceID: wsID,
					WorkflowID:  fnID,
				},
			},
		}

		return shard.EnqueueItem(ctx, item, clock.Now(), osqueue.EnqueueOpts{})
	}

	t.Run("should return true when item exists", func(t *testing.T) {
		r.FlushAll()

		jobID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		enqueued, err := enqueue(ctx, shard, jobID)
		require.NoError(t, err)

		exists, err := q.ItemExists(ctx, shard, enqueued.ID)
		require.NoError(t, err)
		require.True(t, exists, "item should exist")
	})

	t.Run("should return false when item does not exist", func(t *testing.T) {
		r.FlushAll()

		nonExistentJobID := ulid.MustNew(ulid.Now(), rand.Reader).String()

		exists, err := q.ItemExists(ctx, shard, nonExistentJobID)
		require.NoError(t, err)
		require.False(t, exists, "item should not exist")
	})

	t.Run("should return false after item is dequeued", func(t *testing.T) {
		r.FlushAll()

		jobID := ulid.MustNew(ulid.Now(), rand.Reader).String()
		enqueued, err := enqueue(ctx, shard, jobID)
		require.NoError(t, err)

		// Verify it exists
		exists, err := q.ItemExists(ctx, shard, enqueued.ID)
		require.NoError(t, err)
		require.True(t, exists)

		// Dequeue the item
		err = q.Dequeue(ctx, shard, enqueued)
		require.NoError(t, err)

		// Should no longer exist
		exists, err = q.ItemExists(ctx, shard, enqueued.ID)
		require.NoError(t, err)
		require.False(t, exists, "dequeued item should not exist")
	})
}
